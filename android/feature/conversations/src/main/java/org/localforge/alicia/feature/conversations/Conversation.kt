package org.localforge.alicia.feature.conversations

/**
 * UI model representing a conversation in the conversations list.
 *
 * This is a simplified view model tailored for the conversations screen UI,
 * distinct from the domain model which contains additional data and business logic.
 *
 * @property id Unique identifier for the conversation
 * @property title Optional title of the conversation, defaults to "Conversation" in UI if null
 * @property lastMessage Preview text of the most recent message, or null if no messages exist
 * @property timestamp Unix timestamp in milliseconds of the last update to this conversation
 * @property messageCount Total number of messages in the conversation
 */
data class Conversation(
    val id: String,
    val title: String?,
    val lastMessage: String?,
    val timestamp: Long,
    val messageCount: Int = 0
)
