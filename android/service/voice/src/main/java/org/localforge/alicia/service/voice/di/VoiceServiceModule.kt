package org.localforge.alicia.service.voice.di

import dagger.Module
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent

/**
 * Hilt module for voice service dependencies.
 * All dependencies are auto-provided via @Inject constructors on:
 * - AudioManager
 * - WakeWordDetector
 * - VoiceController
 * - PowerAwareWakeWordDetector
 */
@Module
@InstallIn(SingletonComponent::class)
object VoiceServiceModule
