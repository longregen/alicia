package org.localforge.alicia.core.data.mapper

import org.localforge.alicia.core.common.parseTimestamp
import org.localforge.alicia.core.database.entity.MessageEntity
import org.localforge.alicia.core.domain.model.Message
import org.localforge.alicia.core.domain.model.MessageRole
import org.localforge.alicia.core.domain.model.SyncStatus
import org.localforge.alicia.core.network.model.MessageResponse

fun MessageEntity.toDomain(): Message {
    return Message(
        id = id,
        conversationId = conversationId,
        role = MessageRole.fromString(role),
        content = content,
        createdAt = createdAt,
        updatedAt = updatedAt,
        sequenceNumber = sequenceNumber,
        previousId = previousId,
        isVoice = isVoice,
        audioPath = audioPath,
        audioDurationMs = audioDurationMs,
        localId = localId,
        serverId = serverId,
        syncStatus = SyncStatus.fromString(syncStatus),
        syncedAt = syncedAt
    )
}

fun Message.toEntity(): MessageEntity {
    return MessageEntity(
        id = id,
        conversationId = conversationId,
        role = role.value,
        content = content,
        createdAt = createdAt,
        updatedAt = updatedAt,
        sequenceNumber = sequenceNumber,
        previousId = previousId,
        isVoice = isVoice,
        audioPath = audioPath,
        audioDurationMs = audioDurationMs,
        localId = localId,
        serverId = serverId,
        syncStatus = syncStatus.value,
        syncedAt = syncedAt
    )
}

fun MessageResponse.toDomain(): Message {
    return Message(
        id = id,
        conversationId = conversationId,
        role = MessageRole.fromString(role),
        content = contents,
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        sequenceNumber = sequenceNumber,
        previousId = previousId,
        isVoice = false,
        serverId = id,
        syncStatus = SyncStatus.SYNCED,
        syncedAt = parseTimestamp(updatedAt)
    )
}

fun MessageResponse.toEntity(): MessageEntity {
    return MessageEntity(
        id = id,
        conversationId = conversationId,
        role = role,
        content = contents,
        createdAt = parseTimestamp(createdAt),
        updatedAt = parseTimestamp(updatedAt),
        sequenceNumber = sequenceNumber,
        previousId = previousId,
        isVoice = false,
        serverId = id,
        syncStatus = SyncStatus.SYNCED.value,
        syncedAt = parseTimestamp(updatedAt)
    )
}

fun List<MessageEntity>.toDomain(): List<Message> {
    return map { it.toDomain() }
}

fun List<MessageResponse>.toDomainFromResponse(): List<Message> {
    return map { it.toDomain() }
}

fun List<MessageResponse>.toEntities(): List<MessageEntity> {
    return map { it.toEntity() }
}
