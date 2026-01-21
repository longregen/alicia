package org.localforge.alicia.core.domain.model

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
    val categoryDisplayName: String
        get() = when (category) {
            MemoryCategory.PREFERENCE -> "Preference"
            MemoryCategory.FACT -> "Fact"
            MemoryCategory.CONTEXT -> "Context"
            MemoryCategory.INSTRUCTION -> "Instruction"
        }
}
