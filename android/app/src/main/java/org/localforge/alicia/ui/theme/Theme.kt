package org.localforge.alicia.ui.theme

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

/**
 * Alicia Design System Colors
 * Based on the web frontend's OKLCH color palette
 */
object AliciaColors {
    // Core colors - Dark mode (default, matches web frontend)
    val Background = Color(0xFF1E1D24)          // oklch(0.12 0.005 250)
    val Foreground = Color(0xFFF2F2F2)          // oklch(0.95 0 0)

    // Surface hierarchy
    val Card = Color(0xFF262530)                // oklch(0.15 0.005 250)
    val CardForeground = Color(0xFFF2F2F2)      // oklch(0.95 0 0)
    val Popover = Color(0xFF2D2C38)             // oklch(0.17 0.005 250)
    val Elevated = Color(0xFF343340)            // oklch(0.19 0.005 250)

    // Interactive colors
    val Primary = Color(0xFF7B8AFF)             // oklch(0.65 0.18 250) - Purple/blue
    val PrimaryForeground = Color(0xFFFAFAFA)   // oklch(0.98 0 0)
    val Secondary = Color(0xFF383744)           // oklch(0.22 0.005 250)
    val SecondaryForeground = Color(0xFFE6E6E6) // oklch(0.9 0 0)
    val Accent = Color(0xFF4DD4C5)              // oklch(0.6 0.15 180) - Cyan
    val AccentForeground = Color(0xFFFAFAFA)    // oklch(0.98 0 0)
    val Muted = Color(0xFF414050)               // oklch(0.25 0.005 250)
    val MutedForeground = Color(0xFF999999)     // oklch(0.6 0 0)

    // Semantic colors
    val Destructive = Color(0xFFE55B4A)         // oklch(0.55 0.2 25) - Red
    val DestructiveForeground = Color(0xFFFAFAFA)
    val Success = Color(0xFF4DD488)             // oklch(0.6 0.15 160) - Green
    val SuccessForeground = Color(0xFFFAFAFA)
    val Warning = Color(0xFFE5B94D)             // oklch(0.7 0.15 80) - Yellow
    val WarningForeground = Color(0xFF262626)   // oklch(0.15 0 0)

    // UI elements
    val Border = Color(0xFF414050)              // oklch(0.25 0.005 250)
    val Input = Color(0xFF2D2C38)               // oklch(0.18 0.005 250)
    val Ring = Color(0xFF7B8AFF)                // oklch(0.65 0.18 250)

    // Sidebar
    val Sidebar = Color(0xFF191821)             // oklch(0.1 0.005 250)
    val SidebarForeground = Color(0xFFE6E6E6)   // oklch(0.9 0 0)
    val SidebarAccent = Color(0xFF2D2C38)       // oklch(0.18 0.005 250)

    // Voice states
    val VoiceActive = Color(0xFF4DD488)         // oklch(0.6 0.18 160) - Green
    val VoiceListening = Color(0xFF7B8AFF)      // oklch(0.65 0.18 250) - Purple
    val VoiceProcessing = Color(0xFFE5B94D)     // oklch(0.7 0.15 80) - Yellow

    // Feature-specific colors
    val Reasoning = Color(0xFFCC85FF)           // oklch(0.6 0.18 300) - Magenta
    val ToolUse = Color(0xFF5B9EFF)             // oklch(0.6 0.16 220) - Blue
    val ToolResult = Color(0xFFD4CC4D)          // oklch(0.7 0.14 90) - Yellow-green

    // Memory category colors
    val MemoryPreference = Accent
    val MemoryFact = Success
    val MemoryContext = Warning
    val MemoryInstruction = Destructive

