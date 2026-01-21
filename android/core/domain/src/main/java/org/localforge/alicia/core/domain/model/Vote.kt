package org.localforge.alicia.core.domain.model

enum class Vote(val value: String) {
    UP("up"),
    DOWN("down"),
    CRITICAL("critical");

    companion object {
        fun fromString(value: String?): Vote? {
            return entries.find { it.value == value }
        }
    }
}

data class VoteResult(
    val targetId: String,
    val targetType: String,
    val upvotes: Int,
    val downvotes: Int,
    val userVote: Vote?,
    val special: Map<String, Int> = emptyMap()
) {
    val totalVotes: Int
        get() = upvotes + downvotes

    val score: Int
        get() = upvotes - downvotes

    val hasVoted: Boolean
        get() = userVote != null
}

object VoteTargetType {
    const val MESSAGE = "message"
    const val TOOL_USE = "tool_use"
    const val MEMORY = "memory"
    const val MEMORY_USAGE = "memory_usage"
    const val MEMORY_EXTRACTION = "memory_extraction"
    const val REASONING = "reasoning"
}
