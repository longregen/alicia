package org.localforge.alicia.core.data.di

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import org.localforge.alicia.core.data.preferences.SettingsDataStore
import org.localforge.alicia.core.data.preferences.settingsDataStore
import org.localforge.alicia.core.data.repository.ConversationRepositoryImpl
import org.localforge.alicia.core.data.repository.MCPRepositoryImpl
import org.localforge.alicia.core.data.repository.MemoryRepositoryImpl
import org.localforge.alicia.core.data.repository.NotesRepositoryImpl
import org.localforge.alicia.core.data.repository.ServerRepositoryImpl
import org.localforge.alicia.core.data.repository.SettingsRepositoryImpl
import org.localforge.alicia.core.data.repository.VoiceRepositoryImpl
import org.localforge.alicia.core.data.repository.VotingRepositoryImpl
import org.localforge.alicia.core.database.dao.ConversationDao
import org.localforge.alicia.core.database.dao.MessageDao
import org.localforge.alicia.core.domain.repository.ConversationRepository
import org.localforge.alicia.core.domain.repository.MCPRepository
import org.localforge.alicia.core.domain.repository.MemoryRepository
import org.localforge.alicia.core.domain.repository.NotesRepository
import org.localforge.alicia.core.domain.repository.ServerRepository
import org.localforge.alicia.core.domain.repository.SettingsRepository
import org.localforge.alicia.core.domain.repository.VoiceRepository
import org.localforge.alicia.core.domain.repository.VotingRepository
import org.localforge.alicia.core.network.api.AliciaApiService
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DataModule {

    @Provides
    @Singleton
    fun provideDataStore(
        @ApplicationContext context: Context
    ): DataStore<Preferences> {
        return context.settingsDataStore
    }

    @Provides
    @Singleton
    fun provideSettingsDataStore(
        dataStore: DataStore<Preferences>
    ): SettingsDataStore {
        return SettingsDataStore(dataStore)
    }

    @Provides
    @Singleton
    fun provideConversationRepository(
        @ApplicationContext context: Context,
        conversationDao: ConversationDao,
        messageDao: MessageDao,
        apiService: AliciaApiService
    ): ConversationRepository {
        return ConversationRepositoryImpl(
            context = context,
            conversationDao = conversationDao,
            messageDao = messageDao,
            apiService = apiService
        )
    }

    @Provides
    @Singleton
    fun provideSettingsRepository(
        settingsDataStore: SettingsDataStore
    ): SettingsRepository {
        return SettingsRepositoryImpl(settingsDataStore)
    }

    @Provides
    @Singleton
    fun provideVoiceRepository(
        apiService: AliciaApiService,
        settingsDataStore: SettingsDataStore
    ): VoiceRepository {
        return VoiceRepositoryImpl(
            apiService = apiService,
            settingsDataStore = settingsDataStore
        )
    }

    @Provides
    @Singleton
    fun provideMCPRepository(
        apiService: AliciaApiService
    ): MCPRepository {
        return MCPRepositoryImpl(apiService)
    }

    @Provides
    @Singleton
    fun provideMemoryRepository(
        apiService: AliciaApiService
    ): MemoryRepository {
        return MemoryRepositoryImpl(apiService)
    }

    @Provides
    @Singleton
    fun provideServerRepository(
        apiService: AliciaApiService
    ): ServerRepository {
        return ServerRepositoryImpl(apiService)
    }

    @Provides
    @Singleton
    fun provideVotingRepository(
        apiService: AliciaApiService
    ): VotingRepository {
        return VotingRepositoryImpl(apiService)
    }

    @Provides
    @Singleton
    fun provideNotesRepository(
        apiService: AliciaApiService
    ): NotesRepository {
        return NotesRepositoryImpl(apiService)
    }
}
