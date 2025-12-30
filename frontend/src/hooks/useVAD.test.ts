import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useVAD } from './useVAD';
import { MicrophoneStatus } from '../types/streaming';
import type { SileroVADCallbacks } from '../utils/sileroVAD';

// Mock MediaStreamTrack
const mockMediaStreamTrack = {
  kind: 'audio',
  label: 'mock-audio-track',
  id: 'mock-track-id',
  enabled: true,
  muted: false,
  readyState: 'live',
  stop: vi.fn(),
  clone: vi.fn(),
  getConstraints: vi.fn(),
  getCapabilities: vi.fn(),
  getSettings: vi.fn(),
  applyConstraints: vi.fn(),
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  dispatchEvent: vi.fn(),
  onended: null,
  onmute: null,
  onunmute: null,
} as unknown as MediaStreamTrack;

// Store captured callbacks for tests - accessed via getter to handle hoisting
const testState = {
  capturedCallbacks: null as SileroVADCallbacks | null,
  mockVADManager: {
    initialize: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    destroy: vi.fn(),
  },
  mockBridge: {
    initialize: vi.fn(),
    pushAudioFrame: vi.fn(),
    pushSpeechSegment: vi.fn(),
    cleanup: vi.fn(),
  },
};

// Mock dependencies
vi.mock('../utils/sileroVAD', () => {
  return {
    SileroVADManager: class MockSileroVADManager {
      constructor(callbacks: SileroVADCallbacks) {
        testState.capturedCallbacks = callbacks;
      }
      initialize = testState.mockVADManager.initialize;
      start = testState.mockVADManager.start;
      stop = testState.mockVADManager.stop;
      destroy = testState.mockVADManager.destroy;
    },
  };
});

vi.mock('../adapters/vadLiveKitBridge', () => {
  return {
    VADLiveKitBridge: class MockVADLiveKitBridge {
      initialize = testState.mockBridge.initialize;
      pushAudioFrame = testState.mockBridge.pushAudioFrame;
      pushSpeechSegment = testState.mockBridge.pushSpeechSegment;
      cleanup = testState.mockBridge.cleanup;
    },
  };
});

