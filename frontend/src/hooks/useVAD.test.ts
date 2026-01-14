import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useVAD } from './useVAD';
import { MicrophoneStatus } from '../types/streaming';
import type { MicrophoneManagerCallbacks } from '../utils/microphoneManager';

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

// Store captured callbacks for tests
const testState = {
  subscribers: new Set<MicrophoneManagerCallbacks>(),
  status: MicrophoneStatus.Inactive,
  audioTrack: null as MediaStreamTrack | null,
  speechProbability: 0,
  isSpeaking: false,
  mockManager: {
    start: vi.fn(),
    stop: vi.fn(),
    destroy: vi.fn(),
    getStatus: vi.fn(() => testState.status),
    getAudioTrack: vi.fn(() => testState.audioTrack),
    getSpeechProbability: vi.fn(() => testState.speechProbability),
    getIsSpeaking: vi.fn(() => testState.isSpeaking),
    isRecording: vi.fn(() => false),
    subscribe: vi.fn((callbacks: MicrophoneManagerCallbacks) => {
      testState.subscribers.add(callbacks);
      // Immediately notify of current state
      if (testState.status !== MicrophoneStatus.Inactive) {
        callbacks.onStatusChange?.(testState.status);
      }
      if (testState.audioTrack) {
        callbacks.onTrackReady?.(testState.audioTrack);
      }
      return () => {
        testState.subscribers.delete(callbacks);
      };
    }),
  },
};

// Helper to notify all subscribers
const notifySubscribers = <K extends keyof MicrophoneManagerCallbacks>(
  event: K,
  ...args: Parameters<NonNullable<MicrophoneManagerCallbacks[K]>>
): void => {
  testState.subscribers.forEach((callbacks) => {
    const callback = callbacks[event];
    if (callback) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (callback as (...args: any[]) => void)(...args);
    }
  });
};

// Mock the MicrophoneManager
vi.mock('../utils/microphoneManager', () => {
  return {
    getMicrophoneManager: () => testState.mockManager,
  };
});

