import { useState, useCallback, useRef } from 'react';

export interface UseAsyncOptions<T> {
  onSuccess?: (data: T) => void;
  errorMessage?: string;
  /** @default true */
  clearErrorOnExecute?: boolean;
}

export interface UseAsyncReturn<T, Args extends unknown[] = unknown[]> {
  loading: boolean;
  error: string | null;
  execute: (...args: Args) => Promise<T | null>;
  clearError: () => void;
  setLoading: (loading: boolean) => void;
}

export function useAsync<T, Args extends unknown[] = unknown[]>(
  asyncFunction: (...args: Args) => Promise<T>,
  options: UseAsyncOptions<T> = {}
): UseAsyncReturn<T, Args> {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Refs avoid recreating execute callback when function/options change
  const asyncFunctionRef = useRef(asyncFunction);
  const optionsRef = useRef(options);
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
