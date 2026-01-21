package org.localforge.alicia.feature.assistant.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.core.common.ui.AppIcons
import org.localforge.alicia.feature.assistant.BranchDirection
import org.localforge.alicia.ui.theme.AliciaTheme

@Composable
fun BranchNavigator(
    currentIndex: Int,
    totalBranches: Int,
    onNavigate: (direction: BranchDirection) -> Unit,
    modifier: Modifier = Modifier
) {
    if (totalBranches <= 1) return

    val extendedColors = AliciaTheme.extendedColors
    val displayIndex = currentIndex + 1 // Convert from 0-based to 1-based

    val canGoPrev = currentIndex > 0
    val canGoNext = currentIndex < totalBranches - 1

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = Modifier
                .size(24.dp)
                .clip(RoundedCornerShape(4.dp))
                .alpha(if (canGoPrev) 1f else 0.3f)
                .then(
                    if (canGoPrev) {
                        Modifier.clickable { onNavigate(BranchDirection.PREV) }
                    } else {
                        Modifier
                    }
                ),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = AppIcons.ChevronLeft,
                contentDescription = "Previous branch",
                tint = extendedColors.mutedForeground,
                modifier = Modifier.size(12.dp)
            )
        }

        Text(
            text = "$displayIndex/$totalBranches",
            style = MaterialTheme.typography.bodySmall,
            fontSize = 12.sp,
            fontFamily = FontFamily.Monospace,
            color = extendedColors.mutedForeground,
            textAlign = TextAlign.Center,
            modifier = Modifier.widthIn(min = 40.dp)
        )

        Box(
            modifier = Modifier
                .size(24.dp)
                .clip(RoundedCornerShape(4.dp))
                .alpha(if (canGoNext) 1f else 0.3f)
                .then(
                    if (canGoNext) {
                        Modifier.clickable { onNavigate(BranchDirection.NEXT) }
                    } else {
                        Modifier
                    }
                ),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = AppIcons.ChevronRight,
                contentDescription = "Next branch",
                tint = extendedColors.mutedForeground,
                modifier = Modifier.size(12.dp)
            )
        }
    }
}
