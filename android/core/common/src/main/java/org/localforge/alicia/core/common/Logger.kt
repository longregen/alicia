package org.localforge.alicia.core.common

import timber.log.Timber

object Logger {

    private const val TAG_PREFIX = "Alicia"
    private var isDebugMode = true

    enum class Level {
        VERBOSE, DEBUG, INFO, WARN, ERROR
    }

    var minLogLevel: Level = Level.VERBOSE

    fun setDebugMode(enabled: Boolean) {
        isDebugMode = enabled
    }

    fun v(tag: String, message: String, throwable: Throwable? = null) {
        if (shouldLog(Level.VERBOSE)) {
            val prefixedTag = "$TAG_PREFIX:$tag"
            if (throwable != null) {
                Timber.tag(prefixedTag).v(throwable, message)
            } else {
                Timber.tag(prefixedTag).v(message)
            }
        }
    }

    fun d(tag: String, message: String, throwable: Throwable? = null) {
        if (shouldLog(Level.DEBUG)) {
            val prefixedTag = "$TAG_PREFIX:$tag"
            if (throwable != null) {
                Timber.tag(prefixedTag).d(throwable, message)
            } else {
                Timber.tag(prefixedTag).d(message)
            }
        }
    }

    fun i(tag: String, message: String, throwable: Throwable? = null) {
        if (shouldLog(Level.INFO)) {
            val prefixedTag = "$TAG_PREFIX:$tag"
            if (throwable != null) {
                Timber.tag(prefixedTag).i(throwable, message)
            } else {
                Timber.tag(prefixedTag).i(message)
            }
        }
    }

    fun w(tag: String, message: String, throwable: Throwable? = null) {
        if (shouldLog(Level.WARN)) {
            val prefixedTag = "$TAG_PREFIX:$tag"
            if (throwable != null) {
                Timber.tag(prefixedTag).w(throwable, message)
            } else {
                Timber.tag(prefixedTag).w(message)
            }
        }
    }

    fun e(tag: String, message: String, throwable: Throwable? = null) {
        if (shouldLog(Level.ERROR)) {
            val prefixedTag = "$TAG_PREFIX:$tag"
            if (throwable != null) {
                Timber.tag(prefixedTag).e(throwable, message)
            } else {
                Timber.tag(prefixedTag).e(message)
            }
        }
    }

    // wtf() bypasses log level filtering as it indicates critical failures
    fun wtf(tag: String, message: String, throwable: Throwable? = null) {
        val prefixedTag = "$TAG_PREFIX:$tag"
        if (throwable != null) {
            Timber.tag(prefixedTag).wtf(throwable, message)
        } else {
            Timber.tag(prefixedTag).wtf(message)
        }
    }

    private fun shouldLog(level: Level): Boolean {
        return isDebugMode && level.ordinal >= minLogLevel.ordinal
    }

    fun forClass(clazz: Class<*>): TaggedLogger {
        return TaggedLogger(clazz.simpleName)
    }

    fun forTag(tag: String): TaggedLogger {
        return TaggedLogger(tag)
    }

    class TaggedLogger(private val tag: String) {

        fun v(message: String, throwable: Throwable? = null) {
            Logger.v(tag, message, throwable)
        }

        fun d(message: String, throwable: Throwable? = null) {
            Logger.d(tag, message, throwable)
        }

        fun i(message: String, throwable: Throwable? = null) {
            Logger.i(tag, message, throwable)
        }

        fun w(message: String, throwable: Throwable? = null) {
            Logger.w(tag, message, throwable)
        }

        fun e(message: String, throwable: Throwable? = null) {
            Logger.e(tag, message, throwable)
        }

        fun wtf(message: String, throwable: Throwable? = null) {
            Logger.wtf(tag, message, throwable)
        }

        fun enter(methodName: String, vararg params: Any?) {
            if (shouldLog(Level.VERBOSE)) {
                val paramStr = params.joinToString(", ") { it.toString() }
                v("→ $methodName($paramStr)")
            }
        }

        fun exit(methodName: String, result: Any? = null) {
            if (shouldLog(Level.VERBOSE)) {
                if (result != null) {
                    v("← $methodName: $result")
                } else {
                    v("← $methodName")
                }
            }
        }

        fun time(operation: String, durationMs: Long) {
            if (shouldLog(Level.DEBUG)) {
                d("⏱ $operation took ${durationMs}ms")
            }
        }

        fun state(stateName: String, state: Any?) {
            if (shouldLog(Level.DEBUG)) {
                d("State[$stateName] = $state")
            }
        }
    }
}

inline fun <T> Logger.TaggedLogger.measureTime(operation: String, block: () -> T): T {
    val start = System.currentTimeMillis()
    val result = block()
    val duration = System.currentTimeMillis() - start
    time(operation, duration)
    return result
}
