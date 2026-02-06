package com.alicia.assistant.telemetry

import android.app.Application
import android.os.Build
import android.util.Log
import io.opentelemetry.api.OpenTelemetry
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import io.opentelemetry.api.trace.StatusCode
import io.opentelemetry.api.trace.Tracer
import io.opentelemetry.api.trace.propagation.W3CTraceContextPropagator
import io.opentelemetry.context.Context
import io.opentelemetry.context.propagation.ContextPropagators
import io.opentelemetry.exporter.otlp.http.trace.OtlpHttpSpanExporter
import io.opentelemetry.sdk.OpenTelemetrySdk
import io.opentelemetry.sdk.resources.Resource
import io.opentelemetry.sdk.trace.SdkTracerProvider
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor
import java.util.UUID
import java.util.concurrent.TimeUnit
import kotlin.coroutines.cancellation.CancellationException

object AliciaTelemetry {
    private const val TAG = "AliciaTelemetry"
    private const val SERVICE_NAME = "alicia-android"
    private const val OTEL_ENDPOINT = "https://alicia-data.hjkl.lol/otlp/v1/traces"

    private var openTelemetry: OpenTelemetry = OpenTelemetry.noop()
    private var tracer: Tracer = OpenTelemetry.noop().getTracer(SERVICE_NAME)
    private var sdkProvider: SdkTracerProvider? = null

    var appSessionId: String = UUID.randomUUID().toString()
        private set
    var userId: String = "usr"
        private set

    fun initialize(application: Application) {
        try {
            appSessionId = UUID.randomUUID().toString()

            val resource = Resource.builder()
                .put("service.name", SERVICE_NAME)
                .put("service.version", "1.0.0")
                .put("os.type", "android")
                .put("os.version", Build.VERSION.SDK_INT.toString())
                .put("device.model.name", Build.MODEL)
                .put("device.manufacturer", Build.MANUFACTURER)
                .put("app.session_id", appSessionId)
                .put("user.id", userId)
                .build()

            val exporter = OtlpHttpSpanExporter.builder()
                .setEndpoint(OTEL_ENDPOINT)
                .setTimeout(30, TimeUnit.SECONDS)
                .build()

            val spanProcessor = BatchSpanProcessor.builder(exporter)
                .setScheduleDelay(5, TimeUnit.SECONDS)
                .setMaxExportBatchSize(512)
                .build()

            sdkProvider = SdkTracerProvider.builder()
                .setResource(resource)
                .addSpanProcessor(spanProcessor)
                .build()

            openTelemetry = OpenTelemetrySdk.builder()
                .setTracerProvider(sdkProvider!!)
                .setPropagators(ContextPropagators.create(W3CTraceContextPropagator.getInstance()))
                .build()

            tracer = openTelemetry.getTracer(SERVICE_NAME, "1.0.0")

            Log.i(TAG, "OpenTelemetry initialized (session=$appSessionId)")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to initialize OpenTelemetry", e)
        }
    }

    fun shutdown() {
        try {
            sdkProvider?.shutdown()
            Log.i(TAG, "OpenTelemetry shut down")
        } catch (e: Exception) {
            Log.e(TAG, "Error shutting down OpenTelemetry", e)
        }
    }

    fun getOpenTelemetry(): OpenTelemetry = openTelemetry

    private inline fun <T> executeWithSpan(span: Span, block: (Span) -> T): T {
        val scope = span.makeCurrent()
        return try {
            val result = block(span)
            span.end()
            result
        } catch (e: CancellationException) {
            // End span before propagating cancellation to ensure telemetry is recorded
            span.end()
            throw e
        } catch (e: Exception) {
            recordError(span, e)
            span.end()
            throw e
        } finally {
            scope.close()
        }
    }

    private fun buildSpan(name: String, attributes: Attributes): Span {
        return tracer.spanBuilder(name)
            .setAllAttributes(attributes)
            .startSpan()
    }

    fun <T> withSpan(name: String, attributes: Attributes = Attributes.empty(), block: (Span) -> T): T {
        return executeWithSpan(buildSpan(name, attributes), block)
    }

    suspend fun <T> withSpanAsync(name: String, attributes: Attributes = Attributes.empty(), block: suspend (Span) -> T): T {
        val span = buildSpan(name, attributes)
        val scope = span.makeCurrent()
        return try {
            val result = block(span)
            span.end()
            result
        } catch (e: CancellationException) {
            // End span before propagating cancellation to ensure telemetry is recorded
            span.end()
            throw e
        } catch (e: Exception) {
            recordError(span, e)
            span.end()
            throw e
        } finally {
            scope.close()
        }
    }

    fun startSpan(name: String, attributes: Attributes = Attributes.empty()): Span {
        return tracer.spanBuilder(name)
            .setAllAttributes(attributes)
            .startSpan()
    }

    fun startChildSpan(name: String, parent: Span, attributes: Attributes = Attributes.empty()): Span {
        return tracer.spanBuilder(name)
            .setParent(Context.current().with(parent))
            .setAllAttributes(attributes)
            .startSpan()
    }

    fun addSpanEvent(span: Span, name: String, attributes: Attributes = Attributes.empty()) {
        try {
            span.addEvent(name, attributes)
        } catch (e: Exception) {
            Log.w(TAG, "Failed to add span event: $name", e)
        }
    }

    fun recordError(span: Span, throwable: Throwable) {
        try {
            span.setStatus(StatusCode.ERROR, throwable.message ?: "Unknown error")
            span.recordException(throwable)
        } catch (e: Exception) {
            Log.w(TAG, "Failed to record span error", e)
        }
    }

    fun getTraceContext(span: Span? = null): Map<String, Any?> {
        val targetSpan = span ?: Span.current()
        val ctx = targetSpan.spanContext
        if (!ctx.isValid) return emptyMap()

        return mapOf(
            "trace_id" to ctx.traceId,
            "span_id" to ctx.spanId,
            "trace_flags" to ctx.traceFlags.asByte().toInt(),
            "session_id" to appSessionId,
            "user_id" to userId
        )
    }

    fun injectTraceContext(map: MutableMap<String, Any?>, span: Span? = null) {
        val traceCtx = getTraceContext(span)
        map.putAll(traceCtx)
    }
}
