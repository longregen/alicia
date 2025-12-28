package mcp

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Loopback addresses
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback other", "127.0.0.5", true},
		{"IPv6 loopback", "::1", true},

		// Private IPv4 ranges
		{"10.x.x.x", "10.0.0.1", true},
		{"10.255.255.255", "10.255.255.255", true},
		{"172.16.x.x", "172.16.0.1", true},
		{"172.31.x.x", "172.31.255.255", true},
		{"192.168.x.x", "192.168.1.1", true},
		{"192.168.0.1", "192.168.0.1", true},

		// Link-local addresses
		{"IPv4 link-local", "169.254.1.1", true},
		{"IPv6 link-local", "fe80::1", true},

		// Unspecified addresses
		{"IPv4 unspecified", "0.0.0.0", true},
		{"IPv6 unspecified", "::", true},

		// Multicast addresses
		{"IPv4 multicast", "224.0.0.1", true},
		{"IPv6 multicast", "ff02::1", true},

		// Public addresses
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Public IPv6", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP: %s", tt.ip)
			}
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP_Nil(t *testing.T) {
	result := isPrivateIP(nil)
	if result != false {
		t.Error("isPrivateIP(nil) should return false")
	}
}

func TestValidateURL(t *testing.T) {
	// Save original allowed hosts and restore after test
	originalHosts := AllowedURLHosts
	defer func() { AllowedURLHosts = originalHosts }()

	// Clear allowed hosts for blocklist-based tests
	AllowedURLHosts = nil

	tests := []struct {
		name        string
		url         string
		shouldError bool
		errorMsg    string
	}{
		// Invalid URLs
		{"empty URL", "", true, "URL must have a hostname"},
		{"invalid URL", "not-a-url", true, "URL must have a hostname"},

		// Disallowed schemes
		{"file scheme", "file:///etc/passwd", true, "unsupported URL scheme"},
		{"ftp scheme", "ftp://example.com", true, "unsupported URL scheme"},
		{"gopher scheme", "gopher://example.com", true, "unsupported URL scheme"},
		{"javascript scheme", "javascript:alert(1)", true, "unsupported URL scheme"},

		// Internal hostnames
		{"localhost", "http://localhost:8080", true, "internal/metadata hostname"},
		{"localhost.localdomain", "http://localhost.localdomain", true, "internal/metadata hostname"},
		{"metadata", "http://metadata", true, "internal/metadata hostname"},
		{"metadata.google.internal", "http://metadata.google.internal", true, "internal/metadata hostname"},
		{"AWS metadata IP", "http://169.254.169.254", true, "internal/metadata hostname"},
		{"kubernetes", "http://kubernetes", true, "internal/metadata hostname"},
		{"kubernetes.default.svc.cluster.local", "http://kubernetes.default.svc.cluster.local", true, "internal/metadata hostname"},

		// Note: We can't test DNS resolution reliably in unit tests
		// Private IPs would require DNS to resolve, so we test the hostname blocking instead
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.shouldError {
				if err == nil {
					t.Errorf("validateURL(%q) should have returned an error", tt.url)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateURL(%q) error = %q, expected to contain %q", tt.url, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateURL(%q) returned unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestValidateURL_AllowedHosts(t *testing.T) {
	// Save original allowed hosts and restore after test
	originalHosts := AllowedURLHosts
	defer func() { AllowedURLHosts = originalHosts }()

	// Set up allowlist
	AllowedURLHosts = []string{"api.example.com", "mcp.example.org"}

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"allowed host 1", "https://api.example.com/endpoint", false},
		{"allowed host 2", "https://mcp.example.org/sse", false},
		{"allowed host case insensitive", "https://API.EXAMPLE.COM/endpoint", false},
		{"not in allowlist", "https://other.example.com/endpoint", true},
		{"localhost blocked by allowlist", "http://localhost:8080", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.shouldError && err == nil {
				t.Errorf("validateURL(%q) should have returned an error", tt.url)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("validateURL(%q) returned unexpected error: %v", tt.url, err)
			}
		})
	}
}

func TestNewHTTPSSETransport_URLValidation(t *testing.T) {
	// Save original allowed hosts and restore after test
	originalHosts := AllowedURLHosts
	defer func() { AllowedURLHosts = originalHosts }()
	AllowedURLHosts = nil

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"localhost blocked", "http://localhost:8080", true},
		{"private IP hostname", "http://169.254.169.254", true},
		{"file scheme blocked", "file:///etc/passwd", true},
		// Note: Can't test valid public URLs without DNS resolution
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewHTTPSSETransport(tt.url, "")
			if tt.shouldError && err == nil {
				t.Errorf("NewHTTPSSETransport(%q) should have returned an error", tt.url)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("NewHTTPSSETransport(%q) returned unexpected error: %v", tt.url, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
