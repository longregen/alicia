package org.localforge.alicia.core.network.api

import org.localforge.alicia.core.network.model.*
import retrofit2.Response
import retrofit2.http.*

/**
 * Retrofit API service for Alicia backend
 */
interface AliciaApiService {

    /**
     * Create a new conversation
     */
    @POST("api/v1/conversations")
    suspend fun createConversation(
        @Body request: CreateConversationRequest
    ): Response<ConversationResponse>

    /**
     * Get a specific conversation by ID
     */
    @GET("api/v1/conversations/{id}")
    suspend fun getConversation(
        @Path("id") id: String
    ): Response<ConversationResponse>

    /**
     * List all conversations
     */
    @GET("api/v1/conversations")
    suspend fun listConversations(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
        @Query("active") activeOnly: Boolean = false
    ): Response<ConversationListResponse>

    /**
     * Delete a conversation
     */
    @DELETE("api/v1/conversations/{id}")
    suspend fun deleteConversation(
        @Path("id") id: String
    ): Response<Unit>

    /**
     * Generate a LiveKit token for a conversation
     */
    @POST("api/v1/conversations/{id}/token")
    suspend fun getConversationToken(
        @Path("id") id: String,
        @Body request: GenerateTokenRequest
    ): Response<TokenResponse>

    /**
     * Get messages for a conversation
     */
    @GET("api/v1/conversations/{id}/messages")
    suspend fun getMessages(
        @Path("id") id: String
    ): Response<MessageListResponse>

    /**
     * Send a message to a conversation
     */
    @POST("api/v1/conversations/{id}/messages")
    suspend fun sendMessage(
        @Path("id") id: String,
        @Body request: SendMessageRequest
    ): Response<MessageResponse>

    /**
     * Get available voices
     */
    @GET("api/v1/voices")
    suspend fun getVoices(): Response<List<VoiceResponse>>

    /**
     * Convenience alias for [getVoices]. Delegates to the same endpoint.
     */
    suspend fun getAvailableVoices(): Response<List<VoiceResponse>> = getVoices()

    /**
     * Convenience alias for [listConversations]. Delegates to the same endpoint.
     */
    suspend fun getConversations(): Response<ConversationListResponse> = listConversations()

    /**
     * Get all configured MCP servers
     */
    @GET("api/v1/mcp/servers")
    suspend fun getMCPServers(): Response<MCPServersResponse>

    /**
     * Add a new MCP server
     */
    @POST("api/v1/mcp/servers")
    suspend fun addMCPServer(
        @Body request: MCPServerConfigRequest
    ): Response<MCPServerResponse>

    /**
     * Delete an MCP server by name
     */
    @DELETE("api/v1/mcp/servers/{name}")
    suspend fun deleteMCPServer(
        @Path("name") name: String
    ): Response<Unit>

    /**
     * Get all available MCP tools from all servers
     */
    @GET("api/v1/mcp/tools")
    suspend fun getMCPTools(): Response<MCPToolsResponse>

    /**
     * Sync messages with the server.
     */
    @POST("api/v1/conversations/{id}/sync")
    suspend fun syncMessages(
        @Path("id") conversationId: String,
        @Body request: SyncMessagesRequest
    ): Response<SyncMessagesResponse>

    /**
     * Get synchronization status for a conversation.
     */
    @GET("api/v1/conversations/{id}/sync/status")
    suspend fun getSyncStatus(
        @Path("id") conversationId: String
    ): Response<SyncStatusResponse>
}