    // Light mode colors (matching web frontend light mode)
    val BackgroundLight = Color(0xFFFAFAFA)     // oklch(0.98 0 0)
    val ForegroundLight = Color(0xFF262630)     // oklch(0.15 0.005 250)
    val CardLight = Color(0xFFFFFFFF)           // oklch(1 0 0)
    val CardForegroundLight = Color(0xFF262630)
    val PopoverLight = Color(0xFFFFFFFF)
    val ElevatedLight = Color(0xFFFCFCFC)       // oklch(0.99 0 0)
    val PrimaryLight = Color(0xFF5B6BDB)        // oklch(0.5 0.2 250)
    val SecondaryLight = Color(0xFFF0F0F2)      // oklch(0.95 0.005 250)
    val SecondaryForegroundLight = Color(0xFF262630)
    val AccentLight = Color(0xFF2AA89A)         // oklch(0.5 0.15 180)
    val MutedLight = Color(0xFFEDEDEF)          // oklch(0.93 0.005 250)
    val MutedForegroundLight = Color(0xFF737380) // oklch(0.45 0.005 250)
    val DestructiveLight = Color(0xFFCC3B2A)    // oklch(0.5 0.2 25)
    val SuccessLight = Color(0xFF2AA868)        // oklch(0.5 0.15 160)
    val WarningLight = Color(0xFFCC9F2A)        // oklch(0.6 0.15 80)
    val BorderLight = Color(0xFFE6E6E8)         // oklch(0.9 0.005 250)
    val InputLight = Color(0xFFF0F0F2)          // oklch(0.95 0.005 250)
    val RingLight = Color(0xFF5B6BDB)           // oklch(0.5 0.2 250)
    val SidebarLight = Color(0xFFF7F7F7)        // oklch(0.97 0 0)
    val SidebarForegroundLight = Color(0xFF262630)
    val SidebarAccentLight = Color(0xFFF0F0F2)
}

/**
 * Extended color scheme for Alicia-specific colors
 */
data class AliciaExtendedColors(
    val card: Color,
    val cardForeground: Color,
    val popover: Color,
    val elevated: Color,
    val accent: Color,
    val accentForeground: Color,
    val muted: Color,
    val mutedForeground: Color,
    val success: Color,
    val successForeground: Color,
    val warning: Color,
    val warningForeground: Color,
    val destructive: Color,
    val destructiveForeground: Color,
    val border: Color,
    val input: Color,
    val ring: Color,
    val sidebar: Color,
    val sidebarForeground: Color,
    val sidebarAccent: Color,
    val voiceActive: Color,
    val voiceListening: Color,
    val voiceProcessing: Color,
    val reasoning: Color,
    val toolUse: Color,
    val toolResult: Color,
    val memoryPreference: Color,
    val memoryFact: Color,
    val memoryContext: Color,
    val memoryInstruction: Color
)

val LocalAliciaColors = staticCompositionLocalOf {
    AliciaExtendedColors(
        card = AliciaColors.Card,
        cardForeground = AliciaColors.CardForeground,
        popover = AliciaColors.Popover,
        elevated = AliciaColors.Elevated,
        accent = AliciaColors.Accent,
        accentForeground = AliciaColors.AccentForeground,
        muted = AliciaColors.Muted,
        mutedForeground = AliciaColors.MutedForeground,
        success = AliciaColors.Success,
        successForeground = AliciaColors.SuccessForeground,
        warning = AliciaColors.Warning,
        warningForeground = AliciaColors.WarningForeground,
        destructive = AliciaColors.Destructive,
        destructiveForeground = AliciaColors.DestructiveForeground,
        border = AliciaColors.Border,
        input = AliciaColors.Input,
        ring = AliciaColors.Ring,
        sidebar = AliciaColors.Sidebar,
        sidebarForeground = AliciaColors.SidebarForeground,
        sidebarAccent = AliciaColors.SidebarAccent,
        voiceActive = AliciaColors.VoiceActive,
        voiceListening = AliciaColors.VoiceListening,
        voiceProcessing = AliciaColors.VoiceProcessing,
        reasoning = AliciaColors.Reasoning,
        toolUse = AliciaColors.ToolUse,
        toolResult = AliciaColors.ToolResult,
        memoryPreference = AliciaColors.MemoryPreference,
        memoryFact = AliciaColors.MemoryFact,
        memoryContext = AliciaColors.MemoryContext,
        memoryInstruction = AliciaColors.MemoryInstruction
    )
}

