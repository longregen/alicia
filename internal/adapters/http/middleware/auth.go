package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
)

// Simple header-based auth suitable for internal VPN deployments.
// For production with external access, consider OAuth2/OIDC, JWT, or API keys.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		userID = strings.TrimSpace(userID)

		// Default for backward compatibility; production should reject instead
		if userID == "" {
			userID = "default_user"
		}

		// Prevent injection attacks
		if !isValidUserID(userID) {
			log.Printf("HTTP 400: Invalid user ID format: %q (path=%s)", userID, r.URL.Path)
			http.Error(w, "Invalid user ID format", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	if !ok {
		return ""
	}
	return userID
}

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
