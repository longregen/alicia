package org.localforge.alicia.service.voice.di

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import org.localforge.alicia.core.network.LiveKitManager
import org.localforge.alicia.service.voice.AudioManager
import org.localforge.alicia.service.voice.PowerAwareWakeWordDetector
import org.localforge.alicia.service.voice.VoiceController
import org.localforge.alicia.service.voice.WakeWordDetector
import javax.inject.Singleton

/**
 * Hilt module providing voice service dependencies.
 */
@Module
@InstallIn(SingletonComponent::class)
object VoiceServiceModule {

    @Provides
    @Singleton
    fun provideWakeWordDetector(
        @ApplicationContext context: Context
    ): WakeWordDetector {
        return WakeWordDetector(context)
    }

    @Provides
    @Singleton
    fun provideAudioManager(
        @ApplicationContext context: Context
    ): AudioManager {
        return AudioManager(context)
    }

    @Provides
    @Singleton
    fun provideVoiceController(
        @ApplicationContext context: Context,
        wakeWordDetector: WakeWordDetector,
        audioManager: AudioManager,
        liveKitManager: LiveKitManager,
        conversationRepository: org.localforge.alicia.core.domain.repository.ConversationRepository,
        settingsRepository: org.localforge.alicia.core.domain.repository.SettingsRepository
    ): VoiceController {
        return VoiceController(
            context = context,
            wakeWordDetector = wakeWordDetector,
            audioManager = audioManager,
            liveKitManager = liveKitManager,
            conversationRepository = conversationRepository,
            settingsRepository = settingsRepository
        )
    }

    @Provides
    @Singleton
    fun providePowerAwareWakeWordDetector(
        @ApplicationContext context: Context,
        wakeWordDetector: WakeWordDetector
    ): PowerAwareWakeWordDetector {
        return PowerAwareWakeWordDetector(context, wakeWordDetector)
    }
}
