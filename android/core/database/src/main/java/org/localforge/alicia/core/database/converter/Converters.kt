package org.localforge.alicia.core.database.converter

import androidx.room.TypeConverter

/**
 * Type converters for Room database.
 * These allow Room to store complex types in SQLite.
 */
class Converters {

    /**
     * Convert a comma-separated string to a list of strings.
     * Returns an empty list for null or empty input strings.
     */
    @TypeConverter
    fun fromString(value: String?): List<String> {
        return if (value.isNullOrEmpty()) emptyList() else value.split(",")
    }

    /**
     * Convert a list of strings to a comma-separated string.
     */
    @TypeConverter
    fun toString(list: List<String>?): String {
        return list?.joinToString(",") ?: ""
    }
}
