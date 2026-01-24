package com.alicia.assistant.tools

import android.content.ClipboardManager
import android.content.Context
import com.alicia.assistant.ws.ToolExecutor
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

class GetClipboardExecutor(private val context: Context) : ToolExecutor {
    override val name = "get_clipboard"
    override val description = "Get the current clipboard contents from the user's phone"
    override val inputSchema = mapOf<String, Any>(
        "type" to "object",
        "properties" to emptyMap<String, Any>()
    )

    override suspend fun execute(arguments: Map<String, Any>): Map<String, Any> {
        // ClipboardManager must be accessed from the main thread
        return withContext(Dispatchers.Main) {
            val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
            val clip = clipboard.primaryClip
            val text = clip?.getItemAt(0)?.text?.toString() ?: ""
            mapOf(
                "text" to text,
                "hasContent" to text.isNotEmpty()
            )
        }
    }
}
