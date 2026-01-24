package com.alicia.assistant.skills

import android.content.Context
import android.util.Log
import com.alicia.assistant.service.AliciaApiClient

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
    }

    private var voiceConversationId: String? = null

    suspend fun processInput(input: String, screenContext: String? = null): SkillResult {
        return try {
            val convId = getOrCreateVoiceConversation()
            val content = if (screenContext != null) {
                "[Screen content]\n$screenContext\n[End screen content]\n\nUser: $input"
            } else {
                input
            }
            val response = apiClient.sendMessageSync(convId, content)
            SkillResult(
                success = true,
                response = response.assistantMessage.content,
                action = "api_chat"
            )
        } catch (e: Exception) {
            Log.e(TAG, "API chat failed", e)
            SkillResult(
                success = false,
                response = "Sorry, I couldn't get a response right now."
            )
        }
    }

    private suspend fun getOrCreateVoiceConversation(): String {
        voiceConversationId?.let { return it }
        val conversation = apiClient.createConversation("Voice Chat")
        voiceConversationId = conversation.id
        return conversation.id
    }
}
