package otel

import (
	"net/http"

	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Middleware returns an OpenTelemetry middleware for Chi routers.
// Extracts session.id, user.id, and request.id from headers.
func Middleware(serviceName string, opts ...otelchi.Option) func(http.Handler) http.Handler {
	baseMiddleware := otelchi.Middleware(serviceName, opts...)

	return func(next http.Handler) http.Handler {
		return baseMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			if span.IsRecording() {
				if sessionID := r.Header.Get("x-session-id"); sessionID != "" {
					span.SetAttributes(attribute.String(AttrSessionID, sessionID))
				}
				if userID := r.Header.Get("x-user-id"); userID != "" {
					span.SetAttributes(attribute.String(AttrUserID, userID))
				}
				if requestID := r.Header.Get("x-request-id"); requestID != "" {
					span.SetAttributes(attribute.String(AttrRequestID, requestID))
				}
			}
			next.ServeHTTP(w, r)
		}))
	}
}
