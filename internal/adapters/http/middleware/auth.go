package middleware

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDContextKey is the context key for storing the user ID
	UserIDContextKey contextKey = "user_id"
)

// Auth is a middleware that extracts the user ID from the X-User-ID header
// and adds it to the request context. This is a simple header-based authentication
// suitable for internal VPN deployments.
//
// For production deployments with external access, consider implementing:
// - OAuth2/OIDC integration
// - JWT token validation
// - API key authentication
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from X-User-ID header
		userID := r.Header.Get("X-User-ID")

		// Trim whitespace and validate
		userID = strings.TrimSpace(userID)

		// If no user ID is provided, use "default_user" for backward compatibility
		// In a production system, you might want to reject the request instead
		if userID == "" {
			userID = "default_user"
		}

		// Validate user ID format (alphanumeric, hyphens, underscores only)
		// This prevents injection attacks
		if !isValidUserID(userID) {
			http.Error(w, "Invalid user ID format", http.StatusBadRequest)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context
// Returns empty string if not found
func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// isValidUserID validates that the user ID contains only safe characters
// Allows: alphanumeric, hyphens, underscores, dots, @
func isValidUserID(userID string) bool {
	if userID == "" || len(userID) > 255 {
		return false
	}

	for _, ch := range userID {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.' || ch == '@') {
			return false
		}
	}

	return true
}
