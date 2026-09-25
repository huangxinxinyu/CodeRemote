package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	codexpkg "github.com/huangxinxinyu/CodeRemote/internal/codex"
	directorypkg "github.com/huangxinxinyu/CodeRemote/internal/directory"
	terminalpkg "github.com/huangxinxinyu/CodeRemote/internal/terminal"
)

//go:embed assets/*
var embeddedAssets embed.FS

// RuntimeContext is the real terminal context exposed to the private UI.
// Model intentionally remains a native-session marker: the daemon does not
// inspect provider session files or infer model state from terminal output.
type RuntimeContext struct {
	AgentID          string `json:"agent_id"`
	WorkingDirectory string `json:"working_directory"`
	WorkspaceName    string `json:"workspace_name"`
	TerminalID       string `json:"terminal_id"`
	Title            string `json:"title,omitempty"`
	Model            string `json:"model"`
	CreatedAt        string `json:"created_at,omitempty"`
}

// SessionCatalog is the terminal lifecycle surface used by the Web API.
type SessionCatalog interface {
	List() []terminalpkg.Info
	CreateInDirectory(context.Context, string, uint16, uint16) (terminalpkg.Info, error)
	ResumeInDirectory(context.Context, string, string, string, uint16, uint16) (terminalpkg.Info, error)
	Get(string) (TerminalSession, bool)
	Delete(context.Context, string) error
}

// CodexHistory is the provider-native, read-only history surface.
type CodexHistory interface {
	List(context.Context, string) ([]codexpkg.Thread, error)
}

// NewHandler returns the private browser UI and terminal attachment endpoint.
func NewHandler(attach http.Handler) http.Handler {
	return NewHandlerWithContext(attach, RuntimeContext{})
}

// NewHandlerWithContext returns the browser UI with its configured runtime context.
func NewHandlerWithContext(attach http.Handler, runtimeContext RuntimeContext) http.Handler {
	return newHandler(attach, nil, nil, runtimeContext)
}

// NewHandlerWithSessions returns the browser UI with list/create/dynamic attach APIs.
func NewHandlerWithSessions(catalog SessionCatalog, runtimeContext RuntimeContext) http.Handler {
	return newHandler(nil, catalog, nil, runtimeContext)
}

// NewHandlerWithSessionHistory adds Codex-native saved conversations to the
// running terminal picker without introducing a product chat database.
func NewHandlerWithSessionHistory(catalog SessionCatalog, history CodexHistory, runtimeContext RuntimeContext) http.Handler {
	return newHandler(nil, catalog, history, runtimeContext)
}

