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

func TestEmbeddedTerminalMakesANSIWeightChangesVisible(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	script := response.Body.String()

	for _, required := range []string{`fontWeight: "400"`, `fontWeightBold: "700"`} {
		if !strings.Contains(script, required) {
			t.Errorf("terminal typography does not contain %q", required)
		}
	}
}

func TestEmbeddedTerminalSupportsTouchScrollingThroughTmux(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	scriptRequest := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	scriptResponse := httptest.NewRecorder()
	handler.ServeHTTP(scriptResponse, scriptRequest)
	script := scriptResponse.Body.String()

	for _, required := range []string{
		`function installTerminalTouchScrolling()`,
		`terminalElement.addEventListener("touchmove"`,
		`tmuxMouseWheelSequence`,
		`installTerminalTouchScrolling();`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("terminal touch scrolling does not contain %q", required)
		}
	}

	styleRequest := httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)
	styleResponse := httptest.NewRecorder()
	handler.ServeHTTP(styleResponse, styleRequest)
	style := styleResponse.Body.String()
	if !strings.Contains(style, `touch-action: none;`) {
		t.Error("terminal stage does not reserve vertical touch gestures for scrollback")
	}
}

func TestMobileShellOffersSessionManagement(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	pageRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	page := pageResponse.Body.String()
	for _, required := range []string{
		`data-sheet="sessions"`,
		`data-sheet-view="sessions"`,
		`id="session-list"`,
		`id="new-session"`,
		`id="new-session-nav"`,
		`id="active-session-name"`,
		`id="history-list"`,
		`id="history-count"`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("mobile session UI does not contain %q", required)
		}
	}

	scriptRequest := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	scriptResponse := httptest.NewRecorder()
	handler.ServeHTTP(scriptResponse, scriptRequest)
	script := scriptResponse.Body.String()
	for _, required := range []string{
		`fetch("/api/v1/terminals"`,
		`method: "POST"`,
		`connectToTerminal`,
		`renderSessionList`,
		`renderHistoryList`,
		`fetch("/api/v1/codex/threads"`,
		`working_directory=${encodeURIComponent(runtimeContext.working_directory)}`,
		`JSON.stringify({ working_directory: runtimeContext.working_directory })`,
		`localStorage.getItem("code-remote.active-terminal")`,
		`localStorage.setItem("code-remote.active-terminal"`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("mobile session behavior does not contain %q", required)
		}
	}
}

func TestMobileSessionListPrefersNativeAgentTitle(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	script := response.Body.String()

	for _, required := range []string{
		`context.title?.trim()`,
		`nativeTitle.endsWith(workspaceSuffix)`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("mobile session naming does not contain %q", required)
		}
	}
}

func TestMobileActionsSeparateAttachmentsFromNewSession(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, request)
	page := response.Body.String()

	for _, required := range []string{
		`id="composer-action"`,
		`id="add-image"`,
		`id="add-file"`,
		`id="new-session-nav"`,
		`data-sheet="sessions"`,
		`<small>对话</small>`,
		`<small>新对话</small>`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("mobile action hierarchy does not contain %q", required)
		}
	}
	for _, removed := range []string{
		`data-sheet="project"`,
		`id="new-session-action"`,
		`<small>切换项目</small>`,
	} {
		if strings.Contains(page, removed) {
			t.Errorf("mobile action hierarchy still contains %q", removed)
		}
	}
}

func TestMobilePathSheetBrowsesAndCreatesTerminalInSelectedDirectory(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	pageRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	page := pageResponse.Body.String()
	for _, required := range []string{
		`id="path-form"`,
		`id="path-input"`,
		`id="directory-list"`,
		`id="switch-path"`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("mobile path sheet does not contain %q", required)
		}
	}
	for _, removed := range []string{`目录浏览 API 尚未接入`, `路径浏览即将接入`} {
		if strings.Contains(page, removed) {
			t.Errorf("mobile path sheet still contains placeholder %q", removed)
		}
	}

	scriptRequest := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	scriptResponse := httptest.NewRecorder()
	handler.ServeHTTP(scriptResponse, scriptRequest)
	script := scriptResponse.Body.String()
	for _, required := range []string{
		`/api/v1/directories?path=`,
		`working_directory: workingDirectory`,
		`createSession(pathInput.value)`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("mobile path behavior does not contain %q", required)
		}
	}
}

func TestMobileModelSheetUsesCodexNativeSlashCommands(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	pageRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	page := pageResponse.Body.String()
	for _, required := range []string{
		`id="open-model-picker"`,
		`id="open-slash-menu"`,
		`data-native-command="/status"`,
		`data-native-command="/permissions"`,
		`data-native-command="/review"`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("mobile native command sheet does not contain %q", required)
		}
	}
	if strings.Contains(page, "模型切换尚未接入") {
		t.Error("mobile model sheet still says model switching is unavailable")
	}

	scriptRequest := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	scriptResponse := httptest.NewRecorder()
	handler.ServeHTTP(scriptResponse, scriptRequest)
	script := scriptResponse.Body.String()
	for _, required := range []string{
		`function sendNativeCommand(command, submit = true)`,
		`runtimeContext.agent_id !== "codex"`,
		`sendNativeCommand("/model")`,
		`sendNativeCommand("/", false)`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("mobile native command behavior does not contain %q", required)
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
