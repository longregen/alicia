package security

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// allowLocal caches whether MCP_WEB_ALLOW_LOCAL is set at startup.
var allowLocal = os.Getenv("MCP_WEB_ALLOW_LOCAL") != ""

// IsPrivateIP checks if an IP address is private, loopback, or otherwise internal
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check for loopback (127.0.0.0/8 for IPv4, ::1 for IPv6)
	if ip.IsLoopback() {
		return true
	}

	// Check for private addresses
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local addresses (169.254.0.0/16 for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check for unspecified addresses (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check for multicast addresses
	if ip.IsMulticast() {
		return true
	}

	// Check for IPv4-mapped IPv6 addresses
	if len(ip) == net.IPv6len {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.IsLoopback() || ip4.IsPrivate() || ip4.IsLinkLocalUnicast() ||
				ip4.IsLinkLocalMulticast() || ip4.IsUnspecified() || ip4.IsMulticast()
		}
	}

	return false
}

// ValidateURL validates that a URL is safe for server-side requests.
// It prevents SSRF attacks by blocking requests to internal/private networks.
func ValidateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Extract hostname (without port)
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	lowerHostname := strings.ToLower(hostname)

	// When MCP_WEB_ALLOW_LOCAL is set, skip localhost/private-IP checks
	// but still block cloud metadata endpoints (SSRF to metadata services
	// is dangerous even in development).
	metadataHostnames := []string{
		"metadata",
		"metadata.google.internal",
		"instance-data",
		"169.254.169.254",
		"metadata.azure.com",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
	}
	for _, meta := range metadataHostnames {
		if lowerHostname == meta || strings.HasSuffix(lowerHostname, "."+meta) {
			return fmt.Errorf("hostname %q is not allowed: cloud metadata endpoint", hostname)
		}
	}

	if !allowLocal {
		// Block local/internal hostnames
		localHostnames := []string{
			"localhost",
			"localhost.localdomain",
			"local",
			"internal",
		}
		for _, local := range localHostnames {
			if lowerHostname == local || strings.HasSuffix(lowerHostname, "."+local) {
				return fmt.Errorf("hostname %q is not allowed: internal hostname (set MCP_WEB_ALLOW_LOCAL=1 to permit)", hostname)
			}
		}

		// Resolve hostname to IP addresses and check each one
		ips, err := net.LookupIP(hostname)
		if err != nil {
			return fmt.Errorf("cannot resolve hostname %q: %w", hostname, err)
		}

		for _, ip := range ips {
			if IsPrivateIP(ip) {
				return fmt.Errorf("hostname %q resolves to private/internal IP address %s (set MCP_WEB_ALLOW_LOCAL=1 to permit)", hostname, ip.String())
			}
		}
	}

	return nil
}

// SafeDialContext returns a DialContext function that resolves DNS and validates
// all resulting IPs against IsPrivateIP before connecting. This prevents DNS
// rebinding attacks where the first lookup returns a public IP (passing
// ValidateURL) but a subsequent lookup returns a private IP.
func SafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}

	// Resolve DNS ourselves so we can inspect the IPs before connecting.
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %q: %w", host, err)
	}

	if !allowLocal {
		for _, ipAddr := range ips {
			if IsPrivateIP(ipAddr.IP) {
				return nil, fmt.Errorf("hostname %q resolves to private/internal IP %s", host, ipAddr.IP)
			}
		}
	}

	// Connect to the first reachable resolved IP. Use a raw dialer with
	// Control to pin the connection to the resolved addresses and block any
	// OS-level re-resolution.
	var lastErr error
	for _, ipAddr := range ips {
		dialer := &net.Dialer{
			Control: func(network, address string, c syscall.RawConn) error {
				// Extra safety: the kernel is connecting to `address`; verify
				// it is one of the IPs we already validated.
				connHost, _, _ := net.SplitHostPort(address)
				connIP := net.ParseIP(connHost)
				if connIP != nil && !allowLocal && IsPrivateIP(connIP) {
					return fmt.Errorf("blocked connection to private IP %s", connIP)
				}
				return nil
			},
		}
		target := net.JoinHostPort(ipAddr.IP.String(), port)
		conn, err := dialer.DialContext(ctx, network, target)
		if err != nil {
			lastErr = err
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("failed to connect to %q: %w", addr, lastErr)
}
