package org.localforge.alicia.core.database

import androidx.room.Database
import androidx.room.RoomDatabase
import androidx.room.TypeConverters
import org.localforge.alicia.core.database.converter.Converters
import org.localforge.alicia.core.database.dao.ConversationDao
import org.localforge.alicia.core.database.dao.MessageDao
import org.localforge.alicia.core.database.dao.SyncQueueDao
import org.localforge.alicia.core.database.entity.ConversationEntity
import org.localforge.alicia.core.database.entity.MessageEntity
import org.localforge.alicia.core.database.entity.SyncQueueEntity

/**
 * Room database for the Alicia voice assistant.
 * Stores conversations and messages locally for offline access and caching.
 */
@Database(
    entities = [
        ConversationEntity::class,
        MessageEntity::class,
        SyncQueueEntity::class
    ],
    version = 2,
    exportSchema = false
)
@TypeConverters(Converters::class)
abstract class AliciaDatabase : RoomDatabase() {

    /**
     * DAO for conversation operations.
     */
    abstract fun conversationDao(): ConversationDao

    /**
     * DAO for message operations.
     */
    abstract fun messageDao(): MessageDao

    /**
     * DAO for sync queue operations.
     */
    abstract fun syncQueueDao(): SyncQueueDao

    companion object {
        const val DATABASE_NAME = "alicia_database"
    }
}
