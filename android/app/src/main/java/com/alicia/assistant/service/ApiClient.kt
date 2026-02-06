package com.alicia.assistant.service

import com.alicia.assistant.telemetry.AliciaTelemetry
import io.opentelemetry.instrumentation.okhttp.v3_0.OkHttpTelemetry
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.TimeUnit

object ApiClient {
    const val BASE_URL = "https://alicia.hjkl.lol"

    private const val MAX_RETRIES = 3
    private val RETRYABLE_CODES = setOf(429, 500, 502, 503)

    private val retryInterceptor = Interceptor { chain ->
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

    private val otelInterceptor: Interceptor by lazy {
        OkHttpTelemetry.builder(AliciaTelemetry.getOpenTelemetry())
            .build()
            .newInterceptor()
    }

    val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor(retryInterceptor)
            .addInterceptor(otelInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .build()
    }
}
