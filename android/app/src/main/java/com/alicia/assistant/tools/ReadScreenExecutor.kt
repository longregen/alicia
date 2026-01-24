package com.alicia.assistant.tools

import com.alicia.assistant.ws.ToolExecutor

class ReadScreenExecutor : ToolExecutor {
    companion object {
        @Volatile
        var lastScreenContent: String? = null

        fun updateScreenContent(content: String) {
            lastScreenContent = content
        }
    }

    override val name = "read_screen"
    override val description = "Read the text currently displayed on the user's phone screen"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        val text = lastScreenContent
        return if (text != null && text.isNotBlank()) {
            mapOf("text" to text)
        } else {
            mapOf(
                "text" to "",
                "note" to "No screen content available. Screen context is captured during voice assist sessions."
            )
        }
    }
}
