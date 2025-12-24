package org.localforge.alicia.core.network.di

import android.content.Context
import com.squareup.moshi.Moshi
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import org.localforge.alicia.core.network.BuildConfig
import org.localforge.alicia.core.network.LiveKitManager
import org.localforge.alicia.core.network.api.AliciaApiService
import org.localforge.alicia.core.network.protocol.ProtocolHandler
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Qualifier
import javax.inject.Singleton

/**
 * Qualifier for the base URL
 */
@Qualifier
@Retention(AnnotationRetention.BINARY)
annotation class ApiBaseUrl

/**
 * Qualifier for the LiveKit URL
 */
@Qualifier
@Retention(AnnotationRetention.BINARY)
annotation class LiveKitUrl

/**
 * Hilt module for network dependencies
 */
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    /**
     * Provide the API base URL
     * This can be overridden by application configuration
     */
    @Provides
    @Singleton
    @ApiBaseUrl
    fun provideApiBaseUrl(): String {
        return BuildConfig.API_BASE_URL
    }

    /**
     * Provide the LiveKit server URL
     * This can be overridden by application configuration
     */
    @Provides
    @Singleton
    @LiveKitUrl
    fun provideLiveKitUrl(): String {
        return BuildConfig.LIVEKIT_URL
    }

    /**
     * Provide Moshi JSON converter
     */
    @Provides
    @Singleton
    fun provideMoshi(): Moshi {
        return Moshi.Builder().build()
    }

    /**
     * Provide OkHttpClient with logging interceptor
     */
    @Provides
    @Singleton
    fun provideOkHttpClient(): OkHttpClient {
        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) {
                HttpLoggingInterceptor.Level.BODY
            } else {
                HttpLoggingInterceptor.Level.NONE
            }
        }

        return OkHttpClient.Builder()
            .addInterceptor(loggingInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    /**
     * Provide Retrofit instance
     */
    @Provides
    @Singleton
    fun provideRetrofit(
        okHttpClient: OkHttpClient,
        moshi: Moshi,
        @ApiBaseUrl baseUrl: String
    ): Retrofit {
        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(okHttpClient)
            .addConverterFactory(MoshiConverterFactory.create(moshi))
            .build()
    }

    /**
     * Provide AliciaApiService
     */
    @Provides
    @Singleton
    fun provideAliciaApiService(retrofit: Retrofit): AliciaApiService {
        return retrofit.create(AliciaApiService::class.java)
    }

    /**
     * Provide ProtocolHandler for MessagePack encoding/decoding
     */
    @Provides
    @Singleton
    fun provideProtocolHandler(): ProtocolHandler {
        return ProtocolHandler()
    }

    /**
     * Provide LiveKitManager
     */
    @Provides
    @Singleton
    fun provideLiveKitManager(
        @ApplicationContext context: Context,
        protocolHandler: ProtocolHandler
    ): LiveKitManager {
        return LiveKitManager(context, protocolHandler)
    }
}
