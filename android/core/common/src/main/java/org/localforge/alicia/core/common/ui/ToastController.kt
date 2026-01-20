package org.localforge.alicia.core.common.ui

import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.util.UUID

/**
 * Toast variant types matching the web frontend.
 */
enum class ToastVariant {
    DEFAULT,
    SUCCESS,
    WARNING,
    ERROR
}

/**
 * Data class representing a toast notification.
 */
data class Toast(
    val id: String = UUID.randomUUID().toString(),
    val message: String,
    val variant: ToastVariant = ToastVariant.DEFAULT,
    val duration: Long = 4000L
)

/**
 * Controller for managing toast notifications.
 * Can be used as a singleton or injected via Hilt.
 */
class ToastController {
    private val _toasts = MutableStateFlow<List<Toast>>(emptyList())
    val toasts: StateFlow<List<Toast>> = _toasts.asStateFlow()

    fun show(message: String, variant: ToastVariant = ToastVariant.DEFAULT, duration: Long = 4000L) {
        val toast = Toast(message = message, variant = variant, duration = duration)
        _toasts.update { current -> current + toast }
    }

    fun success(message: String, duration: Long = 4000L) {
        show(message, ToastVariant.SUCCESS, duration)
    }

    fun error(message: String, duration: Long = 4000L) {
        show(message, ToastVariant.ERROR, duration)
    }

    fun warning(message: String, duration: Long = 4000L) {
        show(message, ToastVariant.WARNING, duration)
    }

    fun dismiss(id: String) {
        _toasts.update { current -> current.filter { it.id != id } }
    }

    fun clear() {
        _toasts.update { emptyList() }
    }

    companion object {
        // Global singleton for convenience
        val instance = ToastController()
    }
}

/**
 * Convenience object for showing toasts globally.
 */
object AppToast {
    fun show(message: String, variant: ToastVariant = ToastVariant.DEFAULT, duration: Long = 4000L) =
        ToastController.instance.show(message, variant, duration)

    fun success(message: String, duration: Long = 4000L) =
        ToastController.instance.success(message, duration)

    fun error(message: String, duration: Long = 4000L) =
        ToastController.instance.error(message, duration)

    fun warning(message: String, duration: Long = 4000L) =
        ToastController.instance.warning(message, duration)
}

/**
 * Composable that renders the toast container.
 * Should be placed at the root of your app's UI hierarchy.
 */
@Composable
fun ToastHost(
    controller: ToastController = ToastController.instance,
    modifier: Modifier = Modifier
) {
    val toasts by controller.toasts.collectAsState()
    val scope = rememberCoroutineScope()

    Box(
        modifier = modifier.fillMaxSize(),
        contentAlignment = Alignment.BottomEnd
    ) {
        Column(
            modifier = Modifier
                .padding(16.dp)
                .widthIn(max = 400.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
            horizontalAlignment = Alignment.End
        ) {
            toasts.forEach { toast ->
                key(toast.id) {
                    ToastItem(
                        toast = toast,
                        onDismiss = { controller.dismiss(toast.id) },
                        scope = scope
                    )
                }
            }
        }
    }
}

@Composable
private fun ToastItem(
    toast: Toast,
    onDismiss: () -> Unit,
    scope: CoroutineScope
) {
    var isVisible by remember { mutableStateOf(false) }

    LaunchedEffect(toast.id) {
        isVisible = true
        if (toast.duration > 0) {
            delay(toast.duration)
            isVisible = false
            delay(200) // Wait for exit animation
            onDismiss()
        }
    }

    AnimatedVisibility(
        visible = isVisible,
        enter = slideInHorizontally(initialOffsetX = { it }) + fadeIn(),
        exit = slideOutHorizontally(targetOffsetX = { it }) + fadeOut()
    ) {
        ToastContent(
            toast = toast,
            onDismiss = {
                scope.launch {
                    isVisible = false
                    delay(200)
                    onDismiss()
                }
            }
        )
    }
}

@Composable
private fun ToastContent(
    toast: Toast,
    onDismiss: () -> Unit
) {
    val colors = getToastColors(toast.variant)

    Surface(
        shape = RoundedCornerShape(8.dp),
        color = colors.backgroundColor,
        tonalElevation = 4.dp,
        shadowElevation = 8.dp,
        border = androidx.compose.foundation.BorderStroke(1.dp, colors.borderColor)
    ) {
        Row(
            modifier = Modifier
                .padding(16.dp)
                .widthIn(min = 280.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Icon
            Icon(
                imageVector = colors.icon,
                contentDescription = null,
                tint = colors.iconColor,
                modifier = Modifier.size(20.dp)
            )

            // Message
            Text(
                text = toast.message,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium,
                color = colors.textColor,
                modifier = Modifier.weight(1f)
            )

            // Close button
            IconButton(
                onClick = onDismiss,
                modifier = Modifier.size(28.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Dismiss",
                    tint = colors.textColor.copy(alpha = 0.7f),
                    modifier = Modifier.size(16.dp)
                )
            }
        }
    }
}

private data class ToastColors(
    val backgroundColor: Color,
    val borderColor: Color,
    val textColor: Color,
    val iconColor: Color,
    val icon: ImageVector
)

@Composable
private fun getToastColors(variant: ToastVariant): ToastColors {
    val successColor = Color(0xFF4DD488)
    val warningColor = Color(0xFFE5B94D)
    val errorColor = MaterialTheme.colorScheme.error
    val defaultColor = MaterialTheme.colorScheme.onSurface

    return when (variant) {
        ToastVariant.SUCCESS -> ToastColors(
            backgroundColor = successColor.copy(alpha = 0.1f),
            borderColor = successColor.copy(alpha = 0.3f),
            textColor = successColor,
            iconColor = successColor,
            icon = Icons.Default.CheckCircle
        )
        ToastVariant.WARNING -> ToastColors(
            backgroundColor = warningColor.copy(alpha = 0.1f),
            borderColor = warningColor.copy(alpha = 0.3f),
            textColor = warningColor,
            iconColor = warningColor,
            icon = Icons.Default.Warning
        )
        ToastVariant.ERROR -> ToastColors(
            backgroundColor = errorColor.copy(alpha = 0.1f),
            borderColor = errorColor.copy(alpha = 0.3f),
            textColor = errorColor,
            iconColor = errorColor,
            icon = Icons.Default.Error
        )
        ToastVariant.DEFAULT -> ToastColors(
            backgroundColor = MaterialTheme.colorScheme.surfaceVariant,
            borderColor = MaterialTheme.colorScheme.outline.copy(alpha = 0.3f),
            textColor = MaterialTheme.colorScheme.onSurfaceVariant,
            iconColor = defaultColor,
            icon = Icons.Default.Info
        )
    }
}
