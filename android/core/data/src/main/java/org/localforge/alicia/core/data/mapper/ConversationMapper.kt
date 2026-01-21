package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.common.parseTimestamp
import org.localforge.alicia.core.database.entity.ConversationEntity
import org.localforge.alicia.core.domain.model.Conversation
import org.localforge.alicia.core.domain.model.ConversationStatus
import org.localforge.alicia.core.network.model.ConversationResponse

fun ConversationEntity.toDomain(): Conversation {
    return Conversation(
        id = id,
        title = title,
        status = ConversationStatus.fromString(status),
        createdAt = createdAt,
        updatedAt = updatedAt,
        syncedAt = syncedAt,
        isDeleted = isDeleted
    )
}

fun Conversation.toEntity(): ConversationEntity {
    return ConversationEntity(
        id = id,
        title = title,
        status = status.value,
        createdAt = createdAt,
        updatedAt = updatedAt,
        syncedAt = syncedAt,
        isDeleted = isDeleted
    )
}

fun ConversationResponse.toDomain(): Conversation {
    return Conversation(
        id = id,
        title = title,
        status = ConversationStatus.fromString(status),
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        syncedAt = System.currentTimeMillis(),
        isDeleted = false
    )
}

fun ConversationResponse.toEntity(): ConversationEntity {
    return ConversationEntity(
        id = id,
        title = title,
        status = status,
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        syncedAt = System.currentTimeMillis(),
        isDeleted = false
    )
}

fun List<ConversationEntity>.toDomain(): List<Conversation> {
    return map { it.toDomain() }
}

fun List<ConversationResponse>.toDomainFromResponse(): List<Conversation> {
    return map { it.toDomain() }
}

fun List<ConversationResponse>.toEntities(): List<ConversationEntity> {
    return map { it.toEntity() }
}
