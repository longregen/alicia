import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useDatabase } from './useDatabase';
import * as sqlite from '../db/sqlite';

vi.mock('../db/sqlite', () => ({
  initDatabase: vi.fn(),
  loadFromIndexedDB: vi.fn(),
}));

describe('useDatabase', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (sqlite.initDatabase as any).mockResolvedValue({});
    (sqlite.loadFromIndexedDB as any).mockResolvedValue(undefined);
  });

  it('should initialize database on mount', async () => {
    renderHook(() => useDatabase());

    await waitFor(() => {
      expect(sqlite.initDatabase).toHaveBeenCalled();
    });
  });

  it('should load from IndexedDB after initialization', async () => {
    renderHook(() => useDatabase());

    await waitFor(() => {
      expect(sqlite.loadFromIndexedDB).toHaveBeenCalled();
    });

    // Verify order: init before load
    const initOrder = (sqlite.initDatabase as any).mock.invocationCallOrder[0];
    const loadOrder = (sqlite.loadFromIndexedDB as any).mock.invocationCallOrder[0];
    expect(initOrder).toBeLessThan(loadOrder);
  });

  it('should set isReady to true after successful initialization', async () => {
    const { result } = renderHook(() => useDatabase());

    expect(result.current.isReady).toBe(false);

    await waitFor(() => {
      expect(result.current.isReady).toBe(true);
    });

    expect(result.current.error).toBeNull();
  });

  it('should handle initialization errors', async () => {
    const error = new Error('Failed to initialize');
    (sqlite.initDatabase as any).mockRejectedValue(error);

    const { result } = renderHook(() => useDatabase());

    await waitFor(() => {
      expect(result.current.error).toEqual(error);
    });

    expect(result.current.isReady).toBe(false);
  });

  it('should handle loading errors', async () => {
    const error = new Error('Failed to load from IndexedDB');
    (sqlite.loadFromIndexedDB as any).mockRejectedValue(error);

    const { result } = renderHook(() => useDatabase());

    await waitFor(() => {
      expect(result.current.error).toEqual(error);
    });

    expect(result.current.isReady).toBe(false);
  });

  it('should handle non-Error exceptions', async () => {
    (sqlite.initDatabase as any).mockRejectedValue('String error');

    const { result } = renderHook(() => useDatabase());

    await waitFor(() => {
      expect(result.current.error).toBeInstanceOf(Error);
      expect(result.current.error?.message).toBe('Failed to initialize database');
    });
  });

  it('should not update state after unmount', async () => {
    let resolveFn: () => void;
    const promise = new Promise<void>((resolve) => {
      resolveFn = resolve;
    });

    (sqlite.initDatabase as any).mockReturnValue(promise);

    const { result, unmount } = renderHook(() => useDatabase());

    // Unmount before initialization completes
    unmount();

    // Resolve the promise
    resolveFn!();

    await new Promise(resolve => setTimeout(resolve, 10));

    // State should not have changed
    expect(result.current.isReady).toBe(false);
  });

  it('should initialize only once per mount', async () => {
    const { rerender } = renderHook(() => useDatabase());

    await waitFor(() => {
      expect(sqlite.initDatabase).toHaveBeenCalledTimes(1);
    });

    // Rerender should not cause re-initialization
    rerender();

    await new Promise(resolve => setTimeout(resolve, 10));

    expect(sqlite.initDatabase).toHaveBeenCalledTimes(1);
  });

  it('should reinitialize on remount', async () => {
    const { unmount: unmount1 } = renderHook(() => useDatabase());

    await waitFor(() => {
      expect(sqlite.initDatabase).toHaveBeenCalledTimes(1);
    });

    unmount1();

    // Mount again
    renderHook(() => useDatabase());

    await waitFor(() => {
      expect(sqlite.initDatabase).toHaveBeenCalledTimes(2);
    });
  });

  it('should handle slow initialization gracefully', async () => {
    let resolveFn: () => void;
    const slowPromise = new Promise<void>((resolve) => {
      resolveFn = resolve;
    });

    (sqlite.initDatabase as any).mockReturnValue(slowPromise);

    const { result } = renderHook(() => useDatabase());

    // Should be loading initially
    expect(result.current.isReady).toBe(false);
    expect(result.current.error).toBeNull();

    // Resolve after some time
    setTimeout(() => resolveFn!(), 100);

    await waitFor(() => {
      expect(result.current.isReady).toBe(true);
    }, { timeout: 200 });
  });

  it('should return consistent hook result structure', () => {
    const { result } = renderHook(() => useDatabase());

    expect(result.current).toHaveProperty('isReady');
    expect(result.current).toHaveProperty('error');
    expect(typeof result.current.isReady).toBe('boolean');
    expect(result.current.error === null || result.current.error instanceof Error).toBe(true);
  });
});
