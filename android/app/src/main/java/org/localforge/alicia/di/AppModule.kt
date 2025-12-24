package org.localforge.alicia.di

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

/**
 * Main Hilt Module for Application-level Dependencies
 *
 * Provides application-wide singleton dependencies that are shared
 * across the entire app lifecycle.
 */
@Module
@InstallIn(SingletonComponent::class)
object AppModule
