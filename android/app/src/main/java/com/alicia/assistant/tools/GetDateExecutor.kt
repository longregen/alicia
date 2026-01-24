package com.alicia.assistant.tools

import com.alicia.assistant.ws.ToolExecutor
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.format.TextStyle
import java.util.Locale

class GetDateExecutor : ToolExecutor {
    override val name = "get_date"
    override val description = "Get the current date and day of week from the user's phone"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        val today = LocalDate.now()
        return mapOf(
            "date" to today.format(DateTimeFormatter.ISO_LOCAL_DATE),
            "dayOfWeek" to today.dayOfWeek.getDisplayName(TextStyle.FULL, Locale.getDefault()),
            "formatted" to today.format(DateTimeFormatter.ofPattern("MMMM d, yyyy"))
        )
    }
}
