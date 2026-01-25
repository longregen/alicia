package com.alicia.assistant.skills

import android.content.Context
import android.util.Log
import com.alicia.assistant.AliciaApplication
import com.alicia.assistant.service.AliciaApiClient
import com.alicia.assistant.telemetry.AliciaTelemetry
import com.alicia.assistant.ws.AssistantWebSocket
import com.alicia.assistant.ws.MessageListener
import io.opentelemetry.api.common.Attributes
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.coroutines.resume

data class SkillResult(
    val success: Boolean,
    val response: String,
    val action: String? = null
)

class SkillRouter(
    private val context: Context,
    private val apiClient: AliciaApiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
) {
    companion object {
        private const val TAG = "SkillRouter"
        private const val RESPONSE_TIMEOUT_MS = 120_000L
    }

    private var voiceConversationId: String? = null

    private val webSocket: AssistantWebSocket?
        get() = (context.applicationContext as? AliciaApplication)?.assistantWebSocket

    suspend fun processInput(input: String, screenContext: String? = null): SkillResult {
        return AliciaTelemetry.withSpanAsync("skill.process_input",
            Attributes.builder()
                .put("skill.input_length", input.length.toLong())
                .put("skill.has_screen_context", screenContext != null)
                .build()
        ) { span ->
            try {
                val convId = getOrCreateVoiceConversation()
                val content = if (screenContext != null) {
                    "[Screen content]\n$screenContext\n[End screen content]\n\nUser: $input"
                } else {
                    input
                }

                val ws = webSocket
                if (ws != null && ws.isConnected()) {
                    sendViaWebSocket(ws, convId, content)
                } else {
                    Log.w(TAG, "WebSocket not available, falling back to HTTP")
                    sendViaHttp(convId, content)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Chat failed", e)
                AliciaTelemetry.recordError(span, e)
                SkillResult(
                    success = false,
                    response = "Sorry, I couldn't get a response right now."
                )
            }
        }
    }

    private suspend fun sendViaWebSocket(ws: AssistantWebSocket, convId: String, content: String): SkillResult {
        ws.subscribeToConversation(convId)

        val response = withTimeoutOrNull(RESPONSE_TIMEOUT_MS) {
            suspendCancellableCoroutine { continuation ->
                val resumed = AtomicBoolean(false)

                fun resumeOnce(value: String?) {
                    if (resumed.compareAndSet(false, true)) {
                        ws.removeMessageListener(convId)
                        continuation.resume(value)
                    }
                }

                val listener = object : MessageListener {
                    override fun onUserMessage(conversationId: String, messageId: String, content: String, previousId: String?) {
                        Log.d(TAG, "User message confirmed: $messageId")
                    }

                    override fun onAssistantMessage(conversationId: String, messageId: String, content: String, previousId: String?) {
                        resumeOnce(content)
                    }

                    override fun onThinkingUpdate(conversationId: String, messageId: String, content: String, progress: Float) {
                        Log.d(TAG, "Thinking: $content")
                    }

                    override fun onToolUseStarted(conversationId: String, toolName: String, arguments: Map<String, Any?>) {
                        Log.d(TAG, "Tool started: $toolName")
                    }

                    override fun onToolUseCompleted(conversationId: String, toolName: String, success: Boolean) {
                        Log.d(TAG, "Tool completed: $toolName")
                    }

                    override fun onError(conversationId: String, error: String) {
                        resumeOnce("Error: $error")
                    }
                }

                ws.addMessageListener(convId, listener)

                if (!ws.sendUserMessage(convId, content)) {
                    resumeOnce(null)
                }

                continuation.invokeOnCancellation {
                    ws.removeMessageListener(convId)
                }
            }
        }

        if (response == null) {
            ws.removeMessageListener(convId)
        }

        return if (response != null) {
            SkillResult(success = true, response = response, action = "ws_chat")
        } else {
            Log.w(TAG, "WebSocket timeout, falling back to HTTP")
            sendViaHttp(convId, content)
        }
    }

    private suspend fun sendViaHttp(convId: String, content: String): SkillResult {
        val response = apiClient.sendMessageSync(convId, content)
        return SkillResult(
            success = true,
            response = response.assistantMessage.content,
            action = "api_chat"
        )
    }

    private suspend fun getOrCreateVoiceConversation(): String {
        voiceConversationId?.let { return it }
        val conversation = apiClient.createConversation("Voice Chat")
        voiceConversationId = conversation.id
        return conversation.id
    }
}
