package com.alicia.assistant.service

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class AliciaApiClient(
    private val baseUrl: String,
    private val userId: String
) {
    companion object {
        private const val SYNC_TIMEOUT_MS = 120_000L
        val BASE_URL = ApiClient.BASE_URL
        const val USER_ID = "usr"
        private val JSON_MEDIA_TYPE = "application/json; charset=utf-8".toMediaType()
    }

    private val client: OkHttpClient = ApiClient.client

    private val syncClient: OkHttpClient = client.newBuilder()
        .readTimeout(SYNC_TIMEOUT_MS, TimeUnit.MILLISECONDS)
        .build()

    data class Conversation(
        val id: String,
        val title: String,
        val status: String,
        val createdAt: String,
        val updatedAt: String
    )

    data class Message(
        val id: String,
        val conversationId: String,
        val role: String,
        val content: String,
        val status: String,
        val previousId: String? = null,
        val toolUses: List<ToolUseInfo> = emptyList()
    )

    data class ToolUseInfo(
        val toolName: String,
        val status: String
    )

    data class SyncResponse(
        val userMessage: Message,
        val assistantMessage: Message
    )

    suspend fun createConversation(title: String = "New Chat"): Conversation = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("title", title)
        }
        val response = post("/api/v1/conversations", body)
        parseConversation(response)
    }

    suspend fun listConversations(): List<Conversation> = withContext(Dispatchers.IO) {
        val response = get("/api/v1/conversations")
        val conversations = response.optJSONArray("conversations") ?: JSONArray()
        (0 until conversations.length()).map { parseConversation(conversations.getJSONObject(it)) }
    }

    suspend fun getConversation(id: String): Conversation = withContext(Dispatchers.IO) {
        val response = get("/api/v1/conversations/$id")
        parseConversation(response)
    }

    suspend fun getMessages(conversationId: String): List<Message> = withContext(Dispatchers.IO) {
        val response = get("/api/v1/conversations/$conversationId/messages")
        val messages = response.optJSONArray("messages") ?: JSONArray()
        (0 until messages.length()).map { parseMessage(messages.getJSONObject(it)) }
    }

    suspend fun sendMessageSync(
        conversationId: String,
        content: String,
        previousId: String? = null
    ): SyncResponse = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("content", content)
            put("use_pareto", false)
            if (previousId != null) {
                put("previous_id", previousId)
            }
        }
        val response = post(
            "/api/v1/conversations/$conversationId/messages?sync=true",
            body,
            useSyncClient = true
        )

        SyncResponse(
            userMessage = parseMessage(response.getJSONObject("user_message")),
            assistantMessage = parseMessage(response.getJSONObject("assistant_message"))
        )
    }

    suspend fun getPreferences(): JSONObject = withContext(Dispatchers.IO) {
        get("/api/v1/preferences")
    }

    suspend fun updatePreferences(updates: JSONObject): JSONObject = withContext(Dispatchers.IO) {
        patch("/api/v1/preferences", updates)
    }

    private fun get(path: String): JSONObject {
        val request = Request.Builder()
            .url("$baseUrl$path")
            .header("X-User-ID", userId)
            .header("Accept", "application/json")
            .get()
            .build()

        return executeRequest(client, request)
    }

    private fun post(path: String, body: JSONObject, useSyncClient: Boolean = false): JSONObject {
        val requestBody = body.toString().toRequestBody(JSON_MEDIA_TYPE)
        val request = Request.Builder()
            .url("$baseUrl$path")
            .header("X-User-ID", userId)
            .header("Accept", "application/json")
            .post(requestBody)
            .build()

        val httpClient = if (useSyncClient) syncClient else client
        return executeRequest(httpClient, request)
    }

    private fun patch(path: String, body: JSONObject): JSONObject {
        val requestBody = body.toString().toRequestBody(JSON_MEDIA_TYPE)
        val request = Request.Builder()
            .url("$baseUrl$path")
            .header("X-User-ID", userId)
            .header("Accept", "application/json")
            .patch(requestBody)
            .build()

        return executeRequest(client, request)
    }

    private fun executeRequest(httpClient: OkHttpClient, request: Request): JSONObject {
        httpClient.newCall(request).execute().use { response ->
            val responseBody = response.body?.string() ?: ""
            if (!response.isSuccessful) {
                throw ApiException(response.code, responseBody)
            }
            return JSONObject(responseBody)
        }
    }

    private fun parseConversation(json: JSONObject): Conversation {
        return Conversation(
            id = json.getString("id"),
            title = json.optString("title", ""),
            status = json.optString("status", "active"),
            createdAt = json.optString("created_at", ""),
            updatedAt = json.optString("updated_at", "")
        )
    }

    private fun parseMessage(json: JSONObject): Message {
        val toolUses = mutableListOf<ToolUseInfo>()
        json.optJSONArray("tool_uses")?.let { arr ->
            for (i in 0 until arr.length()) {
                val tu = arr.getJSONObject(i)
                toolUses.add(ToolUseInfo(
                    toolName = tu.optString("tool_name", "unknown"),
                    status = tu.optString("status", "")
                ))
            }
        }

        return Message(
            id = json.getString("id"),
            conversationId = json.optString("conversation_id", ""),
            role = json.optString("role", ""),
            content = json.optString("content", ""),
            status = json.optString("status", ""),
            previousId = json.optString("previous_id", "").takeIf { it.isNotEmpty() },
            toolUses = toolUses
        )
    }

    // --- Notes API ---

    data class Note(
        val id: String,
        val title: String,
        val content: String,
        val createdAt: String,
        val updatedAt: String
    )

    suspend fun listNotes(): List<Note> = withContext(Dispatchers.IO) {
        val response = get("/api/v1/notes")
        val notes = response.optJSONArray("notes") ?: JSONArray()
        (0 until notes.length()).map { parseNote(notes.getJSONObject(it)) }
    }

    suspend fun getNote(id: String): Note = withContext(Dispatchers.IO) {
        val response = get("/api/v1/notes/$id")
        parseNote(response)
    }

    suspend fun createNote(id: String, title: String, content: String): Note = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("id", id)
            put("title", title)
            put("content", content)
        }
        val response = post("/api/v1/notes", body)
        parseNote(response)
    }

    suspend fun updateNote(id: String, title: String, content: String): Note = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("title", title)
            put("content", content)
        }
        val response = put("/api/v1/notes/$id", body)
        parseNote(response)
    }

    suspend fun deleteNote(id: String): Unit = withContext(Dispatchers.IO) {
        delete("/api/v1/notes/$id")
    }

    private fun parseNote(json: JSONObject): Note {
        return Note(
            id = json.getString("id"),
            title = json.optString("title", ""),
            content = json.optString("content", ""),
            createdAt = json.optString("created_at", ""),
            updatedAt = json.optString("updated_at", "")
        )
    }

    // --- HTTP helpers ---

    private fun put(path: String, body: JSONObject): JSONObject {
        val requestBody = body.toString().toRequestBody(JSON_MEDIA_TYPE)
        val request = Request.Builder()
            .url("$baseUrl$path")
            .header("X-User-ID", userId)
            .header("Accept", "application/json")
            .put(requestBody)
            .build()

        return executeRequest(client, request)
    }

    private fun delete(path: String) {
        val request = Request.Builder()
            .url("$baseUrl$path")
            .header("X-User-ID", userId)
            .header("Accept", "application/json")
            .delete()
            .build()

        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                val responseBody = response.body?.string() ?: ""
                throw ApiException(response.code, responseBody)
            }
        }
    }

    class ApiException(val statusCode: Int, val body: String) :
        Exception("API error $statusCode: $body")
}
