package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/huangxinxinyu/CodeRemote/internal/codex"
	"github.com/huangxinxinyu/CodeRemote/internal/terminal"
)

type fakeSessionCatalog struct {
	infos                  []terminal.Info
	sessions               map[string]TerminalSession
	created                terminal.Info
	createWorkingDirectory string
	createErr              error
	resumed                terminal.Info
	resumeID               string
	resumeWorkingDirectory string
}

func (catalog *fakeSessionCatalog) List() []terminal.Info {
	return append([]terminal.Info(nil), catalog.infos...)
}

func (catalog *fakeSessionCatalog) CreateInDirectory(_ context.Context, workingDirectory string, _, _ uint16) (terminal.Info, error) {
	catalog.createWorkingDirectory = workingDirectory
	if catalog.createErr != nil {
		return terminal.Info{}, catalog.createErr
	}
	catalog.infos = append(catalog.infos, catalog.created)
	return catalog.created, nil
}

func TestDirectoryAPIListsDirectChildrenAndRejectsMissingPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	child := filepath.Join(root, "Yuniverse")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	catalog := &fakeSessionCatalog{sessions: map[string]TerminalSession{}}
	server := httptest.NewServer(NewHandlerWithSessions(catalog, RuntimeContext{WorkingDirectory: root, TerminalID: "prototype"}))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/api/v1/directories?path=" + url.QueryEscape(root))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET directories status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var listing struct {
		Path        string `json:"path"`
		Directories []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"directories"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Directories) != 1 || listing.Directories[0].Name != "Yuniverse" {
		t.Fatalf("directory listing = %#v, want Yuniverse child", listing)
	}

	response, err = http.Get(server.URL + "/api/v1/directories?path=" + url.QueryEscape(filepath.Join(root, "missing")))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET missing directory status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestSessionAPIReportsUnavailableSubmittedDirectory(t *testing.T) {
	t.Parallel()

	catalog := &fakeSessionCatalog{
		sessions:  map[string]TerminalSession{},
		createErr: terminal.ErrInvalidWorkingDirectory,
	}
	server := httptest.NewServer(NewHandlerWithSessions(catalog, RuntimeContext{TerminalID: "prototype"}))
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/terminals", bytes.NewBufferString(`{"working_directory":"/missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", server.URL)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST unavailable cwd status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestSessionAPICreatesTerminalInSubmittedWorkingDirectory(t *testing.T) {
	t.Parallel()

	target := "/Users/test/Developer/personal/projects/Yuniverse"
	catalog := &fakeSessionCatalog{
		sessions: map[string]TerminalSession{},
		created:  terminal.Info{ID: "session-a1b2c3d4e5f6", AgentID: "codex", WorkingDirectory: target},
	}
	server := httptest.NewServer(NewHandlerWithSessions(catalog, RuntimeContext{TerminalID: "prototype"}))
	t.Cleanup(server.Close)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/terminals", bytes.NewBufferString(`{"working_directory":"`+target+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", server.URL)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("POST terminals status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if catalog.createWorkingDirectory != target {
		t.Fatalf("catalog cwd = %q, want %q", catalog.createWorkingDirectory, target)
	}
	var created RuntimeContext
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.WorkingDirectory != target || created.WorkspaceName != "Yuniverse" {
		t.Fatalf("created context = %#v, want selected directory", created)
	}
}

func (catalog *fakeSessionCatalog) ResumeInDirectory(_ context.Context, nativeID, title, workingDirectory string, _, _ uint16) (terminal.Info, error) {
	catalog.resumeID = nativeID
	catalog.resumeWorkingDirectory = workingDirectory
	catalog.resumed.Title = title
	catalog.infos = append(catalog.infos, catalog.resumed)
	return catalog.resumed, nil
}

func (catalog *fakeSessionCatalog) Get(id string) (TerminalSession, bool) {
	session, ok := catalog.sessions[id]
	return session, ok
}

type fakeCodexHistory struct {
	threads          []codex.Thread
	workingDirectory string
}

func (history *fakeCodexHistory) List(_ context.Context, workingDirectory string) ([]codex.Thread, error) {
	history.workingDirectory = workingDirectory
	return append([]codex.Thread(nil), history.threads...), nil
}

func TestSessionAPIListsAndCreatesIndependentTerminals(t *testing.T) {
	t.Parallel()

	catalog := &fakeSessionCatalog{
		infos: []terminal.Info{{
			ID:               "prototype",
			AgentID:          "codex",
			WorkingDirectory: "/tmp/code-remote",
			Title:            "问候用户 | code-remote",
			CreatedAt:        time.Unix(1789959000, 0).UTC(),
		}},
		sessions: map[string]TerminalSession{},
		created: terminal.Info{
			ID:               "session-a1b2c3d4",
			AgentID:          "codex",
			WorkingDirectory: "/tmp/code-remote",
			CreatedAt:        time.Unix(1789959100, 0).UTC(),
		},
	}
	server := httptest.NewServer(NewHandlerWithSessions(catalog, RuntimeContext{TerminalID: "prototype"}))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/api/v1/terminals")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET terminals status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var listed struct {
		Terminals []struct {
			TerminalID string `json:"terminal_id"`
			Title      string `json:"title"`
		} `json:"terminals"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Terminals) != 1 || listed.Terminals[0].TerminalID != "prototype" {
		t.Fatalf("listed terminals = %#v, want prototype", listed.Terminals)
	}
	if listed.Terminals[0].Title != "问候用户 | code-remote" {
		t.Fatalf("listed terminal title = %q, want native pane title", listed.Terminals[0].Title)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/terminals", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", server.URL)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("POST terminals status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	var created RuntimeContext
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.TerminalID != "session-a1b2c3d4" || created.WorkspaceName != "code-remote" {
		t.Fatalf("created terminal = %#v, want new session context", created)
	}
}

