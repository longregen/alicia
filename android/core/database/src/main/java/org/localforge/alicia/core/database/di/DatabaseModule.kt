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

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {

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
            // WARNING: fallbackToDestructiveMigration() deletes ALL data on schema changes.
            // Before production: remove this, implement proper Migration objects, and set exportSchema = true.
            .fallbackToDestructiveMigration()
            .build()
    }

    @Provides
    @Singleton
    fun provideConversationDao(database: AliciaDatabase): ConversationDao {
        return database.conversationDao()
    }

    @Provides
    @Singleton
    fun provideMessageDao(database: AliciaDatabase): MessageDao {
        return database.messageDao()
    }

    @Provides
    @Singleton
    fun provideSyncQueueDao(database: AliciaDatabase): SyncQueueDao {
        return database.syncQueueDao()
    }
}