func newHandler(attach http.Handler, catalog SessionCatalog, history CodexHistory, runtimeContext RuntimeContext) http.Handler {
	assets, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	runtimeContext = normalizeRuntimeContext(runtimeContext)

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("/api/v1/context", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(response).Encode(runtimeContext); err != nil {
			http.Error(response, "encode runtime context", http.StatusInternalServerError)
		}
	})
	if catalog != nil {
		mux.HandleFunc("/api/v1/directories", func(response http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodGet {
				response.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			listing, err := directorypkg.Browse(request.URL.Query().Get("path"), runtimeContext.WorkingDirectory)
			if err != nil {
				http.Error(response, "working directory is not available", http.StatusBadRequest)
				return
			}
			writeJSONResponse(response, http.StatusOK, listing)
		})
		mux.HandleFunc("/api/v1/terminals", func(response http.ResponseWriter, request *http.Request) {
			switch request.Method {
			case http.MethodGet:
				contexts := make([]RuntimeContext, 0)
				for _, info := range catalog.List() {
					contexts = append(contexts, runtimeContextFromInfo(info))
				}
				writeJSONResponse(response, http.StatusOK, map[string]any{"terminals": contexts})
			case http.MethodPost:
				if !SameOrigin(request) {
					http.Error(response, "request origin is not allowed", http.StatusForbidden)
					return
				}
				var createRequest struct {
					WorkingDirectory string `json:"working_directory"`
				}
				if request.Body != nil && request.ContentLength != 0 {
					decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 4096))
					decoder.DisallowUnknownFields()
					if err := decoder.Decode(&createRequest); err != nil {
						http.Error(response, "invalid terminal create request", http.StatusBadRequest)
						return
					}
				}
				info, err := catalog.CreateInDirectory(request.Context(), createRequest.WorkingDirectory, 80, 24)
				if errors.Is(err, terminalpkg.ErrSessionLimit) {
					http.Error(response, err.Error(), http.StatusConflict)
					return
				}
				if errors.Is(err, terminalpkg.ErrInvalidWorkingDirectory) {
					http.Error(response, "working directory is not available", http.StatusBadRequest)
					return
				}
				if err != nil {
					http.Error(response, "create terminal session", http.StatusInternalServerError)
					return
				}
				writeJSONResponse(response, http.StatusCreated, runtimeContextFromInfo(info))
			default:
				response.WriteHeader(http.StatusMethodNotAllowed)
			}
		})
		if history != nil {
			mux.HandleFunc("/api/v1/codex/threads", func(response http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					response.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				workingDirectory := request.URL.Query().Get("working_directory")
				threads, err := history.List(request.Context(), workingDirectory)
				if err != nil {
					http.Error(response, "read Codex conversation history", http.StatusBadGateway)
					return
				}
				writeJSONResponse(response, http.StatusOK, map[string]any{"threads": threads})
			})
			mux.HandleFunc("/api/v1/codex/threads/", func(response http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost {
					response.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if !SameOrigin(request) {
					http.Error(response, "request origin is not allowed", http.StatusForbidden)
					return
				}
				suffix := strings.TrimPrefix(request.URL.Path, "/api/v1/codex/threads/")
				parts := strings.Split(suffix, "/")
				if len(parts) != 2 || parts[0] == "" || parts[1] != "resume" {
					http.NotFound(response, request)
					return
				}
				var resumeRequest struct {
					WorkingDirectory string `json:"working_directory"`
				}
				if request.Body != nil && request.ContentLength != 0 {
					decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 4096))
					decoder.DisallowUnknownFields()
					if err := decoder.Decode(&resumeRequest); err != nil {
						http.Error(response, "invalid Codex resume request", http.StatusBadRequest)
						return
					}
				}
				threads, err := history.List(request.Context(), resumeRequest.WorkingDirectory)
				if err != nil {
					http.Error(response, "read Codex conversation history", http.StatusBadGateway)
					return
				}
				var selected *codexpkg.Thread
				for index := range threads {
					if threads[index].ID == parts[0] {
						selected = &threads[index]
						break
					}
				}
				if selected == nil {
					http.NotFound(response, request)
					return
				}
				info, err := catalog.ResumeInDirectory(request.Context(), selected.ID, selected.DisplayName(), resumeRequest.WorkingDirectory, 80, 24)
				if errors.Is(err, terminalpkg.ErrSessionLimit) {
					http.Error(response, err.Error(), http.StatusConflict)
					return
				}
				if errors.Is(err, terminalpkg.ErrInvalidWorkingDirectory) {
					http.Error(response, "working directory is not available", http.StatusBadRequest)
					return
				}
				if err != nil {
					http.Error(response, "resume Codex conversation", http.StatusInternalServerError)
					return
				}
				writeJSONResponse(response, http.StatusCreated, runtimeContextFromInfo(info))
			})
		}
		mux.HandleFunc("/api/v1/terminals/", func(response http.ResponseWriter, request *http.Request) {
			suffix := strings.TrimPrefix(request.URL.Path, "/api/v1/terminals/")
			parts := strings.Split(suffix, "/")
			switch request.Method {
			case http.MethodDelete:
				if !SameOrigin(request) {
					http.Error(response, "request origin is not allowed", http.StatusForbidden)
					return
				}
				if len(parts) != 1 || parts[0] == "" {
					http.NotFound(response, request)
					return
				}
				if err := catalog.Delete(request.Context(), parts[0]); err != nil {
					http.Error(response, "end terminal session", http.StatusInternalServerError)
					return
				}
				response.WriteHeader(http.StatusNoContent)
			case http.MethodGet:
				if len(parts) != 2 || parts[0] == "" || parts[1] != "attach" {
					http.NotFound(response, request)
					return
				}
				session, ok := catalog.Get(parts[0])
				if !ok {
					http.NotFound(response, request)
					return
				}
				NewAttachHandler(session).ServeHTTP(response, request)
			default:
				response.WriteHeader(http.StatusMethodNotAllowed)
			}
		})
	}
	mux.HandleFunc("/healthz", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = response.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			http.NotFound(response, request)
			return
		}
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(response, request, assets, "index.html")
	})
	if catalog != nil {
		// Dynamic terminal routes above own all attachment paths.
	} else if attach != nil {
		mux.Handle("/api/v1/terminals/prototype/attach", attach)
	} else {
		mux.HandleFunc("/api/v1/terminals/prototype/attach", func(response http.ResponseWriter, _ *http.Request) {
			http.Error(response, "terminal attachment is not configured", http.StatusServiceUnavailable)
		})
	}

	return securityHeaders(mux)
}

func runtimeContextFromInfo(info terminalpkg.Info) RuntimeContext {
	createdAt := ""
	if !info.CreatedAt.IsZero() {
		createdAt = info.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return normalizeRuntimeContext(RuntimeContext{
		AgentID:          info.AgentID,
		WorkingDirectory: info.WorkingDirectory,
		TerminalID:       info.ID,
		Title:            info.Title,
		Model:            "native session",
		CreatedAt:        createdAt,
	})
}

func normalizeRuntimeContext(runtimeContext RuntimeContext) RuntimeContext {
	if runtimeContext.WorkspaceName == "" && runtimeContext.WorkingDirectory != "" {
		runtimeContext.WorkspaceName = filepath.Base(runtimeContext.WorkingDirectory)
	}
	if runtimeContext.Model == "" {
		runtimeContext.Model = "native session"
	}
	return runtimeContext
}

func writeJSONResponse(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(response, request)
	})
}
