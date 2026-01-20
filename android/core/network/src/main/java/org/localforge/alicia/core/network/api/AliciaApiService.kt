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
     * Get sibling messages for a specific message.
     * Siblings are messages that share the same parent (previous_id).
     * Used for branch navigation.
     */
    @GET("api/v1/messages/{id}/siblings")
    suspend fun getMessageSiblings(
        @Path("id") messageId: String
    ): Response<MessageListResponse>

    /**
     * Switch conversation branch to a different message.
     * Updates the conversation's tip to point to the specified message,
     * which changes the active branch of the conversation.
     */
    @POST("api/v1/conversations/{id}/switch-branch")
    suspend fun switchBranch(
        @Path("id") conversationId: String,
        @Body request: SwitchBranchRequest
    ): Response<ConversationResponse>

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

    /**
     * Update a conversation (rename, archive, etc.)
     */
    @PATCH("api/v1/conversations/{id}")
    suspend fun updateConversation(
        @Path("id") id: String,
        @Body request: UpdateConversationRequest
    ): Response<ConversationResponse>

    // ========== Voting Endpoints ==========

    /**
     * Vote on a message
     */
    @POST("api/v1/messages/{id}/vote")
    suspend fun voteOnMessage(
        @Path("id") messageId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from a message
     */
    @DELETE("api/v1/messages/{id}/vote")
    suspend fun removeMessageVote(
        @Path("id") messageId: String
    ): Response<VoteResponse>

    /**
     * Get votes for a message
     */
    @GET("api/v1/messages/{id}/votes")
    suspend fun getMessageVotes(
        @Path("id") messageId: String
    ): Response<VoteResponse>

    /**
     * Vote on a tool use
     */
    @POST("api/v1/tool-uses/{id}/vote")
    suspend fun voteOnToolUse(
        @Path("id") toolUseId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from a tool use
     */
    @DELETE("api/v1/tool-uses/{id}/vote")
    suspend fun removeToolUseVote(
        @Path("id") toolUseId: String
    ): Response<VoteResponse>

    /**
     * Get votes for a tool use
     */
    @GET("api/v1/tool-uses/{id}/votes")
    suspend fun getToolUseVotes(
        @Path("id") toolUseId: String
    ): Response<VoteResponse>

    /**
     * Submit quick feedback for a tool use
     */
    @POST("api/v1/tool-uses/{id}/quick-feedback")
    suspend fun submitToolUseQuickFeedback(
        @Path("id") toolUseId: String,
        @Body request: QuickFeedbackRequest
    ): Response<Unit>

    /**
     * Vote on a reasoning block
     */
    @POST("api/v1/reasoning/{id}/vote")
    suspend fun voteOnReasoning(
        @Path("id") reasoningId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from a reasoning block
     */
    @DELETE("api/v1/reasoning/{id}/vote")
    suspend fun removeReasoningVote(
        @Path("id") reasoningId: String
    ): Response<VoteResponse>

    /**
     * Get votes for a reasoning block
     */
    @GET("api/v1/reasoning/{id}/votes")
    suspend fun getReasoningVotes(
        @Path("id") reasoningId: String
    ): Response<VoteResponse>

    // ========== Memory Endpoints ==========

    /**
     * Create a new memory
     */
    @POST("api/v1/memories")
    suspend fun createMemory(
        @Body request: CreateMemoryRequest
    ): Response<MemoryResponse>

    /**
     * List all memories
     */
    @GET("api/v1/memories")
    suspend fun listMemories(): Response<MemoryListResponse>

    /**
     * Get a specific memory
     */
    @GET("api/v1/memories/{id}")
    suspend fun getMemory(
        @Path("id") memoryId: String
    ): Response<MemoryResponse>

    /**
     * Update a memory
     */
    @PUT("api/v1/memories/{id}")
    suspend fun updateMemory(
        @Path("id") memoryId: String,
        @Body request: UpdateMemoryRequest
    ): Response<MemoryResponse>

    /**
     * Delete a memory
     */
    @DELETE("api/v1/memories/{id}")
    suspend fun deleteMemory(
        @Path("id") memoryId: String
    ): Response<Unit>

    /**
     * Add tags to a memory
     */
    @POST("api/v1/memories/{id}/tags")
    suspend fun addMemoryTags(
        @Path("id") memoryId: String,
        @Body request: AddTagsRequest
    ): Response<MemoryResponse>

    /**
     * Remove a tag from a memory
     */
    @DELETE("api/v1/memories/{id}/tags/{tag}")
    suspend fun removeMemoryTag(
        @Path("id") memoryId: String,
        @Path("tag") tag: String
    ): Response<MemoryResponse>

    /**
     * Pin or unpin a memory
     */
    @POST("api/v1/memories/{id}/pin")
    suspend fun pinMemory(
        @Path("id") memoryId: String,
        @Body request: PinMemoryRequest
    ): Response<MemoryResponse>

    /**
     * Archive a memory
     */
    @POST("api/v1/memories/{id}/archive")
    suspend fun archiveMemory(
        @Path("id") memoryId: String
    ): Response<MemoryResponse>

    /**
     * Search memories
     */
    @POST("api/v1/memories/search")
    suspend fun searchMemories(
        @Body request: SearchMemoriesRequest
    ): Response<MemoryListResponse>

    /**
     * Vote on a memory
     */
    @POST("api/v1/memories/{id}/vote")
    suspend fun voteOnMemory(
        @Path("id") memoryId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from a memory
     */
    @DELETE("api/v1/memories/{id}/vote")
    suspend fun removeMemoryVote(
        @Path("id") memoryId: String
    ): Response<VoteResponse>

    /**
     * Get votes for a memory
     */
    @GET("api/v1/memories/{id}/votes")
    suspend fun getMemoryVotes(
        @Path("id") memoryId: String
    ): Response<VoteResponse>

    /**
     * Set memory importance
     */
    @PUT("api/v1/memories/{id}/importance")
    suspend fun setMemoryImportance(
        @Path("id") memoryId: String,
        @Body request: SetMemoryImportanceRequest
    ): Response<MemoryResponse>

    /**
     * Get memories by tags
     */
    @GET("api/v1/memories/by-tags")
    suspend fun getMemoriesByTags(
        @Query("tags") tags: List<String>
    ): Response<MemoryListResponse>

    // ========== Memory Usage Voting ==========

    /**
     * Vote on memory usage (selection/retrieval)
     */
    @POST("api/v1/memory-usages/{id}/vote")
    suspend fun voteOnMemoryUsage(
        @Path("id") usageId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from memory usage
     */
    @DELETE("api/v1/memory-usages/{id}/vote")
    suspend fun removeMemoryUsageVote(
        @Path("id") usageId: String
    ): Response<VoteResponse>

    /**
     * Get votes for memory usage
     */
    @GET("api/v1/memory-usages/{id}/votes")
    suspend fun getMemoryUsageVotes(
        @Path("id") usageId: String
    ): Response<VoteResponse>

    /**
     * Submit irrelevance reason for memory usage
     */
    @POST("api/v1/memory-usages/{id}/irrelevance-reason")
    suspend fun submitMemoryUsageIrrelevanceReason(
        @Path("id") usageId: String,
        @Body request: IrrelevanceReasonRequest
    ): Response<VoteResponse>

    // ========== Memory Extraction Voting ==========

    /**
     * Vote on memory extraction
     */
    @POST("api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote")
    suspend fun voteOnMemoryExtraction(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    /**
     * Remove vote from memory extraction
     */
    @DELETE("api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote")
    suspend fun removeMemoryExtractionVote(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String
    ): Response<VoteResponse>

    /**
     * Get votes for memory extraction
     */
    @GET("api/v1/messages/{messageId}/extracted-memories/{memoryId}/votes")
    suspend fun getMemoryExtractionVotes(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String
    ): Response<VoteResponse>

    /**
     * Submit quality feedback for memory extraction
     */
    @POST("api/v1/messages/{messageId}/extracted-memories/{memoryId}/quality-feedback")
    suspend fun submitMemoryExtractionQualityFeedback(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String,
        @Body request: QualityFeedbackRequest
    ): Response<VoteResponse>

    // ========== Notes Endpoints ==========

    /**
     * Create a note on a message
     */
    @POST("api/v1/messages/{id}/notes")
    suspend fun createMessageNote(
        @Path("id") messageId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    /**
     * Get notes for a message
     */
    @GET("api/v1/messages/{id}/notes")
    suspend fun getMessageNotes(
        @Path("id") messageId: String
    ): Response<NoteListResponse>

    /**
     * Create a note on a tool use
     */
    @POST("api/v1/tool-uses/{id}/notes")
    suspend fun createToolUseNote(
        @Path("id") toolUseId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    /**
     * Create a note on reasoning
     */
    @POST("api/v1/reasoning/{id}/notes")
    suspend fun createReasoningNote(
        @Path("id") reasoningId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    /**
     * Update a note
     */
    @PUT("api/v1/notes/{id}")
    suspend fun updateNote(
        @Path("id") noteId: String,
        @Body request: UpdateNoteRequest
    ): Response<NoteResponse>

    /**
     * Delete a note
     */
    @DELETE("api/v1/notes/{id}")
    suspend fun deleteNote(
        @Path("id") noteId: String
    ): Response<Unit>

    // ========== Optimization Endpoints ==========

    /**
     * List optimization runs
     */
    @GET("api/v1/optimizations")
    suspend fun listOptimizationRuns(
        @QueryMap params: Map<String, String> = emptyMap()
    ): Response<List<OptimizationRunResponse>>

    /**
     * Get a specific optimization run
     */
    @GET("api/v1/optimizations/{id}")
    suspend fun getOptimizationRun(
        @Path("id") runId: String
    ): Response<OptimizationRunResponse>

    /**
     * Get optimization candidates
     */
    @GET("api/v1/optimizations/{id}/candidates")
    suspend fun getOptimizationCandidates(
        @Path("id") runId: String
    ): Response<OptimizationCandidatesResponse>

    /**
     * Get best optimization candidate
     */
    @GET("api/v1/optimizations/{id}/best")
    suspend fun getOptimizationBestCandidate(
        @Path("id") runId: String
    ): Response<OptimizationCandidateResponse>

    /**
     * Create an optimization run
     */
    @POST("api/v1/optimizations")
    suspend fun createOptimizationRun(
        @Body request: CreateOptimizationRequest
    ): Response<OptimizationRunResponse>

    // ========== Server Info Endpoints ==========

    /**
     * Get server info
     */
    @GET("api/v1/server/info")
    suspend fun getServerInfo(): Response<ServerInfoResponse>

    /**
     * Get global stats
     */
    @GET("api/v1/server/stats")
    suspend fun getGlobalStats(): Response<SessionStatsResponse>

    /**
     * Get conversation stats
     */
    @GET("api/v1/conversations/{id}/stats")
    suspend fun getConversationStats(
        @Path("id") conversationId: String
    ): Response<SessionStatsResponse>

    /**
     * Get public config
     */
    @GET("api/v1/config")
    suspend fun getConfig(): Response<PublicConfigResponse>
}
