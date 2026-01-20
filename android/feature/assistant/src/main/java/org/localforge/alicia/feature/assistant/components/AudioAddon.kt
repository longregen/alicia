package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * Audio playback state
 */
enum class AudioState {
    IDLE, LOADING, PLAYING, PAUSED
}

/**
 * Display mode for AudioAddon
 */
enum class AudioAddonMode {
    COMPACT, FULL
}

/**
 * AudioAddon component for audio playback controls.
 * Matches the web frontend's AudioAddon.tsx
 *
 * Displays play/pause/stop buttons with progress bar and time display.
 *
 * @param state Current audio state
 * @param onPlay Callback when play is clicked
 * @param onPause Callback when pause is clicked
 * @param onStop Callback when stop is clicked (resets to beginning)
 * @param duration Total duration in seconds
 * @param currentTime Current playback time in seconds
 * @param disabled Whether the addon is disabled
 * @param mode Display mode - compact for inline use, full for expanded view
 * @param modifier Modifier for the component
 */
@Composable
fun AudioAddon(
    state: AudioState = AudioState.IDLE,
    onPlay: () -> Unit = {},
    onPause: () -> Unit = {},
    onStop: () -> Unit = {},
    duration: Int = 0,
    currentTime: Int = 0,
    disabled: Boolean = false,
    mode: AudioAddonMode = AudioAddonMode.FULL,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors

    // Auto-expand when playing in full mode
    var isExpanded by remember { mutableStateOf(false) }

    LaunchedEffect(state, mode) {
        if (mode == AudioAddonMode.FULL) {
            isExpanded = state == AudioState.PLAYING || state == AudioState.PAUSED
        }
    }

    val handleClick = {
        if (!disabled) {
            when (state) {
                AudioState.IDLE -> onPlay()
                AudioState.PLAYING -> onPause()
                AudioState.PAUSED -> onPlay()
                AudioState.LOADING -> { /* do nothing */ }
            }
        }
    }

    if (mode == AudioAddonMode.COMPACT) {
        CompactAudioAddon(
            state = state,
            duration = duration,
            currentTime = currentTime,
            disabled = disabled,
            onClick = handleClick,
            modifier = modifier
        )
    } else {
        FullAudioAddon(
            state = state,
            isExpanded = isExpanded,
            duration = duration,
            currentTime = currentTime,
            disabled = disabled,
            onClick = handleClick,
            onStop = onStop,
            modifier = modifier
        )
    }
}

@Composable
private fun CompactAudioAddon(
    state: AudioState,
    duration: Int,
    currentTime: Int,
    disabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    val isPlaying = state == AudioState.PLAYING

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Play/Pause button
        Box(
            modifier = Modifier
                .size(24.dp)
                .clip(CircleShape)
                .background(
                    if (isPlaying) extendedColors.accent
                    else extendedColors.accent.copy(alpha = 0.15f)
                )
                .clickable(enabled = !disabled && state != AudioState.LOADING, onClick = onClick),
            contentAlignment = Alignment.Center
        ) {
            when (state) {
                AudioState.LOADING -> {
                    CircularProgressIndicator(
                        modifier = Modifier.size(12.dp),
                        strokeWidth = 2.dp,
                        color = extendedColors.accentForeground
                    )
                }
                AudioState.PLAYING -> {
                    Icon(
                        imageVector = AppIcons.Pause,
                        contentDescription = "Pause",
                        tint = extendedColors.accentForeground,
                        modifier = Modifier.size(12.dp)
                    )
                }
                else -> {
                    Icon(
                        imageVector = AppIcons.PlayArrow,
                        contentDescription = "Play",
                        tint = if (isPlaying) extendedColors.accentForeground else extendedColors.accent,
                        modifier = Modifier.size(12.dp)
                    )
                }
            }
        }

        // Time display
        Text(
            text = if (isPlaying) {
                "${formatTime(currentTime)} / ${formatTime(duration)}"
            } else {
                formatTime(duration)
            },
            style = MaterialTheme.typography.bodySmall,
            fontSize = 12.sp,
            color = if (isPlaying) extendedColors.accentForeground else extendedColors.mutedForeground
        )
    }
}

@Composable
private fun FullAudioAddon(
    state: AudioState,
    isExpanded: Boolean,
    duration: Int,
    currentTime: Int,
    disabled: Boolean,
    onClick: () -> Unit,
    onStop: () -> Unit,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors

    Row(
        modifier = modifier.animateContentSize(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Main play/pause button
        Box(
            modifier = Modifier
                .size(32.dp)
                .clip(CircleShape)
                .background(extendedColors.accent)
                .clickable(enabled = !disabled && state != AudioState.LOADING, onClick = onClick),
            contentAlignment = Alignment.Center
        ) {
            when (state) {
                AudioState.LOADING -> {
                    CircularProgressIndicator(
                        modifier = Modifier.size(16.dp),
                        strokeWidth = 2.dp,
                        color = extendedColors.accentForeground
                    )
                }
                AudioState.PLAYING -> {
                    Icon(
                        imageVector = AppIcons.Pause,
                        contentDescription = "Pause",
                        tint = extendedColors.accentForeground,
                        modifier = Modifier.size(16.dp)
                    )
                }
                else -> {
                    Icon(
                        imageVector = AppIcons.PlayArrow,
                        contentDescription = "Play",
                        tint = extendedColors.accentForeground,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
        }

        if (isExpanded) {
            // Progress bar
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height(4.dp)
                    .clip(RoundedCornerShape(2.dp))
                    .background(extendedColors.muted)
            ) {
                val progress = if (duration > 0) currentTime.toFloat() / duration else 0f
                Box(
                    modifier = Modifier
                        .fillMaxHeight()
                        .fillMaxWidth(progress)
                        .background(extendedColors.accent)
                )
            }

            // Time display
            Text(
                text = "${formatTime(currentTime)} / ${formatTime(duration)}",
                style = MaterialTheme.typography.bodySmall,
                fontSize = 12.sp,
                color = extendedColors.mutedForeground,
                modifier = Modifier.widthIn(min = 70.dp)
            )

            // Stop button
            if (state == AudioState.PLAYING || state == AudioState.PAUSED) {
                Box(
                    modifier = Modifier
                        .size(24.dp)
                        .clip(CircleShape)
                        .clickable(onClick = onStop),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = AppIcons.Stop,
                        contentDescription = "Stop",
                        tint = extendedColors.mutedForeground,
                        modifier = Modifier.size(12.dp)
                    )
                }
            }
        }
    }
}

/**
 * Format seconds to MM:SS string
 */
private fun formatTime(seconds: Int): String {
    val mins = seconds / 60
    val secs = seconds % 60
    return "$mins:${secs.toString().padStart(2, '0')}"
}