func TestSessionAPIRejectsCrossOriginCreateAndUnknownAttach(t *testing.T) {
	t.Parallel()

	catalog := &fakeSessionCatalog{sessions: map[string]TerminalSession{}}
	handler := NewHandlerWithSessions(catalog, RuntimeContext{TerminalID: "prototype"})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/terminals", nil)
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST status = %d, want %d", response.Code, http.StatusForbidden)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/terminals/missing/attach", nil)
	request.Header.Set("Origin", "http://example.com")
	request.Host = "example.com"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown attach status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestCodexHistoryAPIListsNativeNamesAndResumesSelectedThread(t *testing.T) {
	t.Parallel()

	threadID := "019abcde-1234-7000-8000-123456789abc"
	catalog := &fakeSessionCatalog{
		sessions: map[string]TerminalSession{},
		resumed: terminal.Info{
			ID:               "session-a1b2c3d4e5f6",
			AgentID:          "codex",
			WorkingDirectory: "/tmp/code-remote",
		},
	}
	history := &fakeCodexHistory{threads: []codex.Thread{{
		ID: threadID, Name: "重构移动端 UI", Preview: "重新设计界面", UpdatedAt: time.Unix(1789959300, 0).UTC(),
	}}}
	server := httptest.NewServer(NewHandlerWithSessionHistory(catalog, history, RuntimeContext{TerminalID: "prototype"}))
	t.Cleanup(server.Close)

	workingDirectory := "/tmp/Yuniverse"
	response, err := http.Get(server.URL + "/api/v1/codex/threads?working_directory=" + url.QueryEscape(workingDirectory))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var listed struct {
		Threads []codex.Thread `json:"threads"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Threads) != 1 || listed.Threads[0].Name != "重构移动端 UI" {
		t.Fatalf("listed history = %#v, want native Codex name", listed.Threads)
	}
	if history.workingDirectory != workingDirectory {
		t.Fatalf("history cwd = %q, want %q", history.workingDirectory, workingDirectory)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/codex/threads/"+threadID+"/resume", bytes.NewBufferString(`{"working_directory":"`+workingDirectory+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", server.URL)
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("resume status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if catalog.resumeID != threadID || catalog.resumed.Title != "重构移动端 UI" || catalog.resumeWorkingDirectory != workingDirectory {
		t.Fatalf("resume = %q / %q, want selected native thread", catalog.resumeID, catalog.resumed.Title)
	}
}
