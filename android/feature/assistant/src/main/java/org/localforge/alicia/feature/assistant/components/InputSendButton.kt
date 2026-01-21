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
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.ui.theme.AliciaTheme

@Composable
fun InputSendButton(
    onSend: () -> Unit = {},
    canSend: Boolean = false,
    disabled: Boolean = false,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors
    val interactionSource = remember { MutableInteractionSource() }
    val isPressed by interactionSource.collectIsPressedAsState()

    val backgroundColor by animateColorAsState(
        targetValue = when {
            disabled -> extendedColors.muted
            !canSend -> extendedColors.card
            else -> extendedColors.accent
        },
        animationSpec = tween(200),
        label = "send_bg_color"
    )

    val iconColor by animateColorAsState(
        targetValue = when {
            disabled -> extendedColors.mutedForeground
            !canSend -> extendedColors.mutedForeground
            else -> extendedColors.accentForeground
        },
        animationSpec = tween(200),
        label = "send_icon_color"
    )

    val scale by animateFloatAsState(
        targetValue = if (isPressed && canSend && !disabled) 0.95f else 1f,
        animationSpec = spring(stiffness = Spring.StiffnessHigh),
        label = "send_scale"
    )

    val iconScale by animateFloatAsState(
        targetValue = if (isPressed && canSend) 1.1f else 1f,
        animationSpec = spring(stiffness = Spring.StiffnessMedium),
        label = "icon_scale"
    )

    val contentDesc = if (!canSend) {
        "Send message (disabled - no message to send)"
    } else {
        "Send message"
    }

    Box(
        modifier = modifier
            .size(40.dp)
            .scale(scale)
            .clip(CircleShape)
            .background(backgroundColor)
            .clickable(
                interactionSource = interactionSource,
                indication = ripple(),
                enabled = !disabled && canSend,
                onClick = onSend
            )
            .semantics { this.contentDescription = contentDesc },
        contentAlignment = Alignment.Center
    ) {
        Icon(
            imageVector = AppIcons.Send,
            contentDescription = null,
            tint = iconColor.copy(
                alpha = if (!canSend) 0.5f else 1f
            ),
            modifier = Modifier
                .size(20.dp)
                .scale(iconScale)
                .graphicsLayer {
                    translationX = 2.dp.toPx()
                }
        )

        if (canSend) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape)
                    .background(
                        extendedColors.accent.copy(
                            alpha = if (isPressed) 0.2f else 0f
                        )
                    )
            )
        }
    }
}
