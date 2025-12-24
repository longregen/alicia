package middleware

import (
	"net/http"
)

// CORS returns a middleware that handles CORS headers securely.
// It validates the request Origin header against the allowed origins list
// and only sets credentials if the origin is explicitly allowed.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	// Pre-build a map for faster origin lookups
	allowedOriginsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginsMap[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if the origin is in the allowed list
			originAllowed := allowedOriginsMap[origin]

			// Only set CORS headers if origin is allowed
			if originAllowed && origin != "" {
				// Set the specific origin, never wildcard with credentials
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Set other CORS headers (these are safe without credentials)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Expose-Headers", "Link")
			w.Header().Set("Access-Control-Max-Age", "300")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				if originAllowed {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
