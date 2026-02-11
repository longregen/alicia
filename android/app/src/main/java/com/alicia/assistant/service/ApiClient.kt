package com.alicia.assistant.service

import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.instrumentation.okhttp.v3_0.OkHttpTelemetry
import okhttp3.Call
import okhttp3.OkHttpClient
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.TimeUnit

object ApiClient {
    private const val DEFAULT_BASE_URL = "https://alicia.hjkl.lol"

    @Volatile
    var baseUrlOverride: String? = null

    val BASE_URL: String
        get() = baseUrlOverride ?: DEFAULT_BASE_URL

    private const val MAX_RETRIES = 3
    private val RETRYABLE_CODES = setOf(429, 500, 502, 503)

    private val retryInterceptor = okhttp3.Interceptor { chain ->
        val request = chain.request()
        var response: Response? = null
        var lastException: IOException? = null

        for (attempt in 0..MAX_RETRIES) {
            try {
                response?.close()
                response = chain.proceed(request)
                if (response.code !in RETRYABLE_CODES || attempt == MAX_RETRIES) {
                    return@Interceptor response
                }
            } catch (e: IOException) {
                lastException = e
                if (attempt == MAX_RETRIES) throw e
            }
            val backoffMs = (1000L * (1 shl attempt))
            Thread.sleep(backoffMs)
        }

        response ?: throw lastException ?: IOException("Retry failed")
    }

    val httpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor(retryInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .build()
    }

    val client: Call.Factory by lazy {
        OkHttpTelemetry.builder(AliciaTelemetry.getOpenTelemetry())
            .build()
            .createCallFactory(httpClient)
    }

    fun createCallFactory(baseClient: OkHttpClient): Call.Factory {
        return OkHttpTelemetry.builder(AliciaTelemetry.getOpenTelemetry())
            .build()
            .createCallFactory(baseClient)
    }
}
