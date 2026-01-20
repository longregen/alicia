package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * Vote type for feedback
 */
enum class VoteType {
    UP, DOWN, NONE
}

/**
 * FeedbackControls component for voting on messages, tools, memories, and reasoning.
 * Matches the web frontend's FeedbackControls.tsx
 *
 * Displays upvote/downvote buttons with counts and handles vote state.
 */
@Composable
fun FeedbackControls(
    currentVote: VoteType,
    onVote: (VoteType) -> Unit,
    upvotes: Int = 0,
    downvotes: Int = 0,
    isLoading: Boolean = false,
    compact: Boolean = false,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Upvote button
        FeedbackButton(
            isSelected = currentVote == VoteType.UP,
            onClick = {
                if (!isLoading) {
                    onVote(if (currentVote == VoteType.UP) VoteType.NONE else VoteType.UP)
                }
            },
            enabled = !isLoading,
            selectedColor = extendedColors.success,
            compact = compact,
            isUpvote = true,
            count = upvotes
        )

        // Downvote button
        FeedbackButton(
            isSelected = currentVote == VoteType.DOWN,
            onClick = {
                if (!isLoading) {
                    onVote(if (currentVote == VoteType.DOWN) VoteType.NONE else VoteType.DOWN)
                }
            },
            enabled = !isLoading,
            selectedColor = extendedColors.destructive,
            compact = compact,
            isUpvote = false,
            count = downvotes
        )

        // Loading indicator
        if (isLoading) {
            CircularProgressIndicator(
                modifier = Modifier.size(16.dp),
                strokeWidth = 2.dp,
                color = extendedColors.accent
            )
        }
    }
}

@Composable
private fun FeedbackButton(
    isSelected: Boolean,
    onClick: () -> Unit,
    enabled: Boolean,
    selectedColor: Color,
    compact: Boolean,
    isUpvote: Boolean,
    count: Int
) {
    val extendedColors = AliciaTheme.extendedColors

    val backgroundColor = if (isSelected) {
        selectedColor.copy(alpha = 0.15f)
    } else {
        extendedColors.card
    }

    val contentColor = if (isSelected) {
        selectedColor
    } else {
        extendedColors.mutedForeground
    }

    val borderColor = if (isSelected) {
        selectedColor
    } else {
        Color.Transparent
    }

    val paddingHorizontal = if (compact) 6.dp else 8.dp
    val paddingVertical = if (compact) 2.dp else 4.dp
    val iconSize = if (compact) 12.dp else 16.dp
    val fontSize = if (compact) 10.sp else 12.sp

    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .background(backgroundColor)
            .border(
                width = 1.dp,
                color = borderColor,
                shape = RoundedCornerShape(6.dp)
            )
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = paddingHorizontal, vertical = paddingVertical),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = if (isUpvote) AppIcons.ThumbUp else AppIcons.ThumbDown,
            contentDescription = if (isUpvote) {
                if (isSelected) "Remove upvote" else "Upvote"
            } else {
                if (isSelected) "Remove downvote" else "Downvote"
            },
            tint = contentColor,
            modifier = Modifier.size(iconSize)
        )

        if (count > 0) {
            Text(
                text = count.toString(),
                color = contentColor,
                fontSize = fontSize
            )
        }
    }
}
