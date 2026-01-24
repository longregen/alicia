package com.alicia.assistant.service

import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.service.voice.VoiceInteractionService
import android.support.v4.media.session.MediaSessionCompat
import android.support.v4.media.session.PlaybackStateCompat
import android.util.Log
import android.view.KeyEvent

class AliciaInteractionService : VoiceInteractionService() {

    companion object {
        private const val TAG = "AliciaVIS"
        private const val SHOW_WITH_ASSIST = 1
        private const val SHOW_WITH_SCREENSHOT = 2

        private var instance: AliciaInteractionService? = null

        fun triggerAssistSession() {
            instance?.triggerSession() ?: Log.w(TAG, "Service not active, cannot trigger session")
        }
    }

    private var mediaSession: MediaSessionCompat? = null

    override fun onReady() {
        super.onReady()
        instance = this
        setupMediaSession()
        Log.d(TAG, "VoiceInteractionService ready")
    }

    private fun setupMediaSession() {
        mediaSession = MediaSessionCompat(this, "AliciaAssistant").apply {
            setCallback(object : MediaSessionCompat.Callback() {
                override fun onMediaButtonEvent(mediaButtonEvent: Intent): Boolean {
                    val keyEvent = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                        mediaButtonEvent.getParcelableExtra(Intent.EXTRA_KEY_EVENT, KeyEvent::class.java)
                    } else {
                        @Suppress("DEPRECATION")
                        mediaButtonEvent.getParcelableExtra(Intent.EXTRA_KEY_EVENT)
                    } ?: return super.onMediaButtonEvent(mediaButtonEvent)

                    if (keyEvent.action == KeyEvent.ACTION_UP) {
                        when (keyEvent.keyCode) {
                            KeyEvent.KEYCODE_HEADSETHOOK,
                            KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE -> {
                                triggerSession()
                                return true
                            }
                        }
                    }
                    return super.onMediaButtonEvent(mediaButtonEvent)
                }
            })

            val state = PlaybackStateCompat.Builder()
                .setActions(PlaybackStateCompat.ACTION_PLAY_PAUSE)
                .setState(PlaybackStateCompat.STATE_STOPPED, 0, 0f)
                .build()
            setPlaybackState(state)
            isActive = true
        }
    }

    private fun triggerSession() {
        Log.d(TAG, "Triggering voice interaction session")
        showSession(Bundle(), SHOW_WITH_ASSIST or SHOW_WITH_SCREENSHOT)
    }

    override fun onShutdown() {
        instance = null
        mediaSession?.apply {
            isActive = false
            release()
        }
        mediaSession = null
        super.onShutdown()
    }
}
