package com.alicia.assistant

/**
 * Derives a display title for a note. If the explicit title is blank,
 * truncates the content to [maxLength] characters with an ellipsis.
 */
fun deriveNoteTitle(title: String, content: String, maxLength: Int = 50): String =
    if (title.isBlank()) {
        content.take(maxLength).let { if (content.length > maxLength) "$it\u2026" else it }
    } else {
        title
    }
