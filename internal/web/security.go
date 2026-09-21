// Package web contains the private HTTP and WebSocket surface exposed by the
// Mac daemon.
package web

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

var (
	tailscaleIPv4 = netip.MustParsePrefix("100.64.0.0/10")
	tailscaleIPv6 = netip.MustParsePrefix("fd7a:115c:a1e0::/48")
)

// ValidateListenAddress rejects wildcard, ordinary LAN, and public listeners.
// Code Remote may listen only on loopback for Tailscale Serve or on a numeric
// Tailscale interface address for direct tailnet access.
func ValidateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("listen address must include a numeric host and port: %w", err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("listen address has invalid port %q", port)
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("listen host %q must be a numeric IP address", host)
	}
	if ip.IsLoopback() || tailscaleIPv4.Contains(ip) || tailscaleIPv6.Contains(ip) {
		return nil
	}

	return fmt.Errorf("listen host %q is neither loopback nor a Tailscale address", host)
}

// SameOrigin reports whether a browser request came from the page served by
// the same daemon host. Missing and opaque origins are rejected.
func SameOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Host, request.Host)
}
