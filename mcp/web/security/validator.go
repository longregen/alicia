package security

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
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
