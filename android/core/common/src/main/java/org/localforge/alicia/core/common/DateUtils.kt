package org.localforge.alicia.core.common

import java.time.Instant

/**
 * Utility functions for date and time operations
 */

/**
 * Parse ISO timestamp string to milliseconds since epoch.
 * Falls back to current time if parsing fails.
 *
 * @param timestamp ISO-8601 formatted timestamp string
 * @return Milliseconds since epoch
 */
fun parseTimestamp(timestamp: String): Long {
    return try {
        Instant.parse(timestamp).toEpochMilli()
    } catch (e: Exception) {
        // Silent fallback to current time to ensure graceful degradation when parsing malformed
        // timestamps from external sources. This prevents app crashes while maintaining
        // functionality, though it may result in incorrect timestamp ordering in edge cases.
        System.currentTimeMillis()
    }
}
