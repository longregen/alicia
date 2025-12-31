package org.localforge.alicia.service.hotkey

import android.animation.ValueAnimator
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.graphics.PixelFormat
import android.os.IBinder
import timber.log.Timber
import android.view.Gravity
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import android.view.animation.DecelerateInterpolator
import androidx.core.app.NotificationCompat
import dagger.hilt.android.AndroidEntryPoint
import kotlin.math.abs

/**
 * Foreground service that displays a draggable floating button overlay
 * for quick access to the Alicia voice assistant.
 *
 * Features:
 * - Draggable button that can be positioned anywhere on screen
 * - Click to activate assistant
 * - Long-press for quick menu (not yet implemented)
 * - Auto-snap to screen edges
 * - Runs as a foreground service with persistent notification
 */
@AndroidEntryPoint
class FloatingButtonService : Service() {

    companion object {
        private const val TAG = "FloatingButtonService"

        const val ACTION_START = "org.localforge.alicia.service.hotkey.ACTION_START_FLOATING_BUTTON"
        const val ACTION_STOP = "org.localforge.alicia.service.hotkey.ACTION_STOP_FLOATING_BUTTON"

        private const val NOTIFICATION_ID = 2001
        private const val NOTIFICATION_CHANNEL_ID = "alicia_floating_button"
        private const val NOTIFICATION_CHANNEL_NAME = "Floating Button"

        // Touch handling
        private const val CLICK_THRESHOLD_MS = 200L
        private const val CLICK_DISTANCE_THRESHOLD = 10 // pixels
        private const val LONG_PRESS_THRESHOLD_MS = 500L

        // Animation
        private const val SNAP_ANIMATION_DURATION = 300L
    }

    private var windowManager: WindowManager? = null
    private var floatingView: View? = null
    private var layoutParams: WindowManager.LayoutParams? = null

    // Lock for synchronizing view layout updates
    private val layoutUpdateLock = Any()

    // Touch event tracking
    private var initialX = 0
    private var initialY = 0
    private var initialTouchX = 0f
    private var initialTouchY = 0f
    private var touchDownTime = 0L
    private var isDragging = false

    override fun onCreate() {
        super.onCreate()
        Timber.d("FloatingButtonService created")

        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())

