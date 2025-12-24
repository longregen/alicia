package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name                   string
		method                 string
		origin                 string
		expectAllowOrigin      string
		expectAllowCredentials string
		expectStatusCode       int
	}{
		{
			name:                   "Allowed origin with credentials",
			method:                 "GET",
			origin:                 "http://localhost:3000",
			expectAllowOrigin:      "http://localhost:3000",
			expectAllowCredentials: "true",
			expectStatusCode:       http.StatusOK,
		},
		{
			name:                   "Another allowed origin",
			method:                 "POST",
			origin:                 "https://example.com",
			expectAllowOrigin:      "https://example.com",
			expectAllowCredentials: "true",
			expectStatusCode:       http.StatusOK,
		},
		{
			name:                   "Disallowed origin",
			method:                 "GET",
			origin:                 "https://evil.com",
			expectAllowOrigin:      "",
			expectAllowCredentials: "",
			expectStatusCode:       http.StatusOK,
		},
		{
			name:                   "No origin header",
			method:                 "GET",
			origin:                 "",
			expectAllowOrigin:      "",
			expectAllowCredentials: "",
			expectStatusCode:       http.StatusOK,
		},
		{
			name:                   "Preflight request allowed origin",
			method:                 "OPTIONS",
			origin:                 "http://localhost:3000",
			expectAllowOrigin:      "http://localhost:3000",
			expectAllowCredentials: "true",
			expectStatusCode:       http.StatusNoContent,
		},
		{
			name:                   "Preflight request disallowed origin",
			method:                 "OPTIONS",
			origin:                 "https://evil.com",
			expectAllowOrigin:      "",
			expectAllowCredentials: "",
			expectStatusCode:       http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectStatusCode, rr.Code)
			}

			allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if allowOrigin != tt.expectAllowOrigin {
				t.Errorf("Expected Access-Control-Allow-Origin '%s', got '%s'", tt.expectAllowOrigin, allowOrigin)
			}

			allowCredentials := rr.Header().Get("Access-Control-Allow-Credentials")
			if allowCredentials != tt.expectAllowCredentials {
				t.Errorf("Expected Access-Control-Allow-Credentials '%s', got '%s'", tt.expectAllowCredentials, allowCredentials)
			}

			// Verify other headers are always set
			allowMethods := rr.Header().Get("Access-Control-Allow-Methods")
			if allowMethods == "" {
				t.Error("Access-Control-Allow-Methods should be set")
			}

			allowHeaders := rr.Header().Get("Access-Control-Allow-Headers")
			if allowHeaders == "" {
				t.Error("Access-Control-Allow-Headers should be set")
			}
		})
	}
}

// TestCORS_SecurityVulnerability verifies that wildcard + credentials is never allowed
func TestCORS_SecurityVulnerability(t *testing.T) {
	allowedOrigins := []string{"http://localhost:3000"}
	handler := CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
	allowCredentials := rr.Header().Get("Access-Control-Allow-Credentials")

	// Verify we never have wildcard + credentials together
	if allowOrigin == "*" && allowCredentials == "true" {
		t.Error("SECURITY VULNERABILITY: wildcard origin with credentials is not allowed by CORS spec")
	}

	// Verify we set specific origin, not wildcard
	if allowOrigin == "*" {
		t.Error("Should not use wildcard origin when credentials are enabled")
	}

	if allowOrigin != "http://localhost:3000" {
		t.Errorf("Expected specific origin 'http://localhost:3000', got '%s'", allowOrigin)
	}

	if allowCredentials != "true" {
		t.Error("Credentials should be enabled for allowed origins")
	}
}