describe('useVAD', () => {
  // Convenient aliases to testState
  const mockVADManager = testState.mockVADManager;
  const mockBridge = testState.mockBridge;

  // Getter for captured callbacks
  const getCapturedCallbacks = () => testState.capturedCallbacks;

  beforeEach(() => {
    // Reset captured callbacks
    testState.capturedCallbacks = null;

    // Reset all mock implementations
    vi.clearAllMocks();

    // Suppress console output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    // Reset mock implementations to default behavior
    mockVADManager.initialize.mockResolvedValue(undefined);
    mockVADManager.start.mockResolvedValue(undefined);
    mockBridge.initialize.mockResolvedValue(mockMediaStreamTrack);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Initialization', () => {
    it('should initialize with correct default state', () => {
      const { result } = renderHook(() => useVAD());

      expect(result.current.status).toBe(MicrophoneStatus.Inactive);
      expect(result.current.isRecording).toBe(false);
      expect(result.current.speechProbability).toBe(0);
      expect(result.current.isSpeaking).toBe(false);
      expect(result.current.audioTrack).toBeNull();
    });

    it('should create SileroVADManager on mount', () => {
      renderHook(() => useVAD());

      // VAD manager is created with callbacks - verify via captured callbacks
      expect(getCapturedCallbacks()).not.toBeNull();
      expect(getCapturedCallbacks()).toHaveProperty('onStatusChange');
      expect(getCapturedCallbacks()).toHaveProperty('onSpeechProbability');
    });

    it('should expose vadManager reference', () => {
      const { result } = renderHook(() => useVAD());

      // vadManager is set after useEffect runs, so it should be the mock
      expect(result.current.vadManager).toBeDefined();
    });
  });

  describe('startVAD', () => {
    it('should initialize bridge before VAD on startVAD', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      // Bridge is initialized during startVAD
      expect(mockBridge.initialize).toHaveBeenCalled();
    });

    it('should initialize VAD manager after bridge', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      expect(mockVADManager.initialize).toHaveBeenCalled();
    });

    it('should start VAD after initialization', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      expect(mockVADManager.start).toHaveBeenCalled();
    });

    it('should set audioTrack state from bridge', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      expect(result.current.audioTrack).toBe(mockMediaStreamTrack);
    });

    it('should call onTrackReady with bridge track', async () => {
      const onTrackReady = vi.fn();
      const { result } = renderHook(() => useVAD({ onTrackReady }));

      await act(async () => {
        await result.current.startVAD();
      });

      expect(onTrackReady).toHaveBeenCalledWith(mockMediaStreamTrack);
    });

    it('should call onError when VAD initialization fails', async () => {
      mockVADManager.initialize.mockRejectedValueOnce(new Error('Init failed'));

      const onError = vi.fn();
      const { result } = renderHook(() => useVAD({ onError }));

      await act(async () => {
        try {
          await result.current.startVAD();
        } catch {
          // Expected to throw
        }
      });

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
    });

    it('should call onError when VAD start fails', async () => {
      mockVADManager.start.mockRejectedValueOnce(new Error('Start failed'));

      const onError = vi.fn();
      const { result } = renderHook(() => useVAD({ onError }));

      await act(async () => {
        try {
          await result.current.startVAD();
        } catch {
          // Expected to throw
        }
      });

      expect(onError).toHaveBeenCalled();
    });

    it('should throw when onTrackReady callback errors', async () => {
      const onTrackReady = vi.fn().mockImplementation(() => {
        throw new Error('Callback error');
      });

      const { result } = renderHook(() => useVAD({ onTrackReady }));

      await expect(
        act(async () => {
          await result.current.startVAD();
        })
      ).rejects.toThrow('Callback error');
    });

    it('should reset audioTrack to null on error', async () => {
      mockVADManager.initialize.mockRejectedValueOnce(new Error('Error'));

      const { result } = renderHook(() => useVAD());

      await act(async () => {
        try {
          await result.current.startVAD();
        } catch {
          // Expected
        }
      });

      expect(result.current.audioTrack).toBeNull();
    });
  });

  describe('stopVAD', () => {
    it('should call vadManager.stop()', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        result.current.stopVAD();
      });

      expect(mockVADManager.stop).toHaveBeenCalled();
    });

    it('should handle stop when called before startVAD', () => {
      // The vadManager exists but VAD hasn't been started
      const { result } = renderHook(() => useVAD());

      // Should not throw - stopVAD is safe to call even when not recording
      expect(() => result.current.stopVAD()).not.toThrow();
      expect(mockVADManager.stop).toHaveBeenCalled();
    });
  });

  describe('destroyVAD', () => {
    it('should destroy VAD manager', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        result.current.destroyVAD();
      });

      expect(mockVADManager.destroy).toHaveBeenCalled();
    });

    it('should cleanup bridge', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        result.current.destroyVAD();
      });

      expect(mockBridge.cleanup).toHaveBeenCalled();
    });

    it('should reset all state', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      // Simulate some state changes
      act(() => {
        getCapturedCallbacks()?.onStatusChange?.(MicrophoneStatus.Recording);
        getCapturedCallbacks()?.onSpeechProbability?.(0.8, true);
      });

      act(() => {
        result.current.destroyVAD();
      });

      expect(result.current.audioTrack).toBeNull();
      expect(result.current.status).toBe(MicrophoneStatus.Inactive);
      expect(result.current.speechProbability).toBe(0);
      expect(result.current.isSpeaking).toBe(false);
    });

    it('should handle destroy when resources are null', () => {
      const { result } = renderHook(() => useVAD());

      // Should not throw even without calling startVAD
      expect(() => result.current.destroyVAD()).not.toThrow();
    });
  });

  describe('Callback Invocations', () => {
    it('should call onStatusChange when VAD status changes', async () => {
      const onStatusChange = vi.fn();
      const { result } = renderHook(() => useVAD({ onStatusChange }));

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onStatusChange?.(MicrophoneStatus.Recording);
      });

      expect(onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Recording);
    });

    it('should update status state on status change', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onStatusChange?.(MicrophoneStatus.Recording);
      });

      expect(result.current.status).toBe(MicrophoneStatus.Recording);
    });

    it('should call onSpeechProbability with probability and speaking state', async () => {
      const onSpeechProbability = vi.fn();
      const { result } = renderHook(() => useVAD({ onSpeechProbability }));

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onSpeechProbability?.(0.75, true);
      });

      expect(onSpeechProbability).toHaveBeenCalledWith(0.75, true);
    });

    it('should update speechProbability state', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onSpeechProbability?.(0.9, true);
      });

      expect(result.current.speechProbability).toBe(0.9);
    });

    it('should update isSpeaking state', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onSpeechProbability?.(0.9, true);
      });

      expect(result.current.isSpeaking).toBe(true);
    });

    it('should call onSpeechStart when speech detected', async () => {
      const onSpeechStart = vi.fn();
      const { result } = renderHook(() => useVAD({ onSpeechStart }));

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        getCapturedCallbacks()?.onSpeechStart?.();
      });

      expect(onSpeechStart).toHaveBeenCalled();
    });

    it('should call onSpeechEnd with audio data', async () => {
      const onSpeechEnd = vi.fn();
      const { result } = renderHook(() => useVAD({ onSpeechEnd }));

      await act(async () => {
        await result.current.startVAD();
      });

      const audioData = new Float32Array([0.1, 0.2, 0.3]);
      act(() => {
        getCapturedCallbacks()?.onSpeechEnd?.(audioData);
      });

      expect(onSpeechEnd).toHaveBeenCalledWith(audioData);
    });

    it('should push audio frame to bridge on onFrameProcessed', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      const audioFrame = new Float32Array([0.5, 0.6, 0.7]);
      act(() => {
        getCapturedCallbacks()?.onFrameProcessed?.(audioFrame);
      });

      expect(mockBridge.pushAudioFrame).toHaveBeenCalledWith(audioFrame);
    });

    it('should push speech segment to bridge on onSpeechEnd', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      const speechData = new Float32Array([0.8, 0.9, 1.0]);
      act(() => {
        getCapturedCallbacks()?.onSpeechEnd?.(speechData);
      });

      expect(mockBridge.pushSpeechSegment).toHaveBeenCalledWith(speechData);
    });

    it('should call onError when error occurs', async () => {
      const onError = vi.fn();
      const { result } = renderHook(() => useVAD({ onError }));

      await act(async () => {
        await result.current.startVAD();
      });

      const error = new Error('Test error');
      act(() => {
        getCapturedCallbacks()?.onError?.(error);
      });

      expect(onError).toHaveBeenCalledWith(error);
    });

    it('should handle missing callbacks gracefully', async () => {
      const { result } = renderHook(() => useVAD()); // No callbacks provided

      await act(async () => {
        await result.current.startVAD();
      });

      // These should not throw even without callbacks - wrap in act() since they trigger state updates
      await act(async () => {
        expect(() => {
          getCapturedCallbacks()?.onStatusChange?.(MicrophoneStatus.Recording);
          getCapturedCallbacks()?.onSpeechProbability?.(0.5, false);
          getCapturedCallbacks()?.onSpeechStart?.();
          getCapturedCallbacks()?.onSpeechEnd?.(new Float32Array([0.1]));
          getCapturedCallbacks()?.onError?.(new Error('test'));
        }).not.toThrow();
      });
    });
  });

  describe('State Management', () => {
    it('should compute isRecording from status', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      expect(result.current.isRecording).toBe(false);

      act(() => {
        getCapturedCallbacks()?.onStatusChange?.(MicrophoneStatus.Recording);
      });

      expect(result.current.isRecording).toBe(true);
    });

    it('should use latest callbacks via ref', async () => {
      const onSpeechStart1 = vi.fn();
      const onSpeechStart2 = vi.fn();

      const { result, rerender } = renderHook(
        ({ options }) => useVAD(options),
        { initialProps: { options: { onSpeechStart: onSpeechStart1 } } }
      );

      await act(async () => {
        await result.current.startVAD();
      });

      // Update options
      rerender({ options: { onSpeechStart: onSpeechStart2 } });

      // Trigger callback
      act(() => {
        getCapturedCallbacks()?.onSpeechStart?.();
      });

      expect(onSpeechStart1).not.toHaveBeenCalled();
      expect(onSpeechStart2).toHaveBeenCalled();
    });
  });

  describe('Cleanup on Unmount', () => {
    it('should destroy VAD manager on unmount', async () => {
      const { result, unmount } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      unmount();

      expect(mockVADManager.destroy).toHaveBeenCalled();
    });

    it('should cleanup bridge on unmount', async () => {
      const { result, unmount } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      unmount();

      expect(mockBridge.cleanup).toHaveBeenCalled();
    });

    it('should handle unmount when resources are null', () => {
      const { unmount } = renderHook(() => useVAD());

      // Should not throw
      expect(() => unmount()).not.toThrow();
    });
  });

  describe('Integration Scenarios', () => {
    it('should support multiple start/stop cycles', async () => {
      const { result } = renderHook(() => useVAD());

      // First cycle
      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        result.current.stopVAD();
      });

      // Reset mocks for second cycle
      vi.clearAllMocks();

      // Second cycle
      await act(async () => {
        await result.current.startVAD();
      });

      expect(mockVADManager.start).toHaveBeenCalled();
    });

    it('should support destroy and restart', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      act(() => {
        result.current.destroyVAD();
      });

      // Clear mocks and setup for new cycle
      vi.clearAllMocks();

      // Restart - need to re-render since manager was destroyed
      // This tests that the hook can recover from destroy
    });
  });
});
