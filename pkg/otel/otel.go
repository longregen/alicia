// Package otel provides OpenTelemetry SDK initialization for Alicia services.
package otel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName  string
	Environment  string
	OTLPEndpoint string // HTTP endpoint URL (e.g., https://alicia-data.hjkl.lol/otlp)
}

// InitResult holds the logger and shutdown function from Init.
type InitResult struct {
	Logger   *slog.Logger
	Shutdown func(context.Context) error
}

// Init initializes the OpenTelemetry SDK with OTLP HTTP exporters for traces and logs.
// Returns an InitResult with a structured logger that exports to both stderr and SigNoz.
func Init(cfg Config) (*InitResult, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// --- Traces ---
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint),
		otlptracehttp.WithURLPath("/otlp/v1/traces"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// --- Logs ---
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(cfg.OTLPEndpoint),
		otlploghttp.WithURLPath("/otlp/v1/logs"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)

	otelHandler := otelslog.NewHandler(cfg.ServiceName, otelslog.WithLoggerProvider(lp))
	stderrHandler := &prettyHandler{level: slog.LevelInfo, w: os.Stderr}
	logger := slog.New(&teeHandler{handlers: []slog.Handler{otelHandler, stderrHandler}})

	shutdown := func(ctx context.Context) error {
		_ = lp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
		return nil
	}

	return &InitResult{Logger: logger, Shutdown: shutdown}, nil
}

type teeHandler struct {
	handlers []slog.Handler
	attrs    []slog.Attr
	groups   []string
}

func (t *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (t *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range t.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (t *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &teeHandler{handlers: handlers}
}

func (t *teeHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &teeHandler{handlers: handlers}
}

// NewPrettyHandler returns a slog.Handler that formats as [LEVEL hh:mm:ss] msg key=value ...
func NewPrettyHandler() slog.Handler {
	return &prettyHandler{level: slog.LevelInfo, w: os.Stderr}
}

// prettyHandler formats log records as [LEVEL hh:mm:ss] msg key=value ...
type prettyHandler struct {
	level slog.Level
	w     *os.File
	attrs []slog.Attr
	group string
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String()
	ts := r.Time.Format("15:04:05")

	var buf []byte
	buf = append(buf, '[')
	buf = append(buf, level...)
	buf = append(buf, ' ')
	buf = append(buf, ts...)
	buf = append(buf, "] "...)
	buf = append(buf, r.Message...)

	// Append pre-set attrs
	for _, a := range h.attrs {
		buf = append(buf, ' ')
		if h.group != "" {
			buf = append(buf, h.group...)
			buf = append(buf, '.')
		}
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
	}

	// Append record attrs
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		if h.group != "" {
			buf = append(buf, h.group...)
			buf = append(buf, '.')
		}
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})

	buf = append(buf, '\n')
	_, err := h.w.Write(buf)
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &prettyHandler{level: h.level, w: h.w, attrs: newAttrs, group: h.group}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	g := name
	if h.group != "" {
		g = h.group + "." + name
	}
	return &prettyHandler{level: h.level, w: h.w, attrs: h.attrs, group: g}
}

// Tracer returns a tracer for the given instrumentation name.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// TraceContext holds W3C trace context for propagation across services.
type TraceContext struct {
	TraceID    string `msgpack:"trace_id,omitempty" json:"trace_id,omitempty"`
	SpanID     string `msgpack:"span_id,omitempty" json:"span_id,omitempty"`
	TraceFlags byte   `msgpack:"trace_flags,omitempty" json:"trace_flags,omitempty"`
	SessionID  string `msgpack:"session_id,omitempty" json:"session_id,omitempty"`
	UserID     string `msgpack:"user_id,omitempty" json:"user_id,omitempty"`
}

