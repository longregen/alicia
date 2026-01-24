package com.alicia.assistant.skills

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.AlarmClock
import android.util.Log
import com.alicia.assistant.service.AliciaApiClient
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

data class SkillResult(
    val success: Boolean,
    val response: String,
    val action: String? = null
)

class SkillRouter(
    private val context: Context,
    private val apiClient: AliciaApiClient = AliciaApiClient(AliciaApiClient.BASE_URL, AliciaApiClient.USER_ID)
) {
    companion object {
        private const val TAG = "SkillRouter"
        private val NOTE_PATTERN = Regex(".*(save|make|create|leave)\\s+(a\\s+)?(voice\\s+)?note.*")
        private val REMIND_PATTERN = Regex(".*(remind|remember)\\s+me.*")
        private val APP_LAUNCH_PATTERN = Regex("^(open|launch|start|go to|switch to)\\s+.*")
        private val TIMER_PATTERN = Regex(".*(set|start|create)\\s+(a\\s+)?(timer|alarm).*")
        private val TIMER_START_PATTERN = Regex("^(timer|alarm).*")
        private val TIME_QUERY_PATTERN = Regex(".*(what('s|\\s+is)?\\s+the\\s+time|tell.*time|current\\s+time).*")
        private val DATE_QUERY_PATTERN = Regex(".*(what('s|\\s+is)?\\s+(the\\s+)?(date|day)|today('s|\\s+is)).*")
        private val APP_NAME_PATTERN = Regex("(open|launch|start|go to|switch to)\\s+(the\\s+)?", RegexOption.IGNORE_CASE)
    }

    // Lazy conversation for voice interactions (ephemeral, one per app session)
    private var voiceConversationId: String? = null

    suspend fun processInput(input: String, screenContext: String? = null): SkillResult {
        val lowerInput = input.lowercase().trim()

        return when {
            lowerInput.matches(NOTE_PATTERN) ||
            lowerInput.matches(REMIND_PATTERN) -> handleVoiceNote(input)

            lowerInput.matches(APP_LAUNCH_PATTERN) -> handleAppLauncher(input)

            lowerInput.contains("play") && lowerInput.contains("music") -> handleMusicControl(input)

            lowerInput.matches(TIMER_PATTERN) ||
            lowerInput.matches(TIMER_START_PATTERN) -> handleTimer(input)

            lowerInput.matches(TIME_QUERY_PATTERN) -> handleTimeQuery()

            lowerInput.matches(DATE_QUERY_PATTERN) -> handleDateQuery()

            else -> handleApiChat(input, screenContext)
        }
    }

    private suspend fun handleApiChat(input: String, screenContext: String?): SkillResult {
        return try {
            val convId = getOrCreateVoiceConversation()
            val content = if (screenContext != null) {
                "[Screen content]\n$screenContext\n[End screen content]\n\nUser: $input"
            } else {
                input
            }
            val response = apiClient.sendMessageSync(convId, content)
            SkillResult(
                success = true,
                response = response.assistantMessage.content,
                action = "api_chat"
            )
        } catch (e: Exception) {
            Log.e(TAG, "API chat failed", e)
            SkillResult(
                success = false,
                response = "Sorry, I couldn't get a response right now."
            )
        }
    }

    private suspend fun getOrCreateVoiceConversation(): String {
        voiceConversationId?.let { return it }
        val conversation = apiClient.createConversation("Voice Chat")
        voiceConversationId = conversation.id
        return conversation.id
    }
    
    private fun handleVoiceNote(input: String): SkillResult {
        return SkillResult(
            success = true,
            response = "To save a voice note, use the note recording button on the main screen.",
            action = "note_hint"
        )
    }
    
    private fun handleAppLauncher(input: String): SkillResult {
        val appName = input
            .replace(APP_NAME_PATTERN, "")
            .trim()
            .lowercase()
        
        val appPackages = mapOf(
            "spotify" to "com.spotify.music",
            "youtube" to "com.google.android.youtube",
            "instagram" to "com.instagram.android",
            "twitter" to "com.twitter.android",
            "facebook" to "com.facebook.katana",
            "gmail" to "com.google.android.gm",
            "chrome" to "com.android.chrome",
            "maps" to "com.google.android.apps.maps",
            "whatsapp" to "com.whatsapp",
            "telegram" to "org.telegram.messenger"
        )
        
        val match = appPackages.entries.firstOrNull { (key, _) ->
            appName.contains(key) || key.contains(appName)
        }
        val packageName = match?.value
        val displayName = match?.key ?: appName

        return if (packageName != null) {
            try {
                val intent = context.packageManager.getLaunchIntentForPackage(packageName)
                if (intent != null) {
                    intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    context.startActivity(intent)
                    SkillResult(
                        success = true,
                        response = "Opening $displayName",
                        action = "open_app"
                    )
                } else {
                    SkillResult(
                        success = false,
                        response = "$displayName is not installed on your device"
                    )
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to open app: $appName", e)
                SkillResult(
                    success = false,
                    response = "Failed to open $displayName"
                )
            }
        } else {
            SkillResult(
                success = false,
                response = "I don't know how to open $appName"
            )
        }
    }
    
    private fun handleMusicControl(input: String): SkillResult {
        return try {
            val intent = context.packageManager.getLaunchIntentForPackage("com.spotify.music")
            if (intent != null) {
                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                context.startActivity(intent)
                SkillResult(
                    success = true,
                    response = "Opening Spotify to play music",
                    action = "play_music"
                )
            } else {
                val musicIntent = Intent(Intent.ACTION_VIEW).apply {
                    data = Uri.parse("music://")
                    addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                }
                context.startActivity(musicIntent)
                SkillResult(
                    success = true,
                    response = "Opening music app",
                    action = "play_music"
                )
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to open music app", e)
            SkillResult(
                success = false,
                response = "Please open your music app manually"
            )
        }
    }
    
    private fun handleTimer(input: String): SkillResult {
        return try {
            val intent = Intent(AlarmClock.ACTION_SET_TIMER).apply {
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            context.startActivity(intent)
            SkillResult(
                success = true,
                response = "Opening clock app to set a timer",
                action = "set_timer"
            )
        } catch (e: Exception) {
            Log.e(TAG, "Failed to set timer", e)
            SkillResult(
                success = false,
                response = "Please open your clock app to set a timer"
            )
        }
    }
    
    private fun handleTimeQuery(): SkillResult {
        val currentTime = SimpleDateFormat("h:mm a", Locale.US)
            .format(Date())
        return SkillResult(
            success = true,
            response = "The current time is $currentTime",
            action = "time_query"
        )
    }
    
    private fun handleDateQuery(): SkillResult {
        val currentDate = SimpleDateFormat("EEEE, MMMM d, yyyy", Locale.US)
            .format(Date())
        return SkillResult(
            success = true,
            response = "Today is $currentDate",
            action = "date_query"
        )
    }

}
