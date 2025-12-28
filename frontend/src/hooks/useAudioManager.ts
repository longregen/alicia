import { useEffect, useRef, useCallback } from 'react';
import { openDB, DBSchema, IDBPDatabase } from 'idb';
import { AudioRefId, AudioRef, createAudioRefId } from '../types/streaming';
import { useAudioStore } from '../stores/audioStore';

// IndexedDB schema for audio storage
interface AudioDB extends DBSchema {
  'audio-chunks': {
    key: string;
    value: {
      id: string;
      data: ArrayBuffer;
      timestamp: number;
    };
  };
}

const DB_NAME = 'alicia-audio';
const DB_VERSION = 1;
const STORE_NAME = 'audio-chunks';

export interface AudioManager {
  store(data: ArrayBuffer | Uint8Array, metadata?: Partial<AudioRef>): Promise<AudioRefId>;
  retrieve(id: AudioRefId): Promise<ArrayBuffer | null>;
  delete(id: AudioRefId): Promise<void>;
  play(id: AudioRefId): Promise<void>;
  stop(): void;
  getMetadata(id: AudioRefId): AudioRef | null;
  cleanup(): Promise<void>;
}

export function useAudioManager(): AudioManager {
  const dbRef = useRef<IDBPDatabase<AudioDB> | null>(null);
  const audioElementRef = useRef<HTMLAudioElement | null>(null);
  const audioStore = useAudioStore();

  // Initialize IndexedDB on mount
  useEffect(() => {
    let mounted = true;

    async function initDB() {
      try {
        const db = await openDB<AudioDB>(DB_NAME, DB_VERSION, {
          upgrade(db) {
            if (!db.objectStoreNames.contains(STORE_NAME)) {
              db.createObjectStore(STORE_NAME, { keyPath: 'id' });
            }
          },
        });

        if (mounted) {
          dbRef.current = db;
        } else {
          // Component unmounted while DB was initializing, close it immediately
          db.close();
        }
      } catch (error) {
        console.error('Failed to initialize audio IndexedDB:', error);
      }
    }

    initDB();

    return () => {
      mounted = false;
      if (dbRef.current) {
        dbRef.current.close();
        dbRef.current = null;
      }
      if (audioElementRef.current) {
        audioElementRef.current.pause();
        audioElementRef.current.src = '';
        audioElementRef.current = null;
      }
    };
  }, []);

  // Store audio data in IndexedDB and create metadata
  const store = useCallback(
    async (data: ArrayBuffer | Uint8Array, metadata?: Partial<AudioRef>): Promise<AudioRefId> => {
      if (!dbRef.current) {
        throw new Error('Audio database not initialized');
      }

      // Convert Uint8Array to ArrayBuffer if needed
      // Create a copy to ensure we have a regular ArrayBuffer (not SharedArrayBuffer)
      let arrayBuffer: ArrayBuffer;
      if (data instanceof Uint8Array) {
        // Slice the buffer to get a copy as a regular ArrayBuffer
        arrayBuffer = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) as ArrayBuffer;
      } else {
        arrayBuffer = data;
      }

      // Generate unique ID
      const id = createAudioRefId(`audio-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`);

      // Store in IndexedDB
      await dbRef.current.put(STORE_NAME, {
        id: id as string,
        data: arrayBuffer,
        timestamp: Date.now(),
      });

      // Create AudioRef metadata
      const audioRef: AudioRef = {
        id,
        sizeBytes: arrayBuffer.byteLength,
        durationMs: metadata?.durationMs ?? 0,
        sampleRate: metadata?.sampleRate ?? 16000,
      };

      // Store metadata in Zustand store
      audioStore.addAudioRef(audioRef);

      return id;
    },
    [audioStore]
  );

  // Retrieve audio data from IndexedDB
  const retrieve = useCallback(async (id: AudioRefId): Promise<ArrayBuffer | null> => {
    if (!dbRef.current) {
      throw new Error('Audio database not initialized');
    }

    try {
      const record = await dbRef.current.get(STORE_NAME, id as string);
      return record?.data ?? null;
    } catch (error) {
      console.error('Failed to retrieve audio:', error);
      return null;
    }
  }, []);

  // Delete audio data from IndexedDB
  const deleteAudio = useCallback(async (id: AudioRefId): Promise<void> => {
    if (!dbRef.current) {
      throw new Error('Audio database not initialized');
    }

    try {
      await dbRef.current.delete(STORE_NAME, id as string);
    } catch (error) {
      console.error('Failed to delete audio:', error);
    }
  }, []);

  // Play audio by ID
  const play = useCallback(
    async (id: AudioRefId): Promise<void> => {
      if (!dbRef.current) {
        throw new Error('Audio database not initialized');
      }

      try {
        // Update store to indicate playback starting
        audioStore.startPlayback(id);

        // Retrieve audio data
        const audioData = await retrieve(id);
        if (!audioData) {
          throw new Error('Audio data not found');
        }

        // Stop any currently playing audio
        if (audioElementRef.current) {
          audioElementRef.current.pause();
          audioElementRef.current.src = '';
        }

        // Create audio element if needed
        if (!audioElementRef.current) {
          audioElementRef.current = new Audio();
        }

        const audio = audioElementRef.current;

        // Create blob and object URL
        const blob = new Blob([audioData], { type: 'audio/mpeg' });
        const url = URL.createObjectURL(blob);

        // Set up event handlers
        audio.onended = () => {
          audioStore.stopPlayback();
          URL.revokeObjectURL(url);
        };

        audio.onerror = (error) => {
          console.error('Audio playback error:', error);
          audioStore.stopPlayback();
          URL.revokeObjectURL(url);
        };

        audio.ontimeupdate = () => {
          if (audio.duration > 0) {
            const progress = audio.currentTime / audio.duration;
            audioStore.updatePlaybackProgress(progress);
          }
        };

        // Apply volume settings
        audio.volume = audioStore.playback.isMuted ? 0 : audioStore.playback.volume;

        // Play audio
        audio.src = url;
        await audio.play();
      } catch (error) {
        console.error('Failed to play audio:', error);
        audioStore.stopPlayback();
        throw error;
      }
    },
    [audioStore, retrieve]
  );

  // Stop current playback
  const stop = useCallback(() => {
    if (audioElementRef.current) {
      audioElementRef.current.pause();
      audioElementRef.current.currentTime = 0;
    }
    audioStore.stopPlayback();
  }, [audioStore]);

  // Get metadata from store
  const getMetadata = useCallback(
    (id: AudioRefId): AudioRef | null => {
      return audioStore.getAudioRef(id) ?? null;
    },
    [audioStore]
  );

  // Cleanup old audio chunks (optional utility)
  const cleanup = useCallback(async (): Promise<void> => {
    if (!dbRef.current) {
      return;
    }

    try {
      const tx = dbRef.current.transaction(STORE_NAME, 'readwrite');
      const store = tx.objectStore(STORE_NAME);
      const allRecords = await store.getAll();

      // Delete audio older than 7 days
      const cutoffTime = Date.now() - 7 * 24 * 60 * 60 * 1000;
      const deletePromises = allRecords
        .filter((record) => record.timestamp < cutoffTime)
        .map((record) => store.delete(record.id));

      await Promise.all(deletePromises);
      await tx.done;
    } catch (error) {
      console.error('Failed to cleanup audio storage:', error);
    }
  }, []);

  return {
    store,
    retrieve,
    delete: deleteAudio,
    play,
    stop,
    getMetadata,
    cleanup,
  };
}
