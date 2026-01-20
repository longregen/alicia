package org.localforge.alicia.core.domain.model

/**
 * Memory category types matching the web frontend.
 */
enum class MemoryCategory {
    PREFERENCE,
    FACT,
    CONTEXT,
    INSTRUCTION;

    companion object {
        fun fromString(value: String): MemoryCategory {
            return when (value.lowercase()) {
                "preference" -> PREFERENCE
                "fact" -> FACT
                "context" -> CONTEXT
                "instruction" -> INSTRUCTION
                else -> PREFERENCE
            }
        }
    }

    fun toApiString(): String = name.lowercase()
}

/**
 * Domain model for a memory entry.
 *
 * Memories are persistent knowledge items that the assistant can use
 * to provide more personalized and context-aware responses.
 *
 * @property id Unique identifier for the memory
 * @property content The memory content text
 * @property category Category for organizing memories
 * @property tags Optional tags for the memory
 * @property importance Importance score (0.0 to 1.0)
 * @property pinned Whether the memory is pinned/prioritized
 * @property archived Whether the memory is archived
 * @property createdAt Timestamp when the memory was created
 * @property updatedAt Timestamp when the memory was last updated
 * @property usageCount Number of times this memory has been used
 */
data class Memory(
    val id: String,
    val content: String,
    val category: MemoryCategory,
    val tags: List<String> = emptyList(),
    val importance: Float = 0.5f,
    val pinned: Boolean = false,
    val archived: Boolean = false,
    val createdAt: Long,
    val updatedAt: Long,
    val usageCount: Int = 0
) {
    /**
     * Display name for the category
     */
    val categoryDisplayName: String
        get() = when (category) {
            MemoryCategory.PREFERENCE -> "Preference"
            MemoryCategory.FACT -> "Fact"
            MemoryCategory.CONTEXT -> "Context"
            MemoryCategory.INSTRUCTION -> "Instruction"
        }
}
