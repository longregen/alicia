package org.localforge.alicia.feature.assistant.components

import androidx.compose.animation.core.*
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import kotlin.math.sin

@Composable
fun SoundWaveAnimation(
    modifier: Modifier = Modifier,
    color: Color = Color.White.copy(alpha = 0.6f)
) {
    val infiniteTransition = rememberInfiniteTransition(label = "soundWave")

    val phase1 by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(1500, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "phase1"
    )

    val phase2 by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(1200, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "phase2"
    )

    val phase3 by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(1800, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "phase3"
    )

    Canvas(modifier = modifier.fillMaxSize()) {
        val centerX = size.width / 2
        val centerY = size.height / 2
        val maxRadius = size.minDimension / 2

        listOf(
            Triple(maxRadius * 0.6f, phase1, 0.8f),
            Triple(maxRadius * 0.8f, phase2, 0.5f),
            Triple(maxRadius * 1.0f, phase3, 0.3f)
        ).forEach { (baseRadius, phase, alpha) ->
            val points = 60
            val path = mutableListOf<Offset>()

            for (i in 0..points) {
                val angle = (i.toFloat() / points) * 360f
                val wave = sin(Math.toRadians((angle + phase).toDouble())).toFloat() * 5f
                val radius = baseRadius + wave

                val x = centerX + radius * kotlin.math.cos(Math.toRadians(angle.toDouble())).toFloat()
                val y = centerY + radius * kotlin.math.sin(Math.toRadians(angle.toDouble())).toFloat()

                path.add(Offset(x, y))
            }

            for (i in 0 until path.size - 1) {
                drawLine(
                    color = color.copy(alpha = alpha),
                    start = path[i],
                    end = path[i + 1],
                    strokeWidth = 2f,
                    cap = StrokeCap.Round
                )
            }
        }
    }
}
