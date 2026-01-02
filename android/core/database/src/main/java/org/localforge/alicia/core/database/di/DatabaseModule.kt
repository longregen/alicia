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
import org.localforge.alicia.core.database.dao.SyncQueueDao
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
            // WARNING: DEVELOPMENT MODE - This will DELETE ALL DATA on ANY schema change!
            // This configuration destroys the database on both upgrades AND downgrades.
            // It allows rapid schema iteration during development without writing migrations.
            //
            // BEFORE PRODUCTION RELEASE:
            // 1. Remove .fallbackToDestructiveMigration()
            // 2. Implement proper Room Migration objects for all schema versions
            // 3. Test upgrade paths from all previous versions
            // 4. Consider exportSchema = true in @Database for migration validation
            //
            // See: https://developer.android.com/training/data-storage/room/migrating-db-versions
            .fallbackToDestructiveMigration(dropAllTables = true)
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

    /**
     * Provides the SyncQueueDao.
     */
    @Provides
    @Singleton
    fun provideSyncQueueDao(database: AliciaDatabase): SyncQueueDao {
        return database.syncQueueDao()
    }
}
