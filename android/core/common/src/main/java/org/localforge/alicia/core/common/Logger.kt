package org.localforge.alicia.core.common

import timber.log.Timber

/**
 * Centralized logging utility for the Alicia app.
 * Provides consistent logging with automatic tag prefixing and log level control.
 */
object Logger {

    private const val TAG_PREFIX = "Alicia"
    private var isDebugMode = true

    /**
     * Log levels
     */
    enum class Level {
        VERBOSE, DEBUG, INFO, WARN, ERROR
    }

    /**
     * Minimum log level to display
     */
    var minLogLevel: Level = Level.VERBOSE

    /**
     * Enable or disable debug mode
     */
    fun setDebugMode(enabled: Boolean) {
        isDebugMode = enabled
    }

    /**
     * Log verbose message
     */
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

    /**
     * Log debug message
     */
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

    /**
     * Log info message
     */
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

    /**
     * Log warning message
     */
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

    /**
     * Log error message
     */
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

    /**
     * Log What-A-Terrible-Failure (wtf)
     * Note: wtf() bypasses log level filtering as it indicates critical failures. However, it still requires Timber to be configured.
     */
    fun wtf(tag: String, message: String, throwable: Throwable? = null) {
        val prefixedTag = "$TAG_PREFIX:$tag"
        if (throwable != null) {
            Timber.tag(prefixedTag).wtf(throwable, message)
        } else {
            Timber.tag(prefixedTag).wtf(message)
        }
    }

    /**
     * Check if should log at given level
     */
    private fun shouldLog(level: Level): Boolean {
        return isDebugMode && level.ordinal >= minLogLevel.ordinal
    }

    /**
     * Create a tagged logger instance for a specific class
     */
    fun forClass(clazz: Class<*>): TaggedLogger {
        return TaggedLogger(clazz.simpleName)
    }

    /**
     * Create a tagged logger instance with custom tag
     */
    fun forTag(tag: String): TaggedLogger {
        return TaggedLogger(tag)
    }

    /**
     * Tagged logger for convenient usage
     */
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

        /**
         * Log method entry with parameters
         */
        fun enter(methodName: String, vararg params: Any?) {
            if (shouldLog(Level.VERBOSE)) {
                val paramStr = params.joinToString(", ") { it.toString() }
                v("→ $methodName($paramStr)")
            }
        }

        /**
         * Log method exit with result
         */
        fun exit(methodName: String, result: Any? = null) {
            if (shouldLog(Level.VERBOSE)) {
                if (result != null) {
                    v("← $methodName: $result")
                } else {
                    v("← $methodName")
                }
            }
        }

        /**
         * Log timing information
         */
        fun time(operation: String, durationMs: Long) {
            if (shouldLog(Level.DEBUG)) {
                d("⏱ $operation took ${durationMs}ms")
            }
        }

        /**
         * Log object state
         */
        fun state(stateName: String, state: Any?) {
            if (shouldLog(Level.DEBUG)) {
                d("State[$stateName] = $state")
            }
        }
    }
}

/**
 * Measures and logs the execution time of a code block.
 *
 * This extension function executes the provided block, measures how long it takes,
 * and automatically logs the duration using the TaggedLogger's time() method.
 *
 * Note: Logging only occurs if debug mode is enabled and minimum log level is DEBUG or more verbose (VERBOSE).
 *
 * @param operation A descriptive name for the operation being measured, used in the log message
 * @param block The code block to execute and measure
 * @return The result of executing the block
 *
 * @sample
 * ```
 * val logger = Logger.forClass(MyClass::class.java)
 * val result = logger.measureTime("Database query") {
 *     database.query()
 * }
 * ```
 */
inline fun <T> Logger.TaggedLogger.measureTime(operation: String, block: () -> T): T {
    val start = System.currentTimeMillis()
    val result = block()
    val duration = System.currentTimeMillis() - start
    time(operation, duration)
    return result
}
