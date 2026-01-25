package com.alicia.assistant.telemetry

import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import java.util.concurrent.ConcurrentHashMap

object ServiceTracer {
    private val serviceSpans = ConcurrentHashMap<String, Span>()

    fun onServiceStart(serviceName: String, attributes: Attributes = Attributes.empty()): Span {
        serviceSpans.remove(serviceName)?.end()

        val span = AliciaTelemetry.startSpan(
            "service.$serviceName",
            attributes
        )
        serviceSpans[serviceName] = span
        AliciaTelemetry.addSpanEvent(span, "service_started")
        return span
    }

    fun onServiceStop(serviceName: String) {
        serviceSpans.remove(serviceName)?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "service_stopped")
            span.end()
        }
    }

    fun addServiceEvent(serviceName: String, eventName: String, attributes: Attributes = Attributes.empty()) {
        serviceSpans[serviceName]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, eventName, attributes)
        }
    }

    fun getServiceSpan(serviceName: String): Span? = serviceSpans[serviceName]
}
