package org.localforge.alicia.feature.settings.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.localforge.alicia.ui.theme.AliciaTheme

/**
 * Response length options matching the web frontend
 */
enum class ResponseLength(val displayName: String, val description: String) {
    CONCISE("Concise", "Short, direct answers without elaboration."),
    BALANCED("Balanced", "Clear explanations with relevant details."),
    DETAILED("Detailed", "Comprehensive responses with examples and context.")
}

/**
 * Response length selector component matching the web frontend's Settings.tsx
 *
 * Displays three buttons for selecting response length preference.
 */
@Composable
fun ResponseLengthSelector(
    selectedLength: ResponseLength,
    onLengthSelected: (ResponseLength) -> Unit,
    modifier: Modifier = Modifier
) {
    val extendedColors = AliciaTheme.extendedColors

    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Text(
            text = "Response Length",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Medium
        )

        // Labels row
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            ResponseLength.entries.forEach { length ->
                Text(
                    text = length.displayName,
                    style = MaterialTheme.typography.bodySmall,
                    fontSize = 12.sp,
                    color = extendedColors.mutedForeground,
                    modifier = Modifier.weight(1f),
                    textAlign = when (length) {
                        ResponseLength.CONCISE -> TextAlign.Start
                        ResponseLength.BALANCED -> TextAlign.Center
                        ResponseLength.DETAILED -> TextAlign.End
                    }
                )
            }
        }

        // Buttons row
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            ResponseLength.entries.forEach { length ->
                val isSelected = length == selectedLength

                Box(
                    modifier = Modifier
                        .weight(1f)
                        .clip(RoundedCornerShape(8.dp))
                        .background(
                            if (isSelected) extendedColors.accent
                            else extendedColors.muted
                        )
                        .clickable { onLengthSelected(length) }
                        .padding(vertical = 12.dp, horizontal = 8.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = length.displayName,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = if (isSelected) extendedColors.accentForeground
                                else extendedColors.mutedForeground
                    )
                }
            }
        }

        // Description
        Text(
            text = selectedLength.description,
            style = MaterialTheme.typography.bodySmall,
            fontSize = 12.sp,
            color = extendedColors.mutedForeground
        )
    }
}
