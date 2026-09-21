package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesEmbeddedTerminalProbe(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewHandler(nil))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(body)
	for _, required := range []string{`id="terminal"`, `id="connection-status"`, `id="command-input"`, `/assets/app.js`, `/assets/xterm.css`} {
		if !strings.Contains(page, required) {
			t.Errorf("GET / body does not contain %q", required)
		}
	}
	if got := response.Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("Content-Security-Policy = %q, want self-only default", got)
	}
}

func TestHandlerServesOnlyKnownRoutes(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	for _, path := range []string{"/assets/app.js", "/assets/xterm.mjs", "/assets/addon-fit.mjs", "/assets/xterm.css"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, response.Code, http.StatusOK)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/not-a-route", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("GET unknown route status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestEmbeddedTerminalUsesMobileInstrumentShell(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	page := response.Body.String()

	for _, required := range []string{
		`class="brand-mark"`,
		`class="connection-chip"`,
		`id="project-context"`,
		`id="project-selector"`,
		`id="path-selector"`,
		`id="model-selector"`,
		`id="conversation-shell"`,
		`class="terminal-card"`,
		`id="terminal-size"`,
		`id="command-composer"`,
		`id="command-input"`,
		`id="command-counter"`,
		`id="send-command"`,
		`class="action-bar"`,
		`id="context-sheet"`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("mobile terminal shell does not contain %q", required)
		}
	}
}

func TestMobileShellOmitsRedundantTerminalKeyDeck(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	page := response.Body.String()

	for _, removed := range []string{
		`id="terminal-controls"`,
		`class="key-deck"`,
		`data-key="ctrl-c"`,
		`id="paste"`,
		`id="keyboard"`,
		`id="show-terminal-keys"`,
	} {
		if strings.Contains(page, removed) {
			t.Errorf("mobile terminal shell still contains redundant control %q", removed)
		}
	}
}

func TestComposerDismissesKeyboardAfterSending(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	script := response.Body.String()

	for _, required := range []string{
		`commandInput.blur();`,
		`document.querySelector(".terminal-card").scrollIntoView`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("composer send flow does not contain %q", required)
		}
	}
}

func TestRuntimeContextEndpoint(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithContext(nil, RuntimeContext{
		AgentID:          "codex",
		WorkingDirectory: "/Users/example/Developer/code-remote",
		TerminalID:       "prototype",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET context status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	var got RuntimeContext
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.AgentID != "codex" || got.WorkingDirectory != "/Users/example/Developer/code-remote" || got.WorkspaceName != "code-remote" || got.TerminalID != "prototype" {
		t.Fatalf("context = %#v, want configured runtime values and derived workspace name", got)
	}
	if got.Model != "native session" {
		t.Fatalf("context model = %q, want honest native-session marker", got.Model)
	}
}

func TestRuntimeContextEndpointRejectsMutation(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/context", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST context status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
