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

    abstract fun conversationDao(): ConversationDao

    abstract fun messageDao(): MessageDao

    abstract fun syncQueueDao(): SyncQueueDao

    companion object {
        const val DATABASE_NAME = "alicia_database"
    }
}
