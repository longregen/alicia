package org.localforge.alicia.core.common

sealed class Result<out T> {

    data class Success<T>(val data: T) : Result<T>()

    data class Error(
        val exception: Throwable,
        val message: String? = exception.message
    ) : Result<Nothing>()

    data object Loading : Result<Nothing>()

    val isSuccess: Boolean
        get() = this is Success

    val isError: Boolean
        get() = this is Error

    val isLoading: Boolean
        get() = this is Loading

    fun getOrNull(): T? = when (this) {
        is Success -> data
        else -> null
    }

    fun getOrDefault(default: @UnsafeVariance T): T = when (this) {
        is Success -> data
        else -> default
    }

    fun exceptionOrNull(): Throwable? = when (this) {
        is Error -> exception
        else -> null
    }

    fun messageOrNull(): String? = when (this) {
        is Error -> message
        else -> null
    }

    inline fun <R> map(transform: (T) -> R): Result<R> = when (this) {
        is Success -> Success(transform(data))
        is Error -> this
        is Loading -> this
    }

    inline fun <R> flatMap(transform: (T) -> Result<R>): Result<R> = when (this) {
        is Success -> transform(data)
        is Error -> this
        is Loading -> this
    }

    inline fun onSuccess(block: (T) -> Unit): Result<T> {
        if (this is Success) {
            block(data)
        }
        return this
    }

    inline fun onError(block: (Throwable, String?) -> Unit): Result<T> {
        if (this is Error) {
            block(exception, message)
        }
        return this
    }

    inline fun onLoading(block: () -> Unit): Result<T> {
        if (this is Loading) {
            block()
        }
        return this
    }

    companion object {

        fun <T> success(data: T): Result<T> = Success(data)

        fun error(exception: Throwable, message: String? = null): Result<Nothing> =
            Error(exception, message ?: exception.message)

        fun loading(): Result<Nothing> = Loading

        suspend inline fun <T> wrap(crossinline block: suspend () -> T): Result<T> {
            return try {
                Success(block())
            } catch (e: Exception) {
                Error(e)
            }
        }

        inline fun <T> wrapSync(crossinline block: () -> T): Result<T> {
            return try {
                Success(block())
            } catch (e: Exception) {
                Error(e)
            }
        }
    }
}

fun <T : Any> T?.toResult(errorMessage: String = "Value is null"): Result<T> {
    return if (this != null) {
        Result.Success(this)
    } else {
        Result.Error(NullPointerException(errorMessage), errorMessage)
    }
}

fun <T> kotlin.Result<T>.toAppResult(): Result<T> {
    return fold(
        onSuccess = { Result.Success(it) },
        onFailure = { Result.Error(it) }
    )
}
