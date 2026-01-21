package org.localforge.alicia.core.network.api

import org.localforge.alicia.core.network.model.*
import retrofit2.Response
import retrofit2.http.*

interface AliciaApiService {

    @POST("api/v1/conversations")
    suspend fun createConversation(
        @Body request: CreateConversationRequest
    ): Response<ConversationResponse>

    @GET("api/v1/conversations/{id}")
    suspend fun getConversation(
        @Path("id") id: String
    ): Response<ConversationResponse>

    @GET("api/v1/conversations")
    suspend fun listConversations(
        @Query("limit") limit: Int = 50,
        @Query("offset") offset: Int = 0,
        @Query("active") activeOnly: Boolean = false
    ): Response<ConversationListResponse>

    @DELETE("api/v1/conversations/{id}")
    suspend fun deleteConversation(
        @Path("id") id: String
    ): Response<Unit>

    @POST("api/v1/conversations/{id}/token")
    suspend fun getConversationToken(
        @Path("id") id: String,
        @Body request: GenerateTokenRequest
    ): Response<TokenResponse>

    @GET("api/v1/conversations/{id}/messages")
    suspend fun getMessages(
        @Path("id") id: String
    ): Response<MessageListResponse>

    @POST("api/v1/conversations/{id}/messages")
    suspend fun sendMessage(
        @Path("id") id: String,
        @Body request: SendMessageRequest
    ): Response<MessageResponse>

    @GET("api/v1/messages/{id}/siblings")
    suspend fun getMessageSiblings(
        @Path("id") messageId: String
    ): Response<MessageListResponse>

    @POST("api/v1/conversations/{id}/switch-branch")
    suspend fun switchBranch(
        @Path("id") conversationId: String,
        @Body request: SwitchBranchRequest
    ): Response<ConversationResponse>

    @GET("api/v1/voices")
    suspend fun getVoices(): Response<List<VoiceResponse>>

    suspend fun getAvailableVoices(): Response<List<VoiceResponse>> = getVoices()

    suspend fun getConversations(): Response<ConversationListResponse> = listConversations()

    @GET("api/v1/mcp/servers")
    suspend fun getMCPServers(): Response<MCPServersResponse>

    @POST("api/v1/mcp/servers")
    suspend fun addMCPServer(
        @Body request: MCPServerConfigRequest
    ): Response<MCPServerResponse>

    @DELETE("api/v1/mcp/servers/{name}")
    suspend fun deleteMCPServer(
        @Path("name") name: String
    ): Response<Unit>

    @GET("api/v1/mcp/tools")
    suspend fun getMCPTools(): Response<MCPToolsResponse>

    @POST("api/v1/conversations/{id}/sync")
    suspend fun syncMessages(
        @Path("id") conversationId: String,
        @Body request: SyncMessagesRequest
    ): Response<SyncMessagesResponse>

    @GET("api/v1/conversations/{id}/sync/status")
    suspend fun getSyncStatus(
        @Path("id") conversationId: String
    ): Response<SyncStatusResponse>

    @PATCH("api/v1/conversations/{id}")
    suspend fun updateConversation(
        @Path("id") id: String,
        @Body request: UpdateConversationRequest
    ): Response<ConversationResponse>

