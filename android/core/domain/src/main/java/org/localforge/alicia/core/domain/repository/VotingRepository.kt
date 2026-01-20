package org.localforge.alicia.core.domain.repository

import org.localforge.alicia.core.domain.model.Vote
import org.localforge.alicia.core.domain.model.VoteResult

/**
 * Repository interface for voting operations.
 * Matches the web frontend's voting functionality for messages, tools, memories, and reasoning.
 */
interface VotingRepository {
    // Message voting
    suspend fun voteOnMessage(messageId: String, vote: Vote, quickFeedback: String? = null): VoteResult
    suspend fun removeMessageVote(messageId: String): VoteResult
    suspend fun getMessageVotes(messageId: String): VoteResult

    // Tool use voting
    suspend fun voteOnToolUse(toolUseId: String, vote: Vote, quickFeedback: String? = null): VoteResult
    suspend fun removeToolUseVote(toolUseId: String): VoteResult
    suspend fun getToolUseVotes(toolUseId: String): VoteResult
    suspend fun submitToolUseQuickFeedback(toolUseId: String, feedback: String)

    // Reasoning voting
    suspend fun voteOnReasoning(reasoningId: String, vote: Vote): VoteResult
    suspend fun removeReasoningVote(reasoningId: String): VoteResult
    suspend fun getReasoningVotes(reasoningId: String): VoteResult

    // Memory voting
    suspend fun voteOnMemory(memoryId: String, vote: Vote): VoteResult
    suspend fun removeMemoryVote(memoryId: String): VoteResult
    suspend fun getMemoryVotes(memoryId: String): VoteResult

    // Memory usage voting
    suspend fun voteOnMemoryUsage(usageId: String, vote: Vote): VoteResult
    suspend fun removeMemoryUsageVote(usageId: String): VoteResult
    suspend fun getMemoryUsageVotes(usageId: String): VoteResult
    suspend fun submitMemoryUsageIrrelevanceReason(usageId: String, reason: String)

    // Memory extraction voting
    suspend fun voteOnMemoryExtraction(messageId: String, memoryId: String, vote: Vote): VoteResult
    suspend fun removeMemoryExtractionVote(messageId: String, memoryId: String): VoteResult
    suspend fun getMemoryExtractionVotes(messageId: String, memoryId: String): VoteResult
    suspend fun submitMemoryExtractionQualityFeedback(messageId: String, memoryId: String, feedback: String)
}
