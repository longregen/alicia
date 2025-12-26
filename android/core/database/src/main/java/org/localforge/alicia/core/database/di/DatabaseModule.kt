package org.localforge.alicia.core.database.di

import android.content.Context
import androidx.room.Room
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import org.localforge.alicia.core.database.AliciaDatabase
import org.localforge.alicia.core.database.dao.ConversationDao
import org.localforge.alicia.core.database.dao.MessageDao
import javax.inject.Singleton

/**
 * Hilt module for providing database dependencies.
 */
@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {

    /**
     * Provides the Room database instance.
     * This is a singleton that will be created once and reused throughout the app.
     */
    @Provides
    @Singleton
    fun provideAliciaDatabase(
        @ApplicationContext context: Context
    ): AliciaDatabase {
        return Room.databaseBuilder(
            context,
            AliciaDatabase::class.java,
            AliciaDatabase.DATABASE_NAME
        )
            // Allow destructive migration only on downgrade (version rollback scenarios).
            // This prevents data loss during app upgrades while supporting version rollbacks.
            // For production, implement proper Room migrations for schema changes.
            // See: https://developer.android.com/training/data-storage/room/migrating-db-versions
            .fallbackToDestructiveMigrationOnDowngrade(dropAllTables = true)
            .build()
    }

    /**
     * Provides the ConversationDao.
     */
    @Provides
    @Singleton
    fun provideConversationDao(database: AliciaDatabase): ConversationDao {
        return database.conversationDao()
    }

    /**
     * Provides the MessageDao.
     */
    @Provides
    @Singleton
    fun provideMessageDao(database: AliciaDatabase): MessageDao {
        return database.messageDao()
    }
}
