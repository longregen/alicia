import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useAsync } from './useAsync';

describe('useAsync', () => {
  it('should initialize with loading false and no error', () => {
    const asyncFn = vi.fn().mockResolvedValue('success');
    const { result } = renderHook(() => useAsync(asyncFn));

    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBe(null);
  });

  it('should set loading to true during execution', async () => {
    const asyncFn = vi.fn().mockImplementation(() =>
      new Promise(resolve => setTimeout(() => resolve('success'), 100))
    );
    const { result } = renderHook(() => useAsync(asyncFn));

    let promise: Promise<any>;
    act(() => {
      promise = result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(true);
    });

    await act(async () => {
      await promise;
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
  });

  it('should handle successful execution', async () => {
    const asyncFn = vi.fn().mockResolvedValue('success');
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useAsync(asyncFn, { onSuccess }));

    let returnValue: any;
    await act(async () => {
      returnValue = await result.current.execute();
    });

    expect(returnValue).toBe('success');
    expect(onSuccess).toHaveBeenCalledWith('success');
    expect(result.current.error).toBe(null);
    expect(result.current.loading).toBe(false);
  });

  it('should handle errors and set error message', async () => {
    const error = new Error('Test error');
    const asyncFn = vi.fn().mockRejectedValue(error);
    const { result } = renderHook(() => useAsync(asyncFn));

    let returnValue: any;
    await act(async () => {
      returnValue = await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('Test error');
    });
    expect(returnValue).toBe(null);
    expect(result.current.loading).toBe(false);
  });

  it('should use custom error message when provided', async () => {
    const error = new Error('Original error');
    const asyncFn = vi.fn().mockRejectedValue(error);
    const { result } = renderHook(() =>
      useAsync(asyncFn, { errorMessage: 'Custom error' })
    );

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('Custom error');
    });
  });

  it('should handle non-Error thrown values', async () => {
    const asyncFn = vi.fn().mockRejectedValue('string error');
    const { result } = renderHook(() => useAsync(asyncFn));

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('An error occurred');
    });
  });

  it('should pass arguments to the async function', async () => {
    const asyncFn = vi.fn().mockResolvedValue('success');
    const { result } = renderHook(() => useAsync(asyncFn));

    await act(async () => {
      await result.current.execute('arg1', 'arg2', 123);
    });

    expect(asyncFn).toHaveBeenCalledWith('arg1', 'arg2', 123);
  });

  it('should clear error when clearError is called', async () => {
    const asyncFn = vi.fn().mockRejectedValue(new Error('Test error'));
    const { result } = renderHook(() => useAsync(asyncFn));

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('Test error');
    });

    act(() => {
      result.current.clearError();
    });

    await waitFor(() => {
      expect(result.current.error).toBe(null);
    });
  });

  it('should clear error on new execution by default', async () => {
    const asyncFn = vi.fn()
      .mockRejectedValueOnce(new Error('First error'))
      .mockResolvedValueOnce('success');

    const { result } = renderHook(() => useAsync(asyncFn));

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('First error');
    });

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe(null);
    });
  });

  it('should not clear error on new execution when clearErrorOnExecute is false', async () => {
    const asyncFn = vi.fn()
      .mockRejectedValueOnce(new Error('First error'))
      .mockResolvedValueOnce('success');

    const { result } = renderHook(() =>
      useAsync(asyncFn, { clearErrorOnExecute: false })
    );

    await act(async () => {
      await result.current.execute();
    });

    await waitFor(() => {
      expect(result.current.error).toBe('First error');
    });

    let promise: Promise<any>;
    act(() => {
      promise = result.current.execute();
    });

    // Error should still be set initially
    await waitFor(() => {
      expect(result.current.error).toBe('First error');
    });

    // Wait for completion
    await act(async () => {
      await promise;
    });

    // After success, error should be cleared anyway
    await waitFor(() => {
      expect(result.current.error).toBe(null);
    });
  });

  it('should allow manual control of loading state', async () => {
    const asyncFn = vi.fn().mockResolvedValue('success');
    const { result } = renderHook(() => useAsync(asyncFn));

    expect(result.current.loading).toBe(false);

    act(() => {
      result.current.setLoading(true);
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(true);
    });

    act(() => {
      result.current.setLoading(false);
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
  });

  it('should handle multiple concurrent executions', async () => {
    let resolveCount = 0;
    const asyncFn = vi.fn().mockImplementation(() =>
      new Promise(resolve => {
        setTimeout(() => {
          resolveCount++;
          resolve(`success-${resolveCount}`);
        }, 50);
      })
    );

    const { result } = renderHook(() => useAsync(asyncFn));

    let promise1: Promise<any>;
    let promise2: Promise<any>;
    act(() => {
      promise1 = result.current.execute();
      promise2 = result.current.execute();
    });

    const [result1, result2] = await act(async () => {
      return await Promise.all([promise1, promise2]);
    });

    expect(asyncFn).toHaveBeenCalledTimes(2);
    expect(result1).toBe('success-1');
    expect(result2).toBe('success-2');
  });
});
