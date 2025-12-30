package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import org.localforge.alicia.core.common.ui.AppIcons
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.domain.model.VoiceState

@Composable
fun VoiceStateIndicator(
    state: VoiceState,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")

    val pulseScale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 1.2f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulseScale"
    )

    Box(
        modifier = modifier.size(120.dp),
        contentAlignment = Alignment.Center
    ) {
        // Outer pulse rings when listening
        if (state == VoiceState.LISTENING) {
            Box(
                modifier = Modifier
                    .size(120.dp * pulseScale)
                    .clip(CircleShape)
                    .background(
                        MaterialTheme.colorScheme.primary.copy(alpha = 0.2f)
                    )
            )
        }

        // Main indicator circle
        Box(
            modifier = Modifier
                .size(80.dp)
                .clip(CircleShape)
                .background(
                    when (state) {
                        VoiceState.IDLE -> MaterialTheme.colorScheme.surfaceVariant
                        VoiceState.LISTENING_FOR_WAKE_WORD -> MaterialTheme.colorScheme.secondary.copy(alpha = 0.5f)
                        VoiceState.ACTIVATED -> MaterialTheme.colorScheme.primary
                        VoiceState.LISTENING -> MaterialTheme.colorScheme.tertiary
                        VoiceState.PROCESSING -> MaterialTheme.colorScheme.secondary
                        VoiceState.SPEAKING -> MaterialTheme.colorScheme.primaryContainer
                        VoiceState.ERROR -> MaterialTheme.colorScheme.error
                        VoiceState.CONNECTING -> MaterialTheme.colorScheme.outline
                        VoiceState.DISCONNECTED -> MaterialTheme.colorScheme.surfaceVariant
                    }
                ),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = when (state) {
                    VoiceState.IDLE -> AppIcons.MicOff
                    VoiceState.LISTENING_FOR_WAKE_WORD -> AppIcons.Hearing
                    VoiceState.ACTIVATED -> AppIcons.Mic
                    VoiceState.LISTENING -> AppIcons.Mic
                    VoiceState.PROCESSING -> AppIcons.HourglassEmpty
                    VoiceState.SPEAKING -> AppIcons.VolumeUp
                    VoiceState.ERROR -> AppIcons.Error
                    VoiceState.CONNECTING -> AppIcons.Sync
                    VoiceState.DISCONNECTED -> AppIcons.CloudOff
                },
                contentDescription = state.name,
                tint = Color.White,
                modifier = Modifier.size(40.dp)
            )
        }

        // Sound wave animation when speaking
        if (state == VoiceState.SPEAKING) {
            SoundWaveAnimation(modifier = Modifier.size(120.dp))
        }
    }
}