// InjectToTraceContext extracts span info from context into a TraceContext.
func InjectToTraceContext(ctx context.Context, sessionID, userID string) TraceContext {
	tc := TraceContext{SessionID: sessionID, UserID: userID}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		sc := span.SpanContext()
		tc.TraceID = sc.TraceID().String()
		tc.SpanID = sc.SpanID().String()
		tc.TraceFlags = byte(sc.TraceFlags())
	}
	return tc
}

// ExtractFromTraceContext creates a context with span info from a TraceContext.
func ExtractFromTraceContext(ctx context.Context, tc TraceContext) context.Context {
	if tc.TraceID == "" || tc.SpanID == "" {
		return ctx
	}
	flags := "00"
	if tc.TraceFlags&0x01 != 0 {
		flags = "01"
	}
	carrier := propagation.MapCarrier{
		"traceparent": fmt.Sprintf("00-%s-%s-%s", tc.TraceID, tc.SpanID, flags),
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	if tc.SessionID != "" {
		ctx = WithSessionID(ctx, tc.SessionID)
	}
	if tc.UserID != "" {
		ctx = WithUserID(ctx, tc.UserID)
	}
	return ctx
}

// Context keys for session/user propagation.
type ctxKey int

const (
	ctxKeySessionID ctxKey = iota
	ctxKeyUserID
)

// WithSessionID adds a session ID to the context.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeySessionID, id)
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

// SessionIDFromContext retrieves the session ID from context.
func SessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeySessionID).(string); ok {
		return v
	}
	return ""
}

// UserIDFromContext retrieves the user ID from context.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyUserID).(string); ok {
		return v
	}
	return ""
}

// InjectMCPMeta creates a _meta map with trace context for MCP tool calls.
// This allows MCP servers to link their spans to the calling trace.
func InjectMCPMeta(ctx context.Context) map[string]any {
	meta := make(map[string]any)
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		sc := span.SpanContext()
		flags := "00"
		if sc.TraceFlags().IsSampled() {
			flags = "01"
		}
		meta["traceparent"] = fmt.Sprintf("00-%s-%s-%s", sc.TraceID().String(), sc.SpanID().String(), flags)
	}
	if sessionID := SessionIDFromContext(ctx); sessionID != "" {
		meta["session_id"] = sessionID
	}
	if userID := UserIDFromContext(ctx); userID != "" {
		meta["user_id"] = userID
	}
	return meta
}

// ExtractMCPMeta extracts trace context from MCP _meta field and returns a context.
func ExtractMCPMeta(ctx context.Context, meta map[string]any) context.Context {
	if meta == nil {
		return ctx
	}
	if traceparent, ok := meta["traceparent"].(string); ok && traceparent != "" {
		carrier := propagation.MapCarrier{"traceparent": traceparent}
		if tracestate, ok := meta["tracestate"].(string); ok {
			carrier["tracestate"] = tracestate
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	}
	if sessionID, ok := meta["session_id"].(string); ok && sessionID != "" {
		ctx = WithSessionID(ctx, sessionID)
	}
	if userID, ok := meta["user_id"].(string); ok && userID != "" {
		ctx = WithUserID(ctx, userID)
	}
	return ctx
}

// LangfuseTransport wraps an http.RoundTripper to inject Langfuse trace headers.
type LangfuseTransport struct {
	Base http.RoundTripper
}

// NewLangfuseTransport creates a transport that injects Langfuse headers from context.
func NewLangfuseTransport(base http.RoundTripper) *LangfuseTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &LangfuseTransport{Base: base}
}

func (t *LangfuseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	ctx := req.Context()

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req2.Header))

	if sessionID := SessionIDFromContext(ctx); sessionID != "" {
		req2.Header.Set("X-Session-Id", sessionID)
	}
	if userID := UserIDFromContext(ctx); userID != "" {
		req2.Header.Set("X-User-Id", userID)
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		sc := span.SpanContext()
		req2.Header.Set("X-Trace-Id", sc.TraceID().String())
		req2.Header.Set("X-Span-Id", sc.SpanID().String())
	}

	return t.Base.RoundTrip(req2)
}
