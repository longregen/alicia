package com.alicia.assistant.service

import ai.onnxruntime.OnnxTensor
import ai.onnxruntime.OrtEnvironment
import ai.onnxruntime.OrtSession
import android.content.Context
import android.util.Log
import java.io.Closeable
import java.nio.FloatBuffer
import java.nio.LongBuffer

class SileroVadDetector private constructor(
    private val session: OrtSession,
    private val env: OrtEnvironment
) : Closeable {

    private val stateShape = longArrayOf(2, 1, 128)
    private val state = FloatArray(2 * 1 * 128)
    private val audioContext = FloatArray(CONTEXT_SIZE)
    private val inputWindow = FloatArray(WINDOW_SIZE)
    private val srTensor: OnnxTensor = OnnxTensor.createTensor(
        env, LongBuffer.wrap(longArrayOf(SAMPLE_RATE.toLong())), longArrayOf(1)
    )

    fun isSpeech(audioFrame: FloatArray): Float {
        require(audioFrame.size == FRAME_SIZE) {
            "Expected $FRAME_SIZE samples, got ${audioFrame.size}"
        }

        // Build input: [previous context | current frame], then save tail as next context
        audioContext.copyInto(inputWindow, 0, 0, CONTEXT_SIZE)
        audioFrame.copyInto(inputWindow, CONTEXT_SIZE, 0, FRAME_SIZE)
        audioFrame.copyInto(audioContext, 0, FRAME_SIZE - CONTEXT_SIZE, FRAME_SIZE)

        val inputTensor = OnnxTensor.createTensor(
            env, FloatBuffer.wrap(inputWindow), longArrayOf(1, WINDOW_SIZE.toLong())
        )
        val stateTensor = OnnxTensor.createTensor(
            env, FloatBuffer.wrap(state), stateShape
        )

        val inputs = mapOf("input" to inputTensor, "state" to stateTensor, "sr" to srTensor)

        return try {
            session.run(inputs).use { results ->
                val outputTensor = results[0] as OnnxTensor
                val speechProb = outputTensor.floatBuffer.get(0)

                val stateOutput = results[1] as OnnxTensor
                val stateBuffer = stateOutput.floatBuffer
                stateBuffer.rewind()
                stateBuffer.get(state)

                speechProb
            }
        } catch (e: Exception) {
            Log.e(TAG, "VAD inference failed", e)
            0f
        } finally {
            inputTensor.close()
            stateTensor.close()
        }
    }

    fun resetState() {
        state.fill(0f)
        audioContext.fill(0f)
    }

    private var closed = false

    override fun close() {
        if (closed) return
        closed = true
        srTensor.close()
        session.close()
    }

    companion object {
        private const val TAG = "SileroVAD"
        const val SAMPLE_RATE = 16000
        const val FRAME_SIZE = 512
        private const val CONTEXT_SIZE = 64
        private const val WINDOW_SIZE = FRAME_SIZE + CONTEXT_SIZE
        private const val MODEL_ASSET = "silero_vad.onnx"

        fun create(context: Context): SileroVadDetector {
            val env = OrtEnvironment.getEnvironment()
            val modelBytes = context.assets.open(MODEL_ASSET).readBytes()
            val session = env.createSession(modelBytes)
            Log.d(TAG, "Silero VAD model loaded")
            return SileroVadDetector(session, env)
        }
    }
}
