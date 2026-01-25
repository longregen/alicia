package com.alicia.assistant.telemetry

import android.app.Activity
import android.app.Application
import android.os.Bundle
import io.opentelemetry.api.common.Attributes
import io.opentelemetry.api.trace.Span
import java.util.WeakHashMap

class ActivityLifecycleTracer : Application.ActivityLifecycleCallbacks {
    private val activitySpans = WeakHashMap<Activity, Span>()

    override fun onActivityCreated(activity: Activity, savedInstanceState: Bundle?) {
        val span = AliciaTelemetry.startSpan(
            "activity.${activity.javaClass.simpleName}",
            Attributes.builder()
                .put("activity.name", activity.javaClass.simpleName)
                .put("activity.restored", savedInstanceState != null)
                .build()
        )
        activitySpans[activity] = span
        AliciaTelemetry.addSpanEvent(span, "created")
    }

    override fun onActivityStarted(activity: Activity) {
        activitySpans[activity]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "started")
        }
    }

    override fun onActivityResumed(activity: Activity) {
        activitySpans[activity]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "resumed")
        }
    }

    override fun onActivityPaused(activity: Activity) {
        activitySpans[activity]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "paused")
        }
    }

    override fun onActivityStopped(activity: Activity) {
        activitySpans[activity]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "stopped")
        }
    }

    override fun onActivitySaveInstanceState(activity: Activity, outState: Bundle) {
        activitySpans[activity]?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "save_instance_state")
        }
    }

    override fun onActivityDestroyed(activity: Activity) {
        activitySpans.remove(activity)?.let { span ->
            AliciaTelemetry.addSpanEvent(span, "destroyed")
            span.end()
        }
    }
}
