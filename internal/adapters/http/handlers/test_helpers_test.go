package handlers

import (
	"context"
	"net/http"

	"github.com/longregen/alicia/internal/adapters/http/middleware"
)

// Helper function to add user context to requests
func addUserContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	return req.WithContext(ctx)
}
