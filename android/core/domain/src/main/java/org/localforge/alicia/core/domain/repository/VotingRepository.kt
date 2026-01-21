package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Vote
import org.localforge.alicia.core.domain.model.VoteResult

interface VotingRepository {
    suspend fun voteOnMessage(messageId: String, vote: Vote, quickFeedback: String? = null): VoteResult
    suspend fun removeMessageVote(messageId: String): VoteResult
    suspend fun getMessageVotes(messageId: String): VoteResult

    suspend fun voteOnToolUse(toolUseId: String, vote: Vote, quickFeedback: String? = null): VoteResult
    suspend fun removeToolUseVote(toolUseId: String): VoteResult
    suspend fun getToolUseVotes(toolUseId: String): VoteResult
    suspend fun submitToolUseQuickFeedback(toolUseId: String, feedback: String)

    suspend fun voteOnReasoning(reasoningId: String, vote: Vote): VoteResult
    suspend fun removeReasoningVote(reasoningId: String): VoteResult
    suspend fun getReasoningVotes(reasoningId: String): VoteResult

    suspend fun voteOnMemory(memoryId: String, vote: Vote): VoteResult
    suspend fun removeMemoryVote(memoryId: String): VoteResult
    suspend fun getMemoryVotes(memoryId: String): VoteResult

    suspend fun voteOnMemoryUsage(usageId: String, vote: Vote): VoteResult
    suspend fun removeMemoryUsageVote(usageId: String): VoteResult
    suspend fun getMemoryUsageVotes(usageId: String): VoteResult
    suspend fun submitMemoryUsageIrrelevanceReason(usageId: String, reason: String)

    suspend fun voteOnMemoryExtraction(messageId: String, memoryId: String, vote: Vote): VoteResult
    suspend fun removeMemoryExtractionVote(messageId: String, memoryId: String): VoteResult
    suspend fun getMemoryExtractionVotes(messageId: String, memoryId: String): VoteResult
    suspend fun submitMemoryExtractionQualityFeedback(messageId: String, memoryId: String, feedback: String)
}
