package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.common.parseTimestamp
import org.localforge.alicia.core.database.entity.ConversationEntity
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.network.model.ConversationResponse

/**
 * Mapper for converting between Conversation domain model and data layer entities/responses.
 */

/**
 * Convert ConversationEntity to Conversation domain model.
 */
fun ConversationEntity.toDomain(): Conversation {
    return Conversation(
        id = id,
        title = title,
        createdAt = createdAt,
        updatedAt = updatedAt,
        syncedAt = syncedAt,
        isDeleted = isDeleted
    )
}

/**
 * Convert Conversation domain model to ConversationEntity.
 */
fun Conversation.toEntity(): ConversationEntity {
    return ConversationEntity(
        id = id,
        title = title,
        createdAt = createdAt,
        updatedAt = updatedAt,
        syncedAt = syncedAt,
        isDeleted = isDeleted
    )
}

/**
 * Convert ConversationResponse to Conversation domain model.
 */
fun ConversationResponse.toDomain(): Conversation {
    return Conversation(
        id = id,
        title = title,
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        syncedAt = System.currentTimeMillis(),
        isDeleted = false
    )
}

/**
 * Convert ConversationResponse to ConversationEntity.
 */
fun ConversationResponse.toEntity(): ConversationEntity {
    return ConversationEntity(
        id = id,
        title = title,
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        syncedAt = System.currentTimeMillis(),
        isDeleted = false
    )
}

/**
 * Convert list of ConversationEntities to list of Conversation domain models.
 */
fun List<ConversationEntity>.toDomain(): List<Conversation> {
    return map { it.toDomain() }
}

/**
 * Convert list of ConversationResponses to list of Conversation domain models.
 */
fun List<ConversationResponse>.toDomainFromResponse(): List<Conversation> {
    return map { it.toDomain() }
}

/**
 * Convert list of ConversationResponses to list of ConversationEntities.
 */
fun List<ConversationResponse>.toEntities(): List<ConversationEntity> {
    return map { it.toEntity() }
}
