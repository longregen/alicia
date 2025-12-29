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

/**
 * Singleton AudioManager for use outside React components
 * This allows the protocol adapter to store audio without React hooks
 */
class AudioManagerClass {
  private db: IDBPDatabase<AudioDB> | null = null;
  private initPromise: Promise<void> | null = null;

  constructor() {
    this.initPromise = this.initialize();
  }

  private async initialize(): Promise<void> {
    try {
      this.db = await openDB<AudioDB>(DB_NAME, DB_VERSION, {
        upgrade(db) {
          if (!db.objectStoreNames.contains(STORE_NAME)) {
            db.createObjectStore(STORE_NAME, { keyPath: 'id' });
          }
        },
      });
    } catch (error) {
      console.error('Failed to initialize audio IndexedDB:', error);
      throw error;
    }
  }

  private async ensureInitialized(): Promise<void> {
    if (this.initPromise) {
      await this.initPromise;
    }
    if (!this.db) {
      throw new Error('Audio database not initialized');
    }
  }

  /**
   * Store audio data in IndexedDB and create metadata in the store
   */
  async store(data: ArrayBuffer | Uint8Array, metadata?: Partial<AudioRef>): Promise<AudioRefId> {
    await this.ensureInitialized();

    // Convert Uint8Array to ArrayBuffer if needed
    let arrayBuffer: ArrayBuffer;
    if (data instanceof Uint8Array) {
      // Slice the buffer to get a copy as a regular ArrayBuffer
      const sliced = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength);
      // Ensure it's an ArrayBuffer, not a SharedArrayBuffer
      arrayBuffer = sliced instanceof ArrayBuffer ? sliced : new Uint8Array(data).buffer;
    } else {
      arrayBuffer = data;
    }

    // Generate unique ID
    const id = createAudioRefId(`audio-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`);

    // Store in IndexedDB
    await this.db!.put(STORE_NAME, {
      id: id as string,
      data: arrayBuffer,
      timestamp: Date.now(),
    });

    // Create AudioRef metadata
    const audioRef: AudioRef = {
      id,
      sizeBytes: arrayBuffer.byteLength,
      durationMs: metadata?.durationMs ?? 0,
      sampleRate: metadata?.sampleRate ?? 24000,
    };

    // Store metadata in Zustand store
    useAudioStore.getState().addAudioRef(audioRef);

    return id;
  }

  /**
   * Retrieve audio data from IndexedDB
   */
  async retrieve(id: AudioRefId): Promise<ArrayBuffer | null> {
    await this.ensureInitialized();

    try {
      const record = await this.db!.get(STORE_NAME, id as string);
      return record?.data ?? null;
    } catch (error) {
      console.error('Failed to retrieve audio:', error);
      return null;
    }
  }

  /**
   * Delete audio data from IndexedDB
   */
  async delete(id: AudioRefId): Promise<void> {
    await this.ensureInitialized();

    try {
      await this.db!.delete(STORE_NAME, id as string);
    } catch (error) {
      console.error('Failed to delete audio:', error);
    }
  }

  /**
   * Cleanup old audio chunks (optional utility)
   */
  async cleanup(): Promise<void> {
    await this.ensureInitialized();

    try {
      const tx = this.db!.transaction(STORE_NAME, 'readwrite');
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
  }
}

// Export singleton instance
export const audioManager = new AudioManagerClass();
