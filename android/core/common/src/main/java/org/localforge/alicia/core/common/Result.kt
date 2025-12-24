package org.localforge.alicia.core.common

/**
 * A sealed class representing the result of an operation.
 * Can be either Success with data, Error with exception, or Loading state.
 */
sealed class Result<out T> {
    /**
     * Success state with data
     */
    data class Success<T>(val data: T) : Result<T>()

    /**
     * Error state with exception and optional message
     */
    data class Error(
        val exception: Throwable,
        val message: String? = exception.message
    ) : Result<Nothing>()

    /**
     * Loading state
     */
    data object Loading : Result<Nothing>()

    /**
     * Check if result is success
     */
    val isSuccess: Boolean
        get() = this is Success

    /**
     * Check if result is error
     */
    val isError: Boolean
        get() = this is Error

    /**
     * Check if result is loading
     */
    val isLoading: Boolean
        get() = this is Loading

    /**
     * Get data if success, null otherwise
     */
    fun getOrNull(): T? = when (this) {
        is Success -> data
        else -> null
    }

    /**
     * Get data if success, default value otherwise
     */
    fun getOrDefault(default: @UnsafeVariance T): T = when (this) {
        is Success -> data
        else -> default
    }

    /**
     * Get exception if error, null otherwise
     */
    fun exceptionOrNull(): Throwable? = when (this) {
        is Error -> exception
        else -> null
    }

    /**
     * Get message if error, null otherwise
     */
    fun messageOrNull(): String? = when (this) {
        is Error -> message
        else -> null
    }

    /**
     * Transform the data if success
     */
    inline fun <R> map(transform: (T) -> R): Result<R> = when (this) {
        is Success -> Success(transform(data))
        is Error -> this
        is Loading -> this
    }

    /**
     * Flat map transformation
     */
    inline fun <R> flatMap(transform: (T) -> Result<R>): Result<R> = when (this) {
        is Success -> transform(data)
        is Error -> this
        is Loading -> this
    }

    /**
     * Execute block if success
     */
    inline fun onSuccess(block: (T) -> Unit): Result<T> {
        if (this is Success) {
            block(data)
        }
        return this
    }

    /**
     * Execute block if error
     */
    inline fun onError(block: (Throwable, String?) -> Unit): Result<T> {
        if (this is Error) {
            block(exception, message)
        }
        return this
    }

    /**
     * Execute block if loading
     */
    inline fun onLoading(block: () -> Unit): Result<T> {
        if (this is Loading) {
            block()
        }
        return this
    }

    companion object {
        /**
         * Create a success result
         */
        fun <T> success(data: T): Result<T> = Success(data)

        /**
         * Create an error result
         */
        fun error(exception: Throwable, message: String? = null): Result<Nothing> =
            Error(exception, message ?: exception.message)

        /**
         * Create a loading result
         */
        fun loading(): Result<Nothing> = Loading

        /**
         * Wrap a suspending function in a Result
         */
        suspend inline fun <T> wrap(crossinline block: suspend () -> T): Result<T> {
            return try {
                Success(block())
            } catch (e: Exception) {
                Error(e)
            }
        }

        /**
         * Wrap a regular function in a Result
         */
        inline fun <T> wrapSync(crossinline block: () -> T): Result<T> {
            return try {
                Success(block())
            } catch (e: Exception) {
                Error(e)
            }
        }
    }
}

/**
 * Extension to convert nullable value to Result
 */
fun <T : Any> T?.toResult(errorMessage: String = "Value is null"): Result<T> {
    return if (this != null) {
        Result.Success(this)
    } else {
        Result.Error(NullPointerException(errorMessage), errorMessage)
    }
}

/**
 * Extension to convert Kotlin Result to our Result
 */
fun <T> kotlin.Result<T>.toAppResult(): Result<T> {
    return fold(
        onSuccess = { Result.Success(it) },
        onFailure = { Result.Error(it) }
    )
}
