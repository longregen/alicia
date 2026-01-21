package org.localforge.alicia.core.database.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "conversations")
data class ConversationEntity(
    @PrimaryKey
    val id: String,
    val title: String? = null,
    val status: String = "active",
    val createdAt: Long,
    val updatedAt: Long,
    val syncedAt: Long? = null,
    val isDeleted: Boolean = false
)
