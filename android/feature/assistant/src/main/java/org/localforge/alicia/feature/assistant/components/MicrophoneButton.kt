package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * Microphone status enum matching web MicrophoneStatus
 */
enum class MicrophoneStatus {
    Inactive,
    RequestingPermission,
    Loading,
    Active,
    Recording,
    Sending,
    Error
}

/**
 * MicrophoneButton - Voice input button with VAD visualization.
 * Matches the web's MicrophoneVAD.tsx component.
 *
 * Features:
 * - Animated expanding rings when speech is detected
 * - Color changes based on status
 * - Loading spinner when initializing
 * - Speech probability visualization
 */
@Composable
fun MicrophoneButton(
    microphoneStatus: MicrophoneStatus = MicrophoneStatus.Inactive,
    isSpeaking: Boolean = false,
    speechProbability: Float = 0f,
    onClick: () -> Unit = {},
    disabled: Boolean = false,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    val interactionSource = remember { MutableInteractionSource() }
    val isPressed by interactionSource.collectIsPressedAsState()

    val isActive = microphoneStatus == MicrophoneStatus.Recording ||
            microphoneStatus == MicrophoneStatus.Sending
    val isLoading = microphoneStatus == MicrophoneStatus.Loading ||
            microphoneStatus == MicrophoneStatus.RequestingPermission
    val isReady = microphoneStatus == MicrophoneStatus.Active
    val isError = microphoneStatus == MicrophoneStatus.Error
    val isSending = microphoneStatus == MicrophoneStatus.Sending

    // Background color animation
    val backgroundColor by animateColorAsState(
        targetValue = when {
            disabled -> extendedColors.muted
            isError -> extendedColors.destructive.copy(alpha = 0.2f)
            isLoading -> extendedColors.accent.copy(alpha = 0.2f)
            isSending -> extendedColors.accent.copy(alpha = 0.2f)
            isActive && isSpeaking -> extendedColors.success.copy(alpha = 0.2f)
            isActive -> extendedColors.accent.copy(alpha = 0.2f)
            isReady -> extendedColors.accent.copy(alpha = 0.2f)
            else -> extendedColors.muted
        },
        animationSpec = tween(200),
        label = "mic_bg_color"
    )

    // Icon color animation
    val iconColor by animateColorAsState(
        targetValue = when {
            disabled -> extendedColors.mutedForeground
            isError -> extendedColors.destructive
            isLoading -> extendedColors.accent
            isSending -> extendedColors.accent
            !isActive && !isReady -> extendedColors.mutedForeground
            isSpeaking -> extendedColors.success
            else -> extendedColors.accent
        },
        animationSpec = tween(200),
        label = "mic_icon_color"
    )

    // Ring color for visualization
    val ringColor = if (isSpeaking) extendedColors.success else extendedColors.accent

    // Scale animation when pressed
    val scale by animateFloatAsState(
        targetValue = if (isPressed) 0.95f else 1f,
        animationSpec = spring(stiffness = Spring.StiffnessHigh),
        label = "mic_scale"
    )

    // Pulsing animation for loading state
    val pulseAlpha by rememberInfiniteTransition(label = "pulse").animateFloat(
        initialValue = 0.6f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_alpha"
    )

    // Animated rings state
    val rings = remember { mutableStateListOf<RingState>() }
    var frameCount by remember { mutableIntStateOf(0) }
    var lastRingProbability by remember { mutableFloatStateOf(0f) }

    // Update rings animation
    LaunchedEffect(isActive, isSending, speechProbability) {
        if (!isActive || isSending) {
            rings.clear()
            frameCount = 0
            lastRingProbability = 0f
            return@LaunchedEffect
        }

        // Add new ring every ~5 frames when speech detected
        frameCount++
        if (frameCount % 5 == 0 && speechProbability > 0f) {
            val thresholds = listOf(1.0f, 0.98f, 0.95f, 0.90f, 0.75f, 0.50f, 0f)
            val lastThreshold = thresholds.find { lastRingProbability >= it } ?: 0f
            val currentThreshold = thresholds.find { speechProbability >= it } ?: 0f

            if (currentThreshold != lastThreshold || speechProbability > 0.5f) {
                if (rings.size < 10) {
                    rings.add(RingState(
                        radius = 0f,
                        opacity = kotlin.math.sqrt(speechProbability)
                    ))
                }
                lastRingProbability = speechProbability
            }
        }
    }

    // Animate existing rings
    rings.forEachIndexed { index, ring ->
        val animatedRadius by animateFloatAsState(
            targetValue = ring.radius + 21f,
            animationSpec = tween(500),
            label = "ring_radius_$index"
        )
        val animatedOpacity by animateFloatAsState(
            targetValue = 0f,
            animationSpec = tween(500),
            label = "ring_opacity_$index"
        )
        ring.radius = animatedRadius
        ring.opacity = animatedOpacity
    }

    // Clean up faded rings
    LaunchedEffect(rings.size) {
        rings.removeAll { it.opacity < 0.01f || it.radius > 21f }
    }

    // Content description for accessibility
    val contentDesc = when {
        disabled -> "Microphone disabled"
        isError -> "Microphone error"
        isLoading -> "Loading microphone"
        isSending -> "Sending speech"
        isActive -> "Stop recording"
        isReady -> "Start recording (ready)"
        else -> "Start recording"
    }

    Box(
        modifier = modifier
            .size(40.dp)
            .scale(scale)
            .clip(CircleShape)
            .background(
                if (isLoading) backgroundColor.copy(alpha = pulseAlpha)
                else backgroundColor
            )
            .clickable(
                interactionSource = interactionSource,
                indication = ripple(),
                enabled = !disabled && !isLoading,
                onClick = onClick
            )
            .drawBehind {
                // Draw expanding rings
                rings.forEach { ring ->
                    drawCircle(
                        color = ringColor.copy(alpha = ring.opacity),
                        radius = ring.radius.dp.toPx(),
                        style = Stroke(width = 1.5.dp.toPx())
                    )
                }
            }
            .semantics { this.contentDescription = contentDesc },
        contentAlignment = Alignment.Center
    ) {
        when {
            isLoading -> {
                // Loading spinner
                CircularProgressIndicator(
                    modifier = Modifier.size(20.dp),
                    color = extendedColors.accent,
                    strokeWidth = 2.dp
                )
            }
            isSending -> {
                // Sending indicator (up arrow bouncing)
                val bounce by rememberInfiniteTransition(label = "bounce").animateFloat(
                    initialValue = 0f,
                    targetValue = -4f,
                    animationSpec = infiniteRepeatable(
                        animation = tween(300),
                        repeatMode = RepeatMode.Reverse
                    ),
                    label = "bounce_y"
                )
                Icon(
                    imageVector = AppIcons.ArrowUpward,
                    contentDescription = null,
                    tint = extendedColors.accent,
                    modifier = Modifier
                        .size(12.dp)
                        .offset(y = bounce.dp)
                )
            }
            else -> {
                // Microphone icon
                Icon(
                    imageVector = AppIcons.Mic,
                    contentDescription = null,
                    tint = iconColor.copy(
                        alpha = if (isLoading || isSending) 0.5f else 1f
                    ),
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

private data class RingState(
    var radius: Float,
    var opacity: Float
)
