package com.alicia.assistant.service

import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.TimeUnit

object ApiClient {
    const val BASE_URL = "https://llm.hjkl.lol"

    private const val API_KEY = ""

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

    private val authInterceptor = Interceptor { chain ->
        val original = chain.request()
        if (API_KEY.isNotEmpty()) {
            val authorized = original.newBuilder()
                .header("Authorization", "Bearer $API_KEY")
                .build()
            chain.proceed(authorized)
        } else {
            chain.proceed(original)
        }
    }

    val client: OkHttpClient = OkHttpClient.Builder()
        .addInterceptor(retryInterceptor)
        .addInterceptor(authInterceptor)
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(60, TimeUnit.SECONDS)
        .build()
}
