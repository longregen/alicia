package org.localforge.alicia.core.data.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import org.localforge.alicia.core.data.sync.SyncManager
import org.localforge.alicia.core.domain.repository.ConversationRepository
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object SyncModule {

    @Provides
    @Singleton
    fun provideSyncManager(
        conversationRepository: ConversationRepository
    ): SyncManager {
        return SyncManager(conversationRepository)
    }
}
