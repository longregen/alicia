import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useSync } from './useSync';
import { useWebSocketSync } from './useWebSocketSync';

// Mock dependencies
vi.mock('./useWebSocketSync');

const mockUseWebSocketSync = useWebSocketSync as ReturnType<typeof vi.mocked<typeof useWebSocketSync>>;

describe('useSync', () => {
  const mockSyncNow = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    mockUseWebSocketSync.mockReturnValue({
      isConnected: false,
      error: null,
      send: vi.fn(),
      syncNow: mockSyncNow,
    });
  });

  it('should initialize with WebSocket sync when conversationId is provided', () => {
    renderHook(() => useSync('conv1'));

    expect(mockUseWebSocketSync).toHaveBeenCalledWith('conv1', {
      onSync: expect.any(Function),
      onMessage: expect.any(Function),
      enabled: true,
    });
  });

  it('should return sync state from WebSocket hook', () => {
    mockUseWebSocketSync.mockReturnValue({
      isConnected: true,
      error: null,
      send: vi.fn(),
      syncNow: mockSyncNow,
    });

    const { result } = renderHook(() => useSync('conv1'));

    expect(result.current.isSSEConnected).toBe(true);
    expect(result.current.syncError).toBe(null);
    expect(result.current.isSyncing).toBe(false);
  });

  it('should handle WebSocket connection errors', () => {
    const mockError = new Error('WebSocket connection error');
    mockUseWebSocketSync.mockReturnValue({
      isConnected: false,
      error: mockError,
      send: vi.fn(),
      syncNow: mockSyncNow,
    });

    const { result } = renderHook(() => useSync('conv1'));

    expect(result.current.syncError).toBe('WebSocket connection error');
    expect(result.current.isSSEConnected).toBe(false);
  });

  it('should update lastSyncTime when onSync callback is called', () => {
    let capturedOnSync: (() => void) | undefined;

    mockUseWebSocketSync.mockImplementation((_conversationId, options) => {
      capturedOnSync = options?.onSync;
      return {
        isConnected: true,
        error: null,
        send: vi.fn(),
        syncNow: mockSyncNow,
      };
    });

    const { result } = renderHook(() => useSync('conv1'));

    expect(result.current.lastSyncTime).toBe(null);

    // Simulate sync completion
    act(() => {
      if (capturedOnSync) capturedOnSync();
    });

    expect(result.current.lastSyncTime).toBeInstanceOf(Date);
  });

  it('should expose syncNow function from WebSocket hook', () => {
    mockUseWebSocketSync.mockReturnValue({
      isConnected: true,
      error: null,
      send: vi.fn(),
      syncNow: mockSyncNow,
    });

    const { result } = renderHook(() => useSync('conv1'));

    act(() => {
      result.current.syncNow();
    });

    expect(mockSyncNow).toHaveBeenCalled();
  });

  it('should handle null conversationId', () => {
    renderHook(() => useSync(null));

    expect(mockUseWebSocketSync).toHaveBeenCalledWith(null, {
      onSync: expect.any(Function),
      onMessage: expect.any(Function),
      enabled: false,
    });
  });

  it('should disable sync when conversationId is null', () => {
    const { result } = renderHook(() => useSync(null));

    expect(result.current.isSSEConnected).toBe(false);
    expect(result.current.syncError).toBe(null);
  });

  it('should reconnect when conversationId changes', () => {
    const { rerender } = renderHook(
      ({ conversationId }) => useSync(conversationId),
      { initialProps: { conversationId: 'conv1' } }
    );

    expect(mockUseWebSocketSync).toHaveBeenCalledWith('conv1', {
      onSync: expect.any(Function),
      onMessage: expect.any(Function),
      enabled: true,
    });

    rerender({ conversationId: 'conv2' });

    expect(mockUseWebSocketSync).toHaveBeenCalledWith('conv2', {
      onSync: expect.any(Function),
      onMessage: expect.any(Function),
      enabled: true,
    });
  });
});
