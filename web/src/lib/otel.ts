/**
 * OpenTelemetry instrumentation for Alicia web frontend.
 * Provides distributed tracing with auto-instrumentation for fetch and manual span creation.
 */

import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME } from '@opentelemetry/semantic-conventions';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { trace, context, SpanStatusCode, Span } from '@opentelemetry/api';
import type { Attributes } from '@opentelemetry/api';

let initialized = false;
let provider: WebTracerProvider | null = null;

/** Initialize OpenTelemetry. Call once at app startup. */
export function initOtel(serviceName = 'alicia-web'): WebTracerProvider | null {
  if (initialized) {
    console.warn('OpenTelemetry already initialized');
    return provider;
  }

  try {
    const otelEndpoint = import.meta.env.VITE_OTEL_ENDPOINT;
    const spanProcessors = [];

    if (otelEndpoint) {
      // OTLP exporter when endpoint configured
      const traceUrl = `${otelEndpoint}/v1/traces`;
      const otlpExporter = new OTLPTraceExporter({
        url: traceUrl,
      });
      spanProcessors.push(new BatchSpanProcessor(otlpExporter));
      console.log(`OpenTelemetry exporting to: ${traceUrl}`);
    }

    provider = new WebTracerProvider({
      resource: resourceFromAttributes({
        [ATTR_SERVICE_NAME]: serviceName,
      }),
      spanProcessors,
    });

    provider.register({
      contextManager: new ZoneContextManager(),
      propagator: new W3CTraceContextPropagator(),
    });

    // Build ignore pattern for OTLP endpoint to prevent infinite loop
    const ignoreUrls = otelEndpoint ? [new RegExp(otelEndpoint.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))] : [];

    registerInstrumentations({
      tracerProvider: provider,
      instrumentations: [
        new FetchInstrumentation({
          ignoreUrls,
          propagateTraceHeaderCorsUrls: [
            /localhost/,
            /127\.0\.0\.1/,
            /hjkl\.lol/,
            new RegExp(`^${window.location.origin}`),
          ],
          applyCustomAttributesOnSpan: (span: Span, request: Request | RequestInit) => {
            if (request instanceof Request) {
              span.setAttribute('http.url', request.url);
            }
          },
        }),
      ],
    });

    initialized = true;
    console.log(`OpenTelemetry initialized for service: ${serviceName}`);
    return provider;
  } catch (error) {
    console.error('Failed to initialize OpenTelemetry:', error);
    return null;
  }
}

export function getTracer(name = 'alicia-web') {
  return trace.getTracer(name);
}

export interface TraceContext {
  trace_id: string;
  span_id: string;
  trace_flags: number;
  session_id?: string;
  user_id?: string;
}

/** Get trace context from active span or provided span. */
export function getTraceContext(sessionId?: string, userId?: string, span?: Span): TraceContext | null {
  const targetSpan = span || trace.getActiveSpan();
  if (!targetSpan) return null;

  const ctx = targetSpan.spanContext();
  return {
    trace_id: ctx.traceId,
    span_id: ctx.spanId,
    trace_flags: ctx.traceFlags,
    session_id: sessionId,
    user_id: userId,
  };
}

/** Inject trace context into an object (for WebSocket messages). */
export function injectTraceContext<T extends object>(
  obj: T,
  sessionId?: string,
  userId?: string,
  span?: Span
): T & Partial<TraceContext> {
  const tc = getTraceContext(sessionId, userId, span);
  if (!tc) return obj;

  // Build result with only defined values to avoid msgpack encoding issues
  // (msgpackr encodes undefined as fixext which Go can't decode)
  const result: T & Partial<TraceContext> = {
    ...obj,
    trace_id: tc.trace_id,
    span_id: tc.span_id,
    trace_flags: tc.trace_flags,
  };
  if (tc.session_id !== undefined) result.session_id = tc.session_id;
  if (tc.user_id !== undefined) result.user_id = tc.user_id;
  return result;
}

/** Wrap a sync function with a span. */
export function withSpan<T>(name: string, fn: () => T, attributes?: Attributes): T {
  const tracer = getTracer();
  return tracer.startActiveSpan(name, (span: Span) => {
    try {
      if (attributes) span.setAttributes(attributes);
      const result = fn();
      span.end();
      return result;
    } catch (error) {
      span.setStatus({ code: SpanStatusCode.ERROR });
      span.recordException(error as Error);
      span.end();
      throw error;
    }
  });
}

/** Wrap an async function with a span. */
export async function withSpanAsync<T>(name: string, fn: () => Promise<T>, attributes?: Attributes): Promise<T> {
  const tracer = getTracer();
  return tracer.startActiveSpan(name, async (span: Span) => {
    try {
      if (attributes) span.setAttributes(attributes);
      const result = await fn();
      span.end();
      return result;
    } catch (error) {
      span.setStatus({ code: SpanStatusCode.ERROR });
      span.recordException(error as Error);
      span.end();
      throw error;
    }
  });
}

/** Start a span manually. Remember to call span.end() when done. */
export function startSpan(name: string, attributes?: Attributes): Span {
  const tracer = getTracer();
  const span = tracer.startSpan(name);
  if (attributes) span.setAttributes(attributes);
  return span;
}

/** Run a function within a specific span's context. */
export function runInSpanContext<T>(span: Span, fn: () => T): T {
  return context.with(trace.setSpan(context.active(), span), fn);
}

/** Add an event to the active span. */
export function addSpanEvent(name: string, attributes?: Attributes): void {
  trace.getActiveSpan()?.addEvent(name, attributes);
}

/** Set attributes on the active span. */
export function setSpanAttributes(attributes: Attributes): void {
  trace.getActiveSpan()?.setAttributes(attributes);
}

/** Mark the active span as errored. */
export function recordSpanError(error: Error): void {
  const span = trace.getActiveSpan();
  if (span) {
    span.setStatus({ code: SpanStatusCode.ERROR });
    span.recordException(error);
  }
}

export { SpanStatusCode, trace, context };
export type { Span, Attributes };
