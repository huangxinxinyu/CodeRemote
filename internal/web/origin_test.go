package web

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{name: "direct tailnet HTTP", host: "100.78.102.15:8080", origin: "http://100.78.102.15:8080", want: true},
		{name: "Serve HTTPS", host: "mac-mini.example.ts.net", origin: "https://mac-mini.example.ts.net", want: true},
		{name: "foreign site", host: "100.78.102.15:8080", origin: "https://attacker.example", want: false},
		{name: "different port", host: "100.78.102.15:8080", origin: "http://100.78.102.15:9090", want: false},
		{name: "missing browser origin", host: "100.78.102.15:8080", origin: "", want: false},
		{name: "malformed origin", host: "100.78.102.15:8080", origin: "://", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "http://"+tt.host+"/api/v1/terminals/prototype/attach", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := SameOrigin(req); got != tt.want {
				t.Fatalf("SameOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
