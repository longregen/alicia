package com.alicia.assistant.storage

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.*
import androidx.datastore.preferences.preferencesDataStore
import com.alicia.assistant.model.AppSettings
import com.alicia.assistant.model.VoiceNote
import android.util.Log
import com.google.gson.Gson
import com.google.gson.JsonSyntaxException
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.flow.first

private val Context.dataStore: DataStore<Preferences> by preferencesDataStore(name = "alicia_prefs")

class PreferencesManager(private val context: Context) {

    companion object {
        private val gson = Gson()
        private val WAKE_WORD_ENABLED = booleanPreferencesKey("wake_word_enabled")
        private val WAKE_WORD = stringPreferencesKey("wake_word")
        private val VOICE_FEEDBACK_ENABLED = booleanPreferencesKey("voice_feedback_enabled")
        private val HAPTIC_FEEDBACK_ENABLED = booleanPreferencesKey("haptic_feedback_enabled")
        private val TTS_SPEED = floatPreferencesKey("tts_speed")
        private val VOSK_MODEL_ID = stringPreferencesKey("vosk_model_id")
        private val VOICE_NOTES = stringPreferencesKey("voice_notes")
    }
    
    suspend fun saveSettings(settings: AppSettings) {
        context.dataStore.edit { prefs ->
            prefs[WAKE_WORD_ENABLED] = settings.wakeWordEnabled
            prefs[WAKE_WORD] = settings.wakeWord
            prefs[VOICE_FEEDBACK_ENABLED] = settings.voiceFeedbackEnabled
            prefs[HAPTIC_FEEDBACK_ENABLED] = settings.hapticFeedbackEnabled
            prefs[TTS_SPEED] = settings.ttsSpeed
            prefs[VOSK_MODEL_ID] = settings.voskModelId
        }
    }
    
    suspend fun getSettings(): AppSettings {
        val prefs = context.dataStore.data.first()
        return AppSettings(
            wakeWordEnabled = prefs[WAKE_WORD_ENABLED] ?: false,
            wakeWord = prefs[WAKE_WORD] ?: "alicia",
            voiceFeedbackEnabled = prefs[VOICE_FEEDBACK_ENABLED] ?: true,
            hapticFeedbackEnabled = prefs[HAPTIC_FEEDBACK_ENABLED] ?: true,
            ttsSpeed = prefs[TTS_SPEED] ?: 1.5f,
            voskModelId = prefs[VOSK_MODEL_ID] ?: "small-en-us"
        )
    }

    suspend fun getLegacyNotes(): List<VoiceNote> {
        val prefs = context.dataStore.data.first()
        val json = prefs[VOICE_NOTES] ?: return emptyList()
        return try {
            val type = object : TypeToken<List<VoiceNote>>() {}.type
            gson.fromJson<List<VoiceNote>>(json, type) ?: emptyList()
        } catch (e: JsonSyntaxException) {
            Log.e("PreferencesManager", "Failed to parse voice notes JSON", e)
            emptyList()
        }
    }
    
    suspend fun clearLegacyNotes() {
        context.dataStore.edit { prefs ->
            prefs.remove(VOICE_NOTES)
        }
    }
}
