package com.alicia.assistant.service

import android.app.assist.AssistContent
import android.app.assist.AssistStructure
import android.graphics.Bitmap
import android.util.Log
import com.google.mlkit.vision.common.InputImage
import com.google.mlkit.vision.text.TextRecognition
import com.google.mlkit.vision.text.latin.TextRecognizerOptions
import kotlin.coroutines.resume
import kotlin.coroutines.suspendCoroutine

class ScreenContextManager {

    companion object {
        private const val TAG = "ScreenContext"
        private const val MAX_TEXT_LENGTH = 2000
    }

    private val recognizer = TextRecognition.getClient(TextRecognizerOptions.DEFAULT_OPTIONS)

    fun extractFromStructure(structure: AssistStructure?): String {
        if (structure == null) return ""
        val texts = linkedSetOf<String>()
        for (i in 0 until structure.windowNodeCount) {
            val windowNode = structure.getWindowNodeAt(i)
            traverseNode(windowNode.rootViewNode, texts)
        }
        return texts.joinToString("\n").take(MAX_TEXT_LENGTH)
    }

    fun extractFromContent(content: AssistContent?): String {
        if (content == null) return ""
        val parts = mutableListOf<String>()
        content.webUri?.let { parts.add("URL: $it") }
        content.structuredData?.takeIf { it.isNotBlank() }?.let {
            parts.add("Data: $it")
        }
        return parts.joinToString("\n")
    }

    suspend fun extractFromScreenshot(bitmap: Bitmap): String = suspendCoroutine { cont ->
        val image = InputImage.fromBitmap(bitmap, 0)
        recognizer.process(image)
            .addOnSuccessListener { visionText ->
                val result = visionText.textBlocks.joinToString("\n") { block ->
                    block.lines.joinToString("\n") { it.text }
                }
                cont.resume(result.take(MAX_TEXT_LENGTH))
            }
            .addOnFailureListener { e ->
                Log.e(TAG, "OCR failed", e)
                cont.resume("")
            }
    }

    private fun traverseNode(node: AssistStructure.ViewNode?, texts: MutableSet<String>) {
        if (node == null) return
        node.text?.toString()?.takeIf { it.isNotBlank() }?.let { texts.add(it) }
        node.contentDescription?.toString()?.takeIf { it.isNotBlank() }?.let { texts.add(it) }
        for (i in 0 until node.childCount) {
            traverseNode(node.getChildAt(i), texts)
        }
    }

    fun release() {
        recognizer.close()
    }
}
