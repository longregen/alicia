package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/longregen/alicia/internal/adapters/metrics"
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &statusRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