        windowManager = getSystemService(WINDOW_SERVICE) as WindowManager
        setupFloatingButton()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> {
                Timber.i("Starting floating button")
                showFloatingButton()
            }
            ACTION_STOP -> {
                Timber.i("Stopping floating button")
                stopSelf()
            }
        }
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    /**
     * Set up the floating button view and its touch handling
     */
    @SuppressLint("InflateParams")
    private fun setupFloatingButton() {
        // Inflate the floating button layout for overlay window (root parameter omitted)
        floatingView = LayoutInflater.from(this).inflate(
            R.layout.floating_button,
            null
        )

        // Configure window layout parameters
        layoutParams = WindowManager.LayoutParams(
            WindowManager.LayoutParams.WRAP_CONTENT,
            WindowManager.LayoutParams.WRAP_CONTENT,
            WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY,
            WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or
                    WindowManager.LayoutParams.FLAG_LAYOUT_NO_LIMITS,
            PixelFormat.TRANSLUCENT
        ).apply {
            gravity = Gravity.TOP or Gravity.START
            x = getScreenWidth() - 100 // Start at right edge
            y = getScreenHeight() / 2 // Middle of screen
        }

        // Set up touch listener for drag and click handling
        floatingView?.setOnTouchListener(::handleTouch)
    }

    /**
     * Show the floating button
     */
    private fun showFloatingButton() {
        if (floatingView?.parent == null) {
            try {
                windowManager?.addView(floatingView, layoutParams)
                Timber.d("Floating button added to window")
            } catch (e: Exception) {
                Timber.e(e, "Failed to add floating button")
            }
        }
    }

    /**
     * Hide the floating button
     */
    private fun hideFloatingButton() {
        if (floatingView?.parent != null) {
            try {
                windowManager?.removeView(floatingView)
                Timber.d("Floating button removed from window")
            } catch (e: Exception) {
                Timber.e(e, "Failed to remove floating button")
            }
        }
    }

    /**
     * Handle touch events for dragging and clicking
     */
    private fun handleTouch(view: View, event: MotionEvent): Boolean {
        when (event.action) {
            MotionEvent.ACTION_DOWN -> {
                // Record initial positions
                initialX = layoutParams?.x ?: 0
                initialY = layoutParams?.y ?: 0
                initialTouchX = event.rawX
                initialTouchY = event.rawY
                touchDownTime = System.currentTimeMillis()
                isDragging = false

                // Visual feedback - scale down slightly
                view.animate()
                    .scaleX(0.9f)
                    .scaleY(0.9f)
                    .setDuration(100)
                    .start()

                return true
            }

            MotionEvent.ACTION_MOVE -> {
                // Calculate movement
                val deltaX = event.rawX - initialTouchX
                val deltaY = event.rawY - initialTouchY

                // Check if moved enough to be considered dragging
                if (!isDragging && (abs(deltaX) > CLICK_DISTANCE_THRESHOLD || abs(deltaY) > CLICK_DISTANCE_THRESHOLD)) {
                    isDragging = true
                }

                // Update position if dragging
                if (isDragging) {
                    synchronized(layoutUpdateLock) {
                        layoutParams?.x = (initialX + deltaX).toInt()
                        layoutParams?.y = (initialY + deltaY).toInt()
                        windowManager?.updateViewLayout(floatingView, layoutParams)
                    }
                }

                return true
            }

            MotionEvent.ACTION_UP -> {
                // Restore scale
                view.animate()
                    .scaleX(1.0f)
                    .scaleY(1.0f)
                    .setDuration(100)
                    .start()

                val pressDuration = System.currentTimeMillis() - touchDownTime

                when {
                    !isDragging && pressDuration < CLICK_THRESHOLD_MS -> {
                        // Short tap - activate assistant
                        view.performClick()
                        handleClick()
                    }
                    !isDragging && pressDuration >= LONG_PRESS_THRESHOLD_MS -> {
                        // Long press - show menu
                        handleLongPress()
                    }
                    isDragging -> {
                        // Drag ended - snap to edge
                        snapToEdge()
                    }
                }

                return true
            }

            else -> return false
        }
    }

    /**
     * Handle button click - activate assistant
     */
    private fun handleClick() {
        Timber.i("Floating button clicked - activating assistant")

        // Send activation intent
        val intent = Intent(HotkeyAccessibilityService.ACTION_ACTIVATE).apply {
            setPackage(packageName)
        }
        sendBroadcast(intent)

        // Visual feedback
        floatingView?.animate()
            ?.scaleX(1.2f)
            ?.scaleY(1.2f)
            ?.setDuration(100)
            ?.withEndAction {
                floatingView?.animate()
                    ?.scaleX(1.0f)
                    ?.scaleY(1.0f)
                    ?.setDuration(100)
                    ?.start()
            }
            ?.start()
    }

    /**
     * Handle long press - show options menu
     *
     * TODO: Implement quick menu with options for dismiss, settings, and position adjustment.
     *       Currently only logs the long press event.
     */
    private fun handleLongPress() {
        Timber.d("Floating button long pressed")
    }

    /**
     * Snap button to nearest screen edge
     */
    private fun snapToEdge() {
        val screenWidth = getScreenWidth()
        val currentX = layoutParams?.x ?: 0

        // Snap to nearest screen edge. coerceAtLeast(0) ensures button stays within
        // screen bounds even with unusual screen/button size ratios.
        val targetX = if (currentX < screenWidth / 2) {
            0 // Snap to left
        } else {
            // Snap to right, ensuring we don't go off screen
            val buttonWidth = floatingView?.width ?: 0
            (screenWidth - buttonWidth).coerceAtLeast(0)
        }

        // Animate to target position
        val animator = ValueAnimator.ofInt(currentX, targetX)
        animator.addUpdateListener { animation ->
            synchronized(layoutUpdateLock) {
                layoutParams?.x = animation.animatedValue as Int
                windowManager?.updateViewLayout(floatingView, layoutParams)
            }
        }
        animator.interpolator = DecelerateInterpolator()
        animator.duration = SNAP_ANIMATION_DURATION
        animator.start()

        Timber.d("Snapping button from x=$currentX to x=$targetX")
    }

    /**
     * Get screen width
     */
    private fun getScreenWidth(): Int {
        return getScreenDimension(isWidth = true)
    }

    /**
     * Get screen height
     */
    private fun getScreenHeight(): Int {
        return getScreenDimension(isWidth = false)
    }

    /**
     * Get screen dimension (width or height) using WindowMetrics API
     */
    private fun getScreenDimension(isWidth: Boolean): Int {
        val bounds = windowManager?.currentWindowMetrics?.bounds
        return if (isWidth) bounds?.width() ?: 0 else bounds?.height() ?: 0
    }

    /**
     * Create notification channel
     */
    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            NOTIFICATION_CHANNEL_ID,
            NOTIFICATION_CHANNEL_NAME,
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "Shows when the floating assistant button is active"
            setShowBadge(false)
        }

        val notificationManager = getSystemService(NotificationManager::class.java)
        notificationManager.createNotificationChannel(channel)
    }

    /**
     * Create foreground service notification
     */
    private fun createNotification(): android.app.Notification {
        val stopIntent = Intent(this, FloatingButtonService::class.java).apply {
            action = ACTION_STOP
        }
        val stopPendingIntent = PendingIntent.getService(
            this,
            0,
            stopIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        return NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setContentTitle("Alicia Floating Button")
            .setContentText("Tap the floating button to activate voice assistant")
            .setSmallIcon(android.R.drawable.ic_btn_speak_now)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setOngoing(true)
            .addAction(
                android.R.drawable.ic_menu_close_clear_cancel,
                "Dismiss",
                stopPendingIntent
            )
            .build()
    }

    override fun onDestroy() {
        super.onDestroy()
        hideFloatingButton()
        floatingView = null
        windowManager = null
        Timber.d("FloatingButtonService destroyed")
    }
}
