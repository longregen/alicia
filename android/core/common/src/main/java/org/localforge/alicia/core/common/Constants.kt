package org.localforge.alicia.core.common

object Constants {

    object Actions {
        const val ACTION_ACTIVATE = "org.localforge.alicia.ACTION_ACTIVATE_ASSISTANT"
        const val ACTION_START_WAKE_WORD = "org.localforge.alicia.ACTION_START_WAKE_WORD"
        const val ACTION_STOP_WAKE_WORD = "org.localforge.alicia.ACTION_STOP_WAKE_WORD"
        const val ACTION_START_FLOATING_BUTTON = "org.localforge.alicia.service.hotkey.ACTION_START_FLOATING_BUTTON"
        const val ACTION_STOP_FLOATING_BUTTON = "org.localforge.alicia.service.hotkey.ACTION_STOP_FLOATING_BUTTON"
    }

    object Notifications {
        const val VOICE_SERVICE_ID = 1001
        const val FLOATING_BUTTON_ID = 2001
        const val HOTKEY_SERVICE_ID = 3001
    }

    object Channels {
        const val VOICE_SERVICE = "alicia_voice_service"
        const val FLOATING_BUTTON = "alicia_floating_button"
        const val HOTKEY_SERVICE = "alicia_hotkey_service"
        const val GENERAL = "alicia_general"
    }

    object Preferences {
        const val HOTKEY_VOLUME_UP_ENABLED = "hotkey_volume_up_enabled"
        const val HOTKEY_VOLUME_DOWN_ENABLED = "hotkey_volume_down_enabled"
        const val HOTKEY_POWER_BUTTON_ENABLED = "hotkey_power_button_enabled"
        const val HOTKEY_SHAKE_ENABLED = "hotkey_shake_enabled"
        const val HOTKEY_TAP_COUNT = "hotkey_tap_count"

        const val AUTO_START_ENABLED = "auto_start_enabled"
        const val FLOATING_BUTTON_ENABLED = "floating_button_enabled"
        const val FLOATING_BUTTON_AUTO_START = "floating_button_auto_start"
        const val WAKE_WORD_ENABLED = "wake_word_enabled"
        const val WAKE_WORD_AUTO_START = "wake_word_auto_start"

        const val WAKE_WORD_SELECTION = "wake_word_selection"
        const val WAKE_WORD_SENSITIVITY = "wake_word_sensitivity"
        const val SPEECH_RATE = "speech_rate"
        const val SELECTED_VOICE = "selected_voice"

        const val SERVER_URL = "server_url"
        const val LIVEKIT_URL = "livekit_url"

        const val SAVE_CONVERSATION_HISTORY = "save_conversation_history"

        const val IS_FIRST_RUN = "is_first_run"
        const val ONBOARDING_COMPLETED = "onboarding_completed"
    }

    object Defaults {
        const val WAKE_WORD_SENSITIVITY = 0.7f
        const val SPEECH_RATE = 1.0f
        const val HOTKEY_TAP_COUNT = 3
        const val SHAKE_THRESHOLD = 12.0f
        const val AUTO_START_ENABLED = false
        const val SAVE_CONVERSATION_HISTORY = true
    }

    object Timing {
        const val VOLUME_TAP_INTERVAL_MS = 500L
        const val POWER_LONG_PRESS_MS = 1000L
        const val SHAKE_INTERVAL_MS = 500L
        const val SHAKE_COOLDOWN_MS = 2000L
        const val CLICK_THRESHOLD_MS = 200L
        const val LONG_PRESS_THRESHOLD_MS = 500L
    }

    object WakeWords {
        const val ALICIA = "alicia"
        const val HEY_ALICIA = "hey_alicia"
        const val JARVIS = "jarvis"
        const val COMPUTER = "computer"
    }

    object ErrorCodes {
        const val PERMISSION_DENIED = 1001
        const val SERVICE_UNAVAILABLE = 1002
        const val NETWORK_ERROR = 1003
        const val MICROPHONE_ERROR = 1004
        const val ACCESSIBILITY_SERVICE_DISABLED = 1005
        const val OVERLAY_PERMISSION_DENIED = 1006
    }

    object Tags {
        const val HOTKEY = "HotkeyService"
        const val VOICE = "VoiceService"
        const val SHAKE = "ShakeDetector"
        const val FLOATING_BUTTON = "FloatingButton"
        const val TILE = "TileService"
        const val BOOT = "BootReceiver"
        const val PERMISSION = "PermissionManager"
    }
}
