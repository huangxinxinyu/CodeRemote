package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path/filepath"
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
	Model            string `json:"model"`
}

// NewHandler returns the private browser UI and terminal attachment endpoint.
func NewHandler(attach http.Handler) http.Handler {
	return NewHandlerWithContext(attach, RuntimeContext{})
}

// NewHandlerWithContext returns the browser UI with its configured runtime context.
func NewHandlerWithContext(attach http.Handler, runtimeContext RuntimeContext) http.Handler {
	assets, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	if runtimeContext.WorkspaceName == "" && runtimeContext.WorkingDirectory != "" {
		runtimeContext.WorkspaceName = filepath.Base(runtimeContext.WorkingDirectory)
	}
	if runtimeContext.Model == "" {
		runtimeContext.Model = "native session"
	}

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
	if attach != nil {
		mux.Handle("/api/v1/terminals/prototype/attach", attach)
	} else {
		mux.HandleFunc("/api/v1/terminals/prototype/attach", func(response http.ResponseWriter, _ *http.Request) {
			http.Error(response, "terminal attachment is not configured", http.StatusServiceUnavailable)
		})
	}

	return securityHeaders(mux)
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
