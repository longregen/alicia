package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartMCPToolSpan starts a span for an MCP tool execution.
// It extracts trace context from the _meta field and creates a child span.
func StartMCPToolSpan(ctx context.Context, tracerName, serviceName, toolName string, meta map[string]any) (context.Context, trace.Span) {
	// Extract trace context from _meta field for distributed tracing
	ctx = ExtractMCPMeta(ctx, meta)

	// Create span for tool execution
	ctx, span := Tracer(tracerName).Start(ctx, "tool."+toolName,
		trace.WithAttributes(
			attribute.String("mcp.tool_name", toolName),
			attribute.String(AttrTraceName, serviceName+":"+toolName),
		))

	return ctx, span
}

// EndMCPToolSpan records tool result attributes and ends the span.
func EndMCPToolSpan(span trace.Span, isError bool, resultLength int) {
	span.SetAttributes(
		attribute.Bool("tool.is_error", isError),
		attribute.Int("tool.result_length", resultLength),
	)
	span.End()
}

// RecordToolError records an error on the span and marks it as an error.
func RecordToolError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetAttributes(attribute.Bool("tool.is_error", true))
}