private val DarkColorScheme = darkColorScheme(
    primary = AliciaColors.Primary,
    onPrimary = AliciaColors.PrimaryForeground,
    primaryContainer = AliciaColors.Primary.copy(alpha = 0.2f),
    onPrimaryContainer = AliciaColors.Primary,
    secondary = AliciaColors.Secondary,
    onSecondary = AliciaColors.SecondaryForeground,
    secondaryContainer = AliciaColors.Secondary.copy(alpha = 0.5f),
    onSecondaryContainer = AliciaColors.SecondaryForeground,
    tertiary = AliciaColors.Accent,
    onTertiary = AliciaColors.AccentForeground,
    tertiaryContainer = AliciaColors.Accent.copy(alpha = 0.2f),
    onTertiaryContainer = AliciaColors.Accent,
    error = AliciaColors.Destructive,
    onError = AliciaColors.DestructiveForeground,
    errorContainer = AliciaColors.Destructive.copy(alpha = 0.2f),
    onErrorContainer = AliciaColors.Destructive,
    background = AliciaColors.Background,
    onBackground = AliciaColors.Foreground,
    surface = AliciaColors.Card,
    onSurface = AliciaColors.CardForeground,
    surfaceVariant = AliciaColors.Muted,
    onSurfaceVariant = AliciaColors.MutedForeground,
    outline = AliciaColors.Border,
    outlineVariant = AliciaColors.Border.copy(alpha = 0.5f),
    inverseSurface = AliciaColors.Foreground,
    inverseOnSurface = AliciaColors.Background,
    inversePrimary = AliciaColors.PrimaryLight,
    surfaceTint = AliciaColors.Primary
)

private val LightColorScheme = lightColorScheme(
    primary = AliciaColors.PrimaryLight,
    onPrimary = AliciaColors.PrimaryForeground,
    primaryContainer = AliciaColors.PrimaryLight.copy(alpha = 0.15f),
    onPrimaryContainer = AliciaColors.PrimaryLight,
    secondary = AliciaColors.SecondaryLight,
    onSecondary = AliciaColors.SecondaryForegroundLight,
    secondaryContainer = AliciaColors.SecondaryLight.copy(alpha = 0.5f),
    onSecondaryContainer = AliciaColors.SecondaryForegroundLight,
    tertiary = AliciaColors.AccentLight,
    onTertiary = AliciaColors.AccentForeground,
    tertiaryContainer = AliciaColors.AccentLight.copy(alpha = 0.15f),
    onTertiaryContainer = AliciaColors.AccentLight,
    error = AliciaColors.DestructiveLight,
    onError = AliciaColors.DestructiveForeground,
    errorContainer = AliciaColors.DestructiveLight.copy(alpha = 0.15f),
    onErrorContainer = AliciaColors.DestructiveLight,
    background = AliciaColors.BackgroundLight,
    onBackground = AliciaColors.ForegroundLight,
    surface = AliciaColors.CardLight,
    onSurface = AliciaColors.CardForegroundLight,
    surfaceVariant = AliciaColors.MutedLight,
    onSurfaceVariant = AliciaColors.MutedForegroundLight,
    outline = AliciaColors.BorderLight,
    outlineVariant = AliciaColors.BorderLight.copy(alpha = 0.5f),
    inverseSurface = AliciaColors.ForegroundLight,
    inverseOnSurface = AliciaColors.BackgroundLight,
    inversePrimary = AliciaColors.Primary,
    surfaceTint = AliciaColors.PrimaryLight
)

