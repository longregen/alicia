import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { useAudioManager } from './useAudioManager';
import { useAudioStore } from '../stores/audioStore';
import { AudioRefId, createAudioRefId } from '../types/streaming';

// Mock Audio element
class MockAudioElement {
  src = '';
  volume = 1;
  currentTime = 0;
  duration = 10;
  paused = true;

  onended: (() => void) | null = null;
  onerror: ((error: Event) => void) | null = null;
  ontimeupdate: (() => void) | null = null;

  play = vi.fn().mockImplementation(() => {
    this.paused = false;
    return Promise.resolve();
  });

  pause = vi.fn().mockImplementation(() => {
    this.paused = true;
  });

  // Test helpers
  _triggerEnded() {
    this.onended?.();
  }

  _triggerError(error: Event) {
    this.onerror?.(error);
  }

  _triggerTimeUpdate() {
    this.ontimeupdate?.();
  }
}

// Store mock Audio instance for test access
let mockAudioInstance: MockAudioElement | null = null;

describe('useAudioManager', () => {
  beforeEach(() => {
    // Reset audio store
    useAudioStore.getState().clearAudioStore();

    // Note: indexedDB.deleteDatabase is called but we don't await it
    // because fake-indexeddb handles it synchronously
    indexedDB.deleteDatabase('alicia-audio');

    // Mock Audio constructor
    mockAudioInstance = null;
    vi.stubGlobal(
      'Audio',
      class Audio {
        constructor() {
          mockAudioInstance = new MockAudioElement();
          return mockAudioInstance;
        }
      }
    );

    // Mock URL APIs
    vi.stubGlobal('URL', {
      ...URL,
      createObjectURL: vi.fn().mockReturnValue('blob:mock-url'),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    mockAudioInstance = null;
  });

  // Helper to wait for DB initialization
  const waitForDbInit = async () => {
    // The IndexedDB initialization happens in useEffect
    // We need to wait for the effect to run and complete
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 100));
    });
  };

  describe('Database Initialization', () => {
    it('should initialize IndexedDB on mount', async () => {
      const { result } = renderHook(() => useAudioManager());

      await waitForDbInit();

      // Should be able to store data after init
      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(10));
      });
      expect(id!).toBeDefined();
    });

    it('should close database on unmount', async () => {
      const { result, unmount } = renderHook(() => useAudioManager());

      await waitForDbInit();
      await act(async () => {
        await result.current.store(new ArrayBuffer(10));
      });

      unmount();

      // Database should be closed (can't easily verify, but no errors)
    });
  });

  describe('store', () => {
    it('should store ArrayBuffer and return AudioRefId', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const data = new ArrayBuffer(100);
      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(data);
      });

      expect(id!).toBeDefined();
      expect(typeof id!).toBe('string');
      expect(id!).toMatch(/^audio-\d+-[a-z0-9]+$/);
    });

    it('should store Uint8Array (converted to ArrayBuffer)', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const uint8Data = new Uint8Array([1, 2, 3, 4, 5]);
      let id: AudioRefId;
      let retrieved: ArrayBuffer | null;
      await act(async () => {
        id = await result.current.store(uint8Data);
      });

      expect(id!).toBeDefined();

      // Verify data can be retrieved
      await act(async () => {
        retrieved = await result.current.retrieve(id);
      });
      expect(retrieved!).not.toBeNull();
      expect(new Uint8Array(retrieved!)).toEqual(uint8Data);
    });

    it('should create AudioRef metadata in Zustand store', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const data = new ArrayBuffer(1024);
      let id!: AudioRefId;
      await act(async () => {
        id = await result.current.store(data, {
          durationMs: 5000,
          sampleRate: 44100,
        });
      });

      const metadata = result.current.getMetadata(id);
      expect(metadata).toEqual({
        id,
        sizeBytes: 1024,
        durationMs: 5000,
        sampleRate: 44100,
      });
    });

    it('should use default metadata values when not provided', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const data = new ArrayBuffer(512);
      let id!: AudioRefId;
      await act(async () => {
        id = await result.current.store(data);
      });

      const metadata = result.current.getMetadata(id);
      expect(metadata?.durationMs).toBe(0);
      expect(metadata?.sampleRate).toBe(16000);
    });

    it('should throw when database not initialized', async () => {
      const { result } = renderHook(() => useAudioManager());

      // Try immediately before DB initializes
      await expect(result.current.store(new ArrayBuffer(10))).rejects.toThrow(
        'Audio database not initialized'
      );
    });
  });

  describe('retrieve', () => {
    it('should retrieve stored audio data', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const originalData = new Uint8Array([10, 20, 30, 40, 50]);
      let id: AudioRefId;
      let retrieved: ArrayBuffer | null;
      await act(async () => {
        id = await result.current.store(originalData.buffer);
      });

      await act(async () => {
        retrieved = await result.current.retrieve(id);
      });

      expect(retrieved!).not.toBeNull();
      expect(new Uint8Array(retrieved!)).toEqual(originalData);
    });

    it('should return null for non-existent audio', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const fakeId = createAudioRefId('non-existent-id');
      let retrieved: ArrayBuffer | null;
      await act(async () => {
        retrieved = await result.current.retrieve(fakeId);
      });

      expect(retrieved!).toBeNull();
    });

    it('should throw when database not initialized', async () => {
      const { result } = renderHook(() => useAudioManager());

      await expect(result.current.retrieve(createAudioRefId('any'))).rejects.toThrow(
        'Audio database not initialized'
      );
    });
  });

  describe('delete', () => {
    it('should delete stored audio data', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      let retrieved: ArrayBuffer | null;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });

      // Verify stored
      await act(async () => {
        retrieved = await result.current.retrieve(id);
      });
      expect(retrieved!).not.toBeNull();

      // Delete
      await act(async () => {
        await result.current.delete(id);
      });

      // Verify deleted
      await act(async () => {
        retrieved = await result.current.retrieve(id);
      });
      expect(retrieved!).toBeNull();
    });

    it('should not throw when deleting non-existent audio', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const fakeId = createAudioRefId('non-existent');

      await act(async () => {
        await expect(result.current.delete(fakeId)).resolves.not.toThrow();
      });
    });

    it('should throw when database not initialized', async () => {
      const { result } = renderHook(() => useAudioManager());

      await expect(result.current.delete(createAudioRefId('any'))).rejects.toThrow(
        'Audio database not initialized'
      );
    });
  });

  describe('play', () => {
    it('should play stored audio', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      expect(mockAudioInstance?.play).toHaveBeenCalled();
      expect(URL.createObjectURL).toHaveBeenCalled();
    });

    it('should update audio store playback state on play', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      const storeState = useAudioStore.getState();
      expect(storeState.playback.isPlaying).toBe(true);
      expect(storeState.playback.currentlyPlayingId).toBe(id!);
    });

    it('should apply volume from store', async () => {
      useAudioStore.getState().setVolume(0.5);

      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      expect(mockAudioInstance?.volume).toBe(0.5);
    });

    it('should set volume to 0 when muted', async () => {
      useAudioStore.getState().toggleMute();

      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      expect(mockAudioInstance?.volume).toBe(0);
    });

    it('should stop previous audio before playing new', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id1: AudioRefId;
      let id2: AudioRefId;
      await act(async () => {
        id1 = await result.current.store(new ArrayBuffer(100));
        id2 = await result.current.store(new ArrayBuffer(100));
      });

      await act(async () => {
        await result.current.play(id1!);
      });
      const firstInstance = mockAudioInstance;

      await act(async () => {
        await result.current.play(id2!);
      });

      expect(firstInstance?.pause).toHaveBeenCalled();
    });

    it('should throw when audio data not found', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const fakeId = createAudioRefId('non-existent');

      await act(async () => {
        await expect(result.current.play(fakeId)).rejects.toThrow('Audio data not found');
      });

      consoleErrorSpy.mockRestore();
    });

    it('should handle playback errors', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id!: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });

      // First play creates an Audio instance
      await act(async () => {
        await result.current.play(id);
      });

      // Now modify the mock to reject on next play
      mockAudioInstance!.play = vi.fn().mockRejectedValue(new Error('Playback failed'));

      // Try playing again - should fail and update store
      let playError: Error | undefined;
      await act(async () => {
        try {
          await result.current.play(id);
        } catch (err) {
          playError = err as Error;
        }
      });

      expect(playError).toBeDefined();
      expect(playError!.message).toBe('Playback failed');
      // Store should be updated to stop playback
      expect(useAudioStore.getState().playback.isPlaying).toBe(false);

      consoleErrorSpy.mockRestore();
    });

    it('should update progress on timeupdate event', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      // Simulate progress
      mockAudioInstance!.currentTime = 5;
      mockAudioInstance!.duration = 10;
      act(() => {
        mockAudioInstance!._triggerTimeUpdate();
      });

      expect(useAudioStore.getState().playback.playbackProgress).toBe(0.5);
    });

    it('should cleanup on audio ended', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      act(() => {
        mockAudioInstance!._triggerEnded();
      });

      expect(useAudioStore.getState().playback.isPlaying).toBe(false);
      expect(URL.revokeObjectURL).toHaveBeenCalled();
    });

    it('should cleanup on audio error', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      act(() => {
        mockAudioInstance!._triggerError(new Event('error'));
      });

      expect(useAudioStore.getState().playback.isPlaying).toBe(false);
      expect(URL.revokeObjectURL).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should throw when database not initialized', async () => {
      const { result } = renderHook(() => useAudioManager());

      await expect(result.current.play(createAudioRefId('any'))).rejects.toThrow(
        'Audio database not initialized'
      );
    });
  });

  describe('stop', () => {
    it('should pause audio and reset currentTime', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      act(() => {
        result.current.stop();
      });

      expect(mockAudioInstance?.pause).toHaveBeenCalled();
      expect(mockAudioInstance?.currentTime).toBe(0);
    });

    it('should update store playback state', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      act(() => {
        result.current.stop();
      });

      const state = useAudioStore.getState();
      expect(state.playback.isPlaying).toBe(false);
      expect(state.playback.currentlyPlayingId).toBeNull();
    });

    it('should handle stop when nothing is playing', () => {
      const { result } = renderHook(() => useAudioManager());

      expect(() => result.current.stop()).not.toThrow();
    });
  });

  describe('getMetadata', () => {
    it('should return metadata from Zustand store', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(256), {
          durationMs: 3000,
          sampleRate: 22050,
        });
      });

      const metadata = result.current.getMetadata(id!);

      expect(metadata).toEqual({
        id: id!,
        sizeBytes: 256,
        durationMs: 3000,
        sampleRate: 22050,
      });
    });

    it('should return null for non-existent audio', () => {
      const { result } = renderHook(() => useAudioManager());

      const metadata = result.current.getMetadata(createAudioRefId('fake'));
      expect(metadata).toBeNull();
    });
  });

  describe('cleanup', () => {
    it('should not throw when database not initialized', async () => {
      const { result } = renderHook(() => useAudioManager());

      // Should not throw, just return early
      await act(async () => {
        await expect(result.current.cleanup()).resolves.not.toThrow();
      });
    });

    it('should complete without error when database is empty', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      await act(async () => {
        await expect(result.current.cleanup()).resolves.not.toThrow();
      });
    });
  });

  describe('Lifecycle', () => {
    it('should cleanup audio element on unmount', async () => {
      const { result, unmount } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(100));
      });
      await act(async () => {
        await result.current.play(id!);
      });

      unmount();

      expect(mockAudioInstance?.pause).toHaveBeenCalled();
    });

    it('should handle rapid mount/unmount cycles', async () => {
      for (let i = 0; i < 3; i++) {
        const { unmount } = renderHook(() => useAudioManager());
        unmount();
      }

      // Final mount should work correctly
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let id: AudioRefId;
      await act(async () => {
        id = await result.current.store(new ArrayBuffer(50));
      });
      expect(id!).toBeDefined();
    });
  });

  describe('Multiple Operations', () => {
    it('should handle storing multiple audio files', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      let ids: AudioRefId[];
      await act(async () => {
        ids = await Promise.all([
          result.current.store(new ArrayBuffer(100)),
          result.current.store(new ArrayBuffer(200)),
          result.current.store(new ArrayBuffer(300)),
        ]);
      });

      expect(ids!).toHaveLength(3);
      expect(new Set(ids!).size).toBe(3); // All unique IDs
    });

    it('should retrieve correct data for each stored audio', async () => {
      const { result } = renderHook(() => useAudioManager());
      await waitForDbInit();

      const data1 = new Uint8Array([1, 2, 3]);
      const data2 = new Uint8Array([4, 5, 6, 7]);

      let id1: AudioRefId;
      let id2: AudioRefId;
      await act(async () => {
        id1 = await result.current.store(data1);
        id2 = await result.current.store(data2);
      });

      let retrieved1: ArrayBuffer | null;
      let retrieved2: ArrayBuffer | null;
      await act(async () => {
        retrieved1 = await result.current.retrieve(id1!);
        retrieved2 = await result.current.retrieve(id2!);
      });

      expect(new Uint8Array(retrieved1!)).toEqual(data1);
      expect(new Uint8Array(retrieved2!)).toEqual(data2);
    });
  });
});
