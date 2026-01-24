package com.alicia.assistant.model

data class TimestampedWord(
    val word: String,
    val start: Float,
    val end: Float
)

data class VerboseTranscription(
    val text: String,
    val words: List<TimestampedWord>,
    val durationMs: Int
)

data class VoiceNote(
    val id: String,
    val title: String,
    val content: String,
    val timestamp: Long,
    val duration: Int = 0,
    val audioPath: String? = null,
    val words: List<TimestampedWord> = emptyList()
)

data class AppSettings(
    val wakeWordEnabled: Boolean = false,
    val wakeWord: String = "alicia",
    val voiceFeedbackEnabled: Boolean = true,
    val hapticFeedbackEnabled: Boolean = true,
    val ttsSpeed: Float = 1.5f,
    val voskModelId: String = "small-en-us"
)

enum class VoskModelInfo(
    val id: String,
    val displayName: String,
    val url: String,
    val sizeMb: Int
) {
    SMALL_EN_US("small-en-us", "English (Small, 40MB)", "https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip", 40),
    EN_US_LGRAPH("en-us-lgraph", "English (Medium, 128MB)", "https://alphacephei.com/vosk/models/vosk-model-en-us-0.22-lgraph.zip", 128),
    SMALL_ES("small-es", "Spanish (Small, 39MB)", "https://alphacephei.com/vosk/models/vosk-model-small-es-0.42.zip", 39),
    SMALL_FR("small-fr", "French (Small, 41MB)", "https://alphacephei.com/vosk/models/vosk-model-small-fr-0.22.zip", 41),
    SMALL_DE("small-de", "German (Small, 45MB)", "https://alphacephei.com/vosk/models/vosk-model-small-de-0.15.zip", 45),
    SMALL_JA("small-ja", "Japanese (Small, 48MB)", "https://alphacephei.com/vosk/models/vosk-model-small-ja-0.22.zip", 48),
    SMALL_CN("small-cn", "Chinese (Small, 42MB)", "https://alphacephei.com/vosk/models/vosk-model-small-cn-0.22.zip", 42);

    companion object {
        fun fromId(id: String): VoskModelInfo = entries.find { it.id == id } ?: SMALL_EN_US
    }
}