    @POST("api/v1/messages/{id}/vote")
    suspend fun voteOnMessage(
        @Path("id") messageId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/messages/{id}/vote")
    suspend fun removeMessageVote(
        @Path("id") messageId: String
    ): Response<VoteResponse>

    @GET("api/v1/messages/{id}/votes")
    suspend fun getMessageVotes(
        @Path("id") messageId: String
    ): Response<VoteResponse>

    @POST("api/v1/tool-uses/{id}/vote")
    suspend fun voteOnToolUse(
        @Path("id") toolUseId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/tool-uses/{id}/vote")
    suspend fun removeToolUseVote(
        @Path("id") toolUseId: String
    ): Response<VoteResponse>

    @GET("api/v1/tool-uses/{id}/votes")
    suspend fun getToolUseVotes(
        @Path("id") toolUseId: String
    ): Response<VoteResponse>

    @POST("api/v1/tool-uses/{id}/quick-feedback")
    suspend fun submitToolUseQuickFeedback(
        @Path("id") toolUseId: String,
        @Body request: QuickFeedbackRequest
    ): Response<Unit>

    @POST("api/v1/reasoning/{id}/vote")
    suspend fun voteOnReasoning(
        @Path("id") reasoningId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/reasoning/{id}/vote")
    suspend fun removeReasoningVote(
        @Path("id") reasoningId: String
    ): Response<VoteResponse>

    @GET("api/v1/reasoning/{id}/votes")
    suspend fun getReasoningVotes(
        @Path("id") reasoningId: String
    ): Response<VoteResponse>

    @POST("api/v1/memories")
    suspend fun createMemory(
        @Body request: CreateMemoryRequest
    ): Response<MemoryResponse>

    @GET("api/v1/memories")
    suspend fun listMemories(): Response<MemoryListResponse>

    @GET("api/v1/memories/{id}")
    suspend fun getMemory(
        @Path("id") memoryId: String
    ): Response<MemoryResponse>

    @PUT("api/v1/memories/{id}")
    suspend fun updateMemory(
        @Path("id") memoryId: String,
        @Body request: UpdateMemoryRequest
    ): Response<MemoryResponse>

    @DELETE("api/v1/memories/{id}")
    suspend fun deleteMemory(
        @Path("id") memoryId: String
    ): Response<Unit>

    @POST("api/v1/memories/{id}/tags")
    suspend fun addMemoryTags(
        @Path("id") memoryId: String,
        @Body request: AddTagsRequest
    ): Response<MemoryResponse>

    @DELETE("api/v1/memories/{id}/tags/{tag}")
    suspend fun removeMemoryTag(
        @Path("id") memoryId: String,
        @Path("tag") tag: String
    ): Response<MemoryResponse>

    @POST("api/v1/memories/{id}/pin")
    suspend fun pinMemory(
        @Path("id") memoryId: String,
        @Body request: PinMemoryRequest
    ): Response<MemoryResponse>

    @POST("api/v1/memories/{id}/archive")
    suspend fun archiveMemory(
        @Path("id") memoryId: String
    ): Response<MemoryResponse>

    @POST("api/v1/memories/search")
    suspend fun searchMemories(
        @Body request: SearchMemoriesRequest
    ): Response<MemoryListResponse>

    @POST("api/v1/memories/{id}/vote")
    suspend fun voteOnMemory(
        @Path("id") memoryId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/memories/{id}/vote")
    suspend fun removeMemoryVote(
        @Path("id") memoryId: String
    ): Response<VoteResponse>

    @GET("api/v1/memories/{id}/votes")
    suspend fun getMemoryVotes(
        @Path("id") memoryId: String
    ): Response<VoteResponse>

    @PUT("api/v1/memories/{id}/importance")
    suspend fun setMemoryImportance(
        @Path("id") memoryId: String,
        @Body request: SetMemoryImportanceRequest
    ): Response<MemoryResponse>

    @GET("api/v1/memories/by-tags")
    suspend fun getMemoriesByTags(
        @Query("tags") tags: List<String>
    ): Response<MemoryListResponse>

    @POST("api/v1/memory-usages/{id}/vote")
    suspend fun voteOnMemoryUsage(
        @Path("id") usageId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/memory-usages/{id}/vote")
    suspend fun removeMemoryUsageVote(
        @Path("id") usageId: String
    ): Response<VoteResponse>

    @GET("api/v1/memory-usages/{id}/votes")
    suspend fun getMemoryUsageVotes(
        @Path("id") usageId: String
    ): Response<VoteResponse>

    @POST("api/v1/memory-usages/{id}/irrelevance-reason")
    suspend fun submitMemoryUsageIrrelevanceReason(
        @Path("id") usageId: String,
        @Body request: IrrelevanceReasonRequest
    ): Response<VoteResponse>

    @POST("api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote")
    suspend fun voteOnMemoryExtraction(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String,
        @Body request: VoteRequest
    ): Response<VoteResponse>

    @DELETE("api/v1/messages/{messageId}/extracted-memories/{memoryId}/vote")
    suspend fun removeMemoryExtractionVote(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String
    ): Response<VoteResponse>

    @GET("api/v1/messages/{messageId}/extracted-memories/{memoryId}/votes")
    suspend fun getMemoryExtractionVotes(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String
    ): Response<VoteResponse>

    @POST("api/v1/messages/{messageId}/extracted-memories/{memoryId}/quality-feedback")
    suspend fun submitMemoryExtractionQualityFeedback(
        @Path("messageId") messageId: String,
        @Path("memoryId") memoryId: String,
        @Body request: QualityFeedbackRequest
    ): Response<VoteResponse>

    @POST("api/v1/messages/{id}/notes")
    suspend fun createMessageNote(
        @Path("id") messageId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    @GET("api/v1/messages/{id}/notes")
    suspend fun getMessageNotes(
        @Path("id") messageId: String
    ): Response<NoteListResponse>

    @POST("api/v1/tool-uses/{id}/notes")
    suspend fun createToolUseNote(
        @Path("id") toolUseId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    @POST("api/v1/reasoning/{id}/notes")
    suspend fun createReasoningNote(
        @Path("id") reasoningId: String,
        @Body request: CreateNoteRequest
    ): Response<NoteResponse>

    @PUT("api/v1/notes/{id}")
    suspend fun updateNote(
        @Path("id") noteId: String,
        @Body request: UpdateNoteRequest
    ): Response<NoteResponse>

    @DELETE("api/v1/notes/{id}")
    suspend fun deleteNote(
        @Path("id") noteId: String
    ): Response<Unit>

    @GET("api/v1/optimizations")
    suspend fun listOptimizationRuns(
        @QueryMap params: Map<String, String> = emptyMap()
    ): Response<List<OptimizationRunResponse>>

    @GET("api/v1/optimizations/{id}")
    suspend fun getOptimizationRun(
        @Path("id") runId: String
    ): Response<OptimizationRunResponse>

    @GET("api/v1/optimizations/{id}/candidates")
    suspend fun getOptimizationCandidates(
        @Path("id") runId: String
    ): Response<OptimizationCandidatesResponse>

    @GET("api/v1/optimizations/{id}/best")
    suspend fun getOptimizationBestCandidate(
        @Path("id") runId: String
    ): Response<OptimizationCandidateResponse>

    @POST("api/v1/optimizations")
    suspend fun createOptimizationRun(
        @Body request: CreateOptimizationRequest
    ): Response<OptimizationRunResponse>

    @GET("api/v1/server/info")
    suspend fun getServerInfo(): Response<ServerInfoResponse>

    @GET("api/v1/server/stats")
    suspend fun getGlobalStats(): Response<SessionStatsResponse>

    @GET("api/v1/conversations/{id}/stats")
    suspend fun getConversationStats(
        @Path("id") conversationId: String
    ): Response<SessionStatsResponse>

    @GET("api/v1/config")
    suspend fun getConfig(): Response<PublicConfigResponse>
}