private val DarkExtendedColors = AliciaExtendedColors(
    card = AliciaColors.Card,
    cardForeground = AliciaColors.CardForeground,
    popover = AliciaColors.Popover,
    elevated = AliciaColors.Elevated,
    accent = AliciaColors.Accent,
    accentForeground = AliciaColors.AccentForeground,
    muted = AliciaColors.Muted,
    mutedForeground = AliciaColors.MutedForeground,
    success = AliciaColors.Success,
    successForeground = AliciaColors.SuccessForeground,
    warning = AliciaColors.Warning,
    warningForeground = AliciaColors.WarningForeground,
    destructive = AliciaColors.Destructive,
    destructiveForeground = AliciaColors.DestructiveForeground,
    border = AliciaColors.Border,
    input = AliciaColors.Input,
    ring = AliciaColors.Ring,
    sidebar = AliciaColors.Sidebar,
    sidebarForeground = AliciaColors.SidebarForeground,
    sidebarAccent = AliciaColors.SidebarAccent,
    voiceActive = AliciaColors.VoiceActive,
    voiceListening = AliciaColors.VoiceListening,
    voiceProcessing = AliciaColors.VoiceProcessing,
    reasoning = AliciaColors.Reasoning,
    toolUse = AliciaColors.ToolUse,
    toolResult = AliciaColors.ToolResult,
    memoryPreference = AliciaColors.MemoryPreference,
    memoryFact = AliciaColors.MemoryFact,
    memoryContext = AliciaColors.MemoryContext,
    memoryInstruction = AliciaColors.MemoryInstruction
)

private val LightExtendedColors = AliciaExtendedColors(
    card = AliciaColors.CardLight,
    cardForeground = AliciaColors.CardForegroundLight,
    popover = AliciaColors.PopoverLight,
    elevated = AliciaColors.ElevatedLight,
    accent = AliciaColors.AccentLight,
    accentForeground = AliciaColors.AccentForeground,
    muted = AliciaColors.MutedLight,
    mutedForeground = AliciaColors.MutedForegroundLight,
    success = AliciaColors.SuccessLight,
    successForeground = AliciaColors.SuccessForeground,
    warning = AliciaColors.WarningLight,
    warningForeground = AliciaColors.WarningForeground,
    destructive = AliciaColors.DestructiveLight,
    destructiveForeground = AliciaColors.DestructiveForeground,
    border = AliciaColors.BorderLight,
    input = AliciaColors.InputLight,
    ring = AliciaColors.RingLight,
    sidebar = AliciaColors.SidebarLight,
    sidebarForeground = AliciaColors.SidebarForegroundLight,
    sidebarAccent = AliciaColors.SidebarAccentLight,
    voiceActive = AliciaColors.VoiceActive,
    voiceListening = AliciaColors.VoiceListening,
    voiceProcessing = AliciaColors.VoiceProcessing,
    reasoning = AliciaColors.Reasoning,
    toolUse = AliciaColors.ToolUse,
    toolResult = AliciaColors.ToolResult,
    memoryPreference = AliciaColors.AccentLight,
    memoryFact = AliciaColors.SuccessLight,
    memoryContext = AliciaColors.WarningLight,
    memoryInstruction = AliciaColors.DestructiveLight
)

@Composable
fun AliciaTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    // Disable dynamic colors to maintain consistent branding
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme
    val extendedColors = if (darkTheme) DarkExtendedColors else LightExtendedColors

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val activity = view.context as? Activity
            activity?.window?.let { window ->
                WindowCompat.setDecorFitsSystemWindows(window, false)
                WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = !darkTheme
            }
        }
    }

    CompositionLocalProvider(LocalAliciaColors provides extendedColors) {
        MaterialTheme(
            colorScheme = colorScheme,
            typography = Typography,
            content = content
        )
    }
}

/**
 * Access the extended Alicia color palette from any composable
 */
object AliciaTheme {
    val extendedColors: AliciaExtendedColors
        @Composable
        get() = LocalAliciaColors.current
}