describe('useVAD', () => {
  const mockManager = testState.mockManager;

  beforeEach(() => {
    // Reset test state
    testState.subscribers.clear();
    testState.status = MicrophoneStatus.Inactive;
    testState.audioTrack = null;
    testState.speechProbability = 0;
    testState.isSpeaking = false;

    // Reset all mock implementations
    vi.clearAllMocks();

    // Suppress console output during tests
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});

    // Reset mock implementations to default behavior
    mockManager.start.mockResolvedValue(mockMediaStreamTrack);
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

    it('should subscribe to MicrophoneManager on mount', () => {
      renderHook(() => useVAD());

      expect(mockManager.subscribe).toHaveBeenCalled();
      expect(testState.subscribers.size).toBe(1);
    });

    it('should unsubscribe from MicrophoneManager on unmount', () => {
      const { unmount } = renderHook(() => useVAD());

      expect(testState.subscribers.size).toBe(1);

      unmount();

      expect(testState.subscribers.size).toBe(0);
    });

    it('should initialize state from manager current state', () => {
      // Set up manager state before mounting
      testState.status = MicrophoneStatus.Recording;
      testState.audioTrack = mockMediaStreamTrack;
      testState.speechProbability = 0.75;
      testState.isSpeaking = true;

      const { result } = renderHook(() => useVAD());

      expect(result.current.status).toBe(MicrophoneStatus.Recording);
      expect(result.current.audioTrack).toBe(mockMediaStreamTrack);
      expect(result.current.speechProbability).toBe(0.75);
      expect(result.current.isSpeaking).toBe(true);
    });
  });

  describe('startVAD', () => {
    it('should call manager.start()', async () => {
      const { result } = renderHook(() => useVAD());

      await act(async () => {
        await result.current.startVAD();
      });

      expect(mockManager.start).toHaveBeenCalled();
    });

    it('should handle start errors gracefully', async () => {
      mockManager.start.mockRejectedValueOnce(new Error('Start failed'));

      const { result } = renderHook(() => useVAD());

      await expect(
        act(async () => {
          await result.current.startVAD();
        })
      ).rejects.toThrow('Start failed');
    });
  });

  describe('stopVAD', () => {
    it('should call manager.stop()', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        result.current.stopVAD();
      });

      expect(mockManager.stop).toHaveBeenCalled();
    });

    it('should be safe to call when not recording', () => {
      const { result } = renderHook(() => useVAD());

      // Should not throw
      expect(() => result.current.stopVAD()).not.toThrow();
      expect(mockManager.stop).toHaveBeenCalled();
    });
  });

  describe('Callback Invocations', () => {
    it('should call onStatusChange when manager status changes', () => {
      const onStatusChange = vi.fn();
      renderHook(() => useVAD({ onStatusChange }));

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
      });

      expect(onStatusChange).toHaveBeenCalledWith(MicrophoneStatus.Recording);
    });

    it('should update status state on status change', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
      });

      expect(result.current.status).toBe(MicrophoneStatus.Recording);
    });

    it('should call onSpeechProbability with probability and speaking state', () => {
      const onSpeechProbability = vi.fn();
      renderHook(() => useVAD({ onSpeechProbability }));

      act(() => {
        notifySubscribers('onSpeechProbability', 0.75, true);
      });

      expect(onSpeechProbability).toHaveBeenCalledWith(0.75, true);
    });

    it('should update speechProbability state', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        notifySubscribers('onSpeechProbability', 0.9, true);
      });

      expect(result.current.speechProbability).toBe(0.9);
    });

    it('should update isSpeaking state', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        notifySubscribers('onSpeechProbability', 0.9, true);
      });

      expect(result.current.isSpeaking).toBe(true);
    });

    it('should call onSpeechStart when speech detected', () => {
      const onSpeechStart = vi.fn();
      renderHook(() => useVAD({ onSpeechStart }));

      act(() => {
        notifySubscribers('onSpeechStart');
      });

      expect(onSpeechStart).toHaveBeenCalled();
    });

    it('should call onSpeechEnd with audio data', () => {
      const onSpeechEnd = vi.fn();
      renderHook(() => useVAD({ onSpeechEnd }));

      const audioData = new Float32Array([0.1, 0.2, 0.3]);
      act(() => {
        notifySubscribers('onSpeechEnd', audioData);
      });

      expect(onSpeechEnd).toHaveBeenCalledWith(audioData);
    });

    it('should call onTrackReady when track is ready', () => {
      const onTrackReady = vi.fn();
      renderHook(() => useVAD({ onTrackReady }));

      act(() => {
        notifySubscribers('onTrackReady', mockMediaStreamTrack);
      });

      expect(onTrackReady).toHaveBeenCalledWith(mockMediaStreamTrack);
    });

    it('should update audioTrack state when track is ready', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        notifySubscribers('onTrackReady', mockMediaStreamTrack);
      });

      expect(result.current.audioTrack).toBe(mockMediaStreamTrack);
    });

    it('should call onError when error occurs', () => {
      const onError = vi.fn();
      renderHook(() => useVAD({ onError }));

      const error = new Error('Test error');
      act(() => {
        notifySubscribers('onError', error);
      });

      expect(onError).toHaveBeenCalledWith(error);
    });

    it('should handle missing callbacks gracefully', () => {
      renderHook(() => useVAD()); // No callbacks provided

      // These should not throw even without callbacks
      act(() => {
        expect(() => {
          notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
          notifySubscribers('onSpeechProbability', 0.5, false);
          notifySubscribers('onSpeechStart');
          notifySubscribers('onSpeechEnd', new Float32Array([0.1]));
          notifySubscribers('onError', new Error('test'));
        }).not.toThrow();
      });
    });
  });

  describe('State Management', () => {
    it('should compute isRecording from status (Recording)', () => {
      const { result } = renderHook(() => useVAD());

      expect(result.current.isRecording).toBe(false);

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
      });

      expect(result.current.isRecording).toBe(true);
    });

    it('should compute isRecording from status (Sending)', () => {
      const { result } = renderHook(() => useVAD());

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Sending);
      });

      expect(result.current.isRecording).toBe(true);
    });

    it('should use latest callbacks via ref', () => {
      const onSpeechStart1 = vi.fn();
      const onSpeechStart2 = vi.fn();

      const { rerender } = renderHook(
        ({ options }) => useVAD(options),
        { initialProps: { options: { onSpeechStart: onSpeechStart1 } } }
      );

      // Update options
      rerender({ options: { onSpeechStart: onSpeechStart2 } });

      // Trigger callback
      act(() => {
        notifySubscribers('onSpeechStart');
      });

      expect(onSpeechStart1).not.toHaveBeenCalled();
      expect(onSpeechStart2).toHaveBeenCalled();
    });
  });

  describe('Multiple Hooks', () => {
    it('should allow multiple hooks to subscribe', () => {
      renderHook(() => useVAD());
      renderHook(() => useVAD());

      expect(testState.subscribers.size).toBe(2);
    });

    it('should notify all hooks when status changes', () => {
      const onStatusChange1 = vi.fn();
      const onStatusChange2 = vi.fn();

      renderHook(() => useVAD({ onStatusChange: onStatusChange1 }));
      renderHook(() => useVAD({ onStatusChange: onStatusChange2 }));

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
      });

      expect(onStatusChange1).toHaveBeenCalledWith(MicrophoneStatus.Recording);
      expect(onStatusChange2).toHaveBeenCalledWith(MicrophoneStatus.Recording);
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

      expect(mockManager.start).toHaveBeenCalled();
    });

    it('should handle rapid status changes', () => {
      const onStatusChange = vi.fn();
      renderHook(() => useVAD({ onStatusChange }));

      act(() => {
        notifySubscribers('onStatusChange', MicrophoneStatus.Loading);
        notifySubscribers('onStatusChange', MicrophoneStatus.Active);
        notifySubscribers('onStatusChange', MicrophoneStatus.Recording);
      });

      expect(onStatusChange).toHaveBeenCalledTimes(3);
      expect(onStatusChange).toHaveBeenLastCalledWith(MicrophoneStatus.Recording);
    });
  });
});
