package org.localforge.alicia.core.database.converter

import androidx.room.TypeConverter

class Converters {

    @TypeConverter
    fun fromString(value: String?): List<String> {
        return if (value.isNullOrEmpty()) emptyList() else value.split(",")
    }

    @TypeConverter
    fun toString(list: List<String>?): String {
        return list?.joinToString(",") ?: ""
    }
}
