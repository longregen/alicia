package org.localforge.alicia.feature.conversations

data class Conversation(
    val id: String,
    val title: String?,
    val lastMessage: String?,
    val timestamp: Long,
    val messageCount: Int = 0,
    val isArchived: Boolean = false
)
