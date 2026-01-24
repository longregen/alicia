package com.alicia.assistant.tools

import com.alicia.assistant.ws.ToolExecutor
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter

class GetTimeExecutor : ToolExecutor {
    override val name = "get_time"
    override val description = "Get the current time and timezone from the user's phone"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        val now = ZonedDateTime.now()
        return mapOf(
            "time" to now.format(DateTimeFormatter.ofPattern("HH:mm:ss")),
            "timezone" to now.zone.id,
            "offset" to now.offset.toString()
        )
    }
}
