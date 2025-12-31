import { useState, useCallback, useRef } from 'react';

export interface UseAsyncOptions<T> {
  /**
   * Callback to execute on successful completion
   */
  onSuccess?: (data: T) => void;

  /**
   * Custom error message to use instead of the error's message
   */
  errorMessage?: string;

  /**
   * Whether to clear error on new execution
   * @default true
   */
  clearErrorOnExecute?: boolean;
}

export interface UseAsyncReturn<T, Args extends unknown[] = unknown[]> {
  /**
   * Whether the async operation is currently in progress
   */
  loading: boolean;

  /**
   * Error message if the operation failed
   */
  error: string | null;

  /**
   * Execute the async operation
   * Returns the result or null if an error occurred
   */
  execute: (...args: Args) => Promise<T | null>;

  /**
   * Clear the error state
   */
  clearError: () => void;

  /**
   * Set loading state manually (useful for external control)
   */
  setLoading: (loading: boolean) => void;
}

/**
 * A generic hook for handling async operations with loading and error states
 *
 * @example
 * ```typescript
 * const { loading, error, execute } = useAsync(
 *   async (id: string) => api.getConversation(id),
 *   {
 *     onSuccess: (conversation) => setCurrentConversation(conversation),
 *     errorMessage: 'Failed to fetch conversation'
 *   }
 * );
 *
 * // Later...
 * await execute('conversation-id');
 * ```
 */
export function useAsync<T, Args extends unknown[] = unknown[]>(
  asyncFunction: (...args: Args) => Promise<T>,
  options: UseAsyncOptions<T> = {}
): UseAsyncReturn<T, Args> {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Use refs to avoid recreating execute when function/options change
  const asyncFunctionRef = useRef(asyncFunction);
  const optionsRef = useRef(options);

  // Update refs on each render
  asyncFunctionRef.current = asyncFunction;
  optionsRef.current = options;

  const execute = useCallback(
    async (...args: Args): Promise<T | null> => {
      const { onSuccess, errorMessage, clearErrorOnExecute = true } = optionsRef.current;
      try {
        setLoading(true);
        if (clearErrorOnExecute) {
          setError(null);
        }

        const result = await asyncFunctionRef.current(...args);

        if (onSuccess) {
          onSuccess(result);
        }

        // Ensure error is cleared on success (defensive programming)
        setError(null);
        return result;
      } catch (err) {
        const errorMsg = errorMessage ||
          (err instanceof Error ? err.message : 'An error occurred');
        setError(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    loading,
    error,
    execute,
    clearError,
    setLoading,
  };
}
