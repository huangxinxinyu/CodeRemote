package web

import "testing"

func TestValidateListenAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{name: "IPv4 loopback", address: "127.0.0.1:8080"},
		{name: "IPv6 loopback", address: "[::1]:8080"},
		{name: "Tailscale IPv4", address: "100.78.102.15:8080"},
		{name: "Tailscale IPv6", address: "[fd7a:115c:a1e0::2401:66d7]:8080"},
		{name: "all IPv4 interfaces", address: "0.0.0.0:8080", wantErr: true},
		{name: "all IPv6 interfaces", address: "[::]:8080", wantErr: true},
		{name: "ordinary LAN", address: "192.168.1.20:8080", wantErr: true},
		{name: "missing port", address: "127.0.0.1", wantErr: true},
		{name: "hostname is not a verified interface", address: "localhost:8080", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateListenAddress(tt.address)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateListenAddress(%q) succeeded, want error", tt.address)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateListenAddress(%q) returned error: %v", tt.address, err)
			}
		})
	}
}
