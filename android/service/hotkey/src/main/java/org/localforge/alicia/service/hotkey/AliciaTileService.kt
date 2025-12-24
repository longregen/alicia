package org.localforge.alicia.service.hotkey

import android.app.PendingIntent
import android.content.Intent
import android.graphics.drawable.Icon
import android.os.Handler
import android.os.Looper
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import timber.log.Timber
import dagger.hilt.android.AndroidEntryPoint

/**
 * Quick Settings tile for the Alicia voice assistant.
 *
 * Provides quick access to the voice assistant from the notification shade.
 * Tapping the tile activates the assistant and shows the current state (listening, idle, etc.)
 *
 * Features:
 * - One-tap activation from Quick Settings
 * - Visual state indicator (active/inactive)
 * - Collapses notification shade on activation
 *
 * To add the tile:
 * 1. Pull down notification shade
 * 2. Tap edit button
 * 3. Find "Alicia" tile and drag to active tiles
 */
@AndroidEntryPoint
class AliciaTileService : TileService() {

    companion object {
        private const val TAG = "AliciaTileService"

        // Tile states
        private const val STATE_INACTIVE = Tile.STATE_INACTIVE
        private const val STATE_ACTIVE = Tile.STATE_ACTIVE
        private const val STATE_UNAVAILABLE = Tile.STATE_UNAVAILABLE

        // Labels
        private const val LABEL_INACTIVE = "Alicia"
        private const val LABEL_ACTIVE = "Alicia (Active)"
        private const val LABEL_LISTENING = "Alicia (Listening)"
    }

    // Handler for delayed operations
    private val handler = Handler(Looper.getMainLooper())

    // Runnable for resetting tile state (to allow cancellation)
    private var resetTileRunnable: Runnable? = null

    override fun onCreate() {
        super.onCreate()
        Timber.d("AliciaTileService created")
    }

    override fun onTileAdded() {
        super.onTileAdded()
        Timber.i("Alicia tile added to Quick Settings")
        updateTile(STATE_INACTIVE, LABEL_INACTIVE)
    }

    override fun onTileRemoved() {
        super.onTileRemoved()
        Timber.i("Alicia tile removed from Quick Settings")
    }

    override fun onStartListening() {
        super.onStartListening()
        Timber.d("Tile is now visible, updating state")

        // Update tile state when it becomes visible in the Quick Settings panel
        updateTileBasedOnServiceState()
    }

    override fun onStopListening() {
        super.onStopListening()
        Timber.d("Tile is no longer visible")
    }

    override fun onClick() {
        super.onClick()
        Timber.i("Alicia tile clicked - activating assistant")

        // Provide visual feedback
        updateTile(STATE_ACTIVE, LABEL_ACTIVE)

        // Activate the assistant
        activateAssistant()

        // Collapse the notification shade and launch activity
        val intent = createAssistantIntent()
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )
        startActivityAndCollapse(pendingIntent)

        // Cancel any previous delayed reset
        resetTileRunnable?.let { handler.removeCallbacks(it) }

        // Reset tile state after a short delay
        resetTileRunnable = Runnable {
            updateTile(STATE_INACTIVE, LABEL_INACTIVE)
        }
        handler.postDelayed(resetTileRunnable!!, 1000)
    }

    /**
     * Activate the Alicia voice assistant
     */
    private fun activateAssistant() {
        try {
            // Send broadcast to activate assistant
            val intent = Intent(HotkeyAccessibilityService.ACTION_ACTIVATE).apply {
                setPackage(packageName)
            }
            sendBroadcast(intent)

            Timber.d("Assistant activation broadcast sent")
        } catch (e: Exception) {
            Timber.e(e, "Failed to activate assistant")
            updateTile(STATE_UNAVAILABLE, "Error")
        }
    }

    /**
     * Create intent to launch the assistant UI
     */
    private fun createAssistantIntent(): Intent {
        // Launch MainActivity with voice activation flag when tile is tapped
        val packageName = packageName
        val className = "org.localforge.alicia.MainActivity"

        return Intent().apply {
            setClassName(packageName, className)
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("activate_voice", true)
        }
    }

    /**
     * Update tile based on the current service state
     *
     * Refreshes the tile appearance when it becomes visible in the Quick Settings panel.
     * Currently defaults to inactive state.
     *
     * TODO: Integrate with VoiceService to display real-time listening or active status.
     *       See: https://github.com/localforge/alicia/issues/TODO
     */
    private fun updateTileBasedOnServiceState() {
        updateTile(STATE_INACTIVE, LABEL_INACTIVE)
    }

    /**
     * Update the tile appearance
     */
    private fun updateTile(state: Int, label: String) {
        val tile = qsTile ?: run {
            Timber.w("Tile is null, cannot update")
            return
        }

        tile.state = state
        tile.label = label

        // Set icon based on state
        // Note: Currently using the same icon for both ACTIVE and INACTIVE states.
        // This is intentional to maintain consistent visual appearance in Quick Settings.
        tile.icon = Icon.createWithResource(
            this,
            when (state) {
                STATE_ACTIVE -> android.R.drawable.ic_btn_speak_now
                STATE_INACTIVE -> android.R.drawable.ic_btn_speak_now
                else -> android.R.drawable.ic_dialog_alert
            }
        )

        tile.subtitle = when (state) {
            STATE_ACTIVE -> "Active - tap to disable"
            STATE_INACTIVE -> "Ready"
            else -> "Unavailable"
        }

        // Update the tile
        tile.updateTile()

        Timber.d("Tile updated - state: $state, label: $label")
    }

    /**
     * Public method to update tile state from external components
     * Can be called via binding to the service
     */
    fun updateState(isListening: Boolean, isActive: Boolean) {
        when {
            isListening -> updateTile(STATE_ACTIVE, LABEL_LISTENING)
            isActive -> updateTile(STATE_ACTIVE, LABEL_ACTIVE)
            else -> updateTile(STATE_INACTIVE, LABEL_INACTIVE)
        }
    }

    override fun onDestroy() {
        super.onDestroy()

        // Clean up handlers
        resetTileRunnable?.let { handler.removeCallbacks(it) }
        handler.removeCallbacksAndMessages(null)

        Timber.d("AliciaTileService destroyed")
    }
}
