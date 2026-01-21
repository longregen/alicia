package org.localforge.alicia.core.common

import java.time.Instant

fun parseTimestamp(timestamp: String): Long {
    return try {
        Instant.parse(timestamp).toEpochMilli()
    } catch (e: Exception) {
        // Fallback to current time for malformed external timestamps to prevent crashes,
        // though this may result in incorrect timestamp ordering in edge cases
        System.currentTimeMillis()
    }
}
