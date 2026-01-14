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
  private audioElement: HTMLAudioElement | null = null;
  private currentBlobUrl: string | null = null;
  private audioQueue: AudioRefId[] = [];
  private isProcessingQueue: boolean = false;

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
   * @param data - Audio data as ArrayBuffer or Uint8Array
   * @param metadata - Optional partial AudioRef metadata
   * @returns The AudioRefId for the stored audio data
   */
  async store(data: ArrayBuffer | Uint8Array, metadata?: Partial<AudioRef>): Promise<AudioRefId> {
    await this.ensureInitialized();

    // Convert Uint8Array to ArrayBuffer if needed
    let arrayBuffer: ArrayBuffer;
    if (data instanceof Uint8Array) {
      // Extract ArrayBuffer from Uint8Array view (creates a copy)
      // Note: slice() always returns ArrayBuffer even if source is SharedArrayBuffer
      arrayBuffer = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) as ArrayBuffer;
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
      format: metadata?.format,
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
   * Cleanup old audio chunks (deletes audio older than 7 days)
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

  /**
   * Queue audio for playback. Audio will be played in order.
   * @param id - The AudioRefId to queue for playback
   */
  queuePlayback(id: AudioRefId): void {
    this.audioQueue.push(id);
    this.processQueue();
  }

  /**
   * Process the audio queue, playing the next audio if not already playing
   */
  private async processQueue(): Promise<void> {
    if (this.isProcessingQueue || this.audioQueue.length === 0) {
      return;
    }

    this.isProcessingQueue = true;

    while (this.audioQueue.length > 0) {
      const id = this.audioQueue.shift()!;
      try {
        await this.play(id);
      } catch (error) {
        console.error('Failed to play queued audio:', error);
      }
    }

    this.isProcessingQueue = false;
  }

  /**
   * Play audio by ID
   * @param id - The AudioRefId to play
   * @returns Promise that resolves when audio finishes playing
   */
  async play(id: AudioRefId): Promise<void> {
    await this.ensureInitialized();

    const audioStore = useAudioStore.getState();

    try {
      // Retrieve audio data
      const audioData = await this.retrieve(id);
      if (!audioData) {
        throw new Error('Audio data not found');
      }

      // Clean up previous blob URL
      if (this.currentBlobUrl) {
        URL.revokeObjectURL(this.currentBlobUrl);
        this.currentBlobUrl = null;
      }

      // Stop any currently playing audio
      if (this.audioElement) {
        this.audioElement.pause();
        this.audioElement.src = '';
      }

      // Create audio element if needed
      if (!this.audioElement) {
        this.audioElement = new Audio();
      }

      const audio = this.audioElement;

      // Get format from AudioRef metadata
      const audioRef = audioStore.getAudioRef(id);
      const mimeType = this.getMimeType(audioRef?.format);

      // Create blob and object URL
      const blob = new Blob([audioData], { type: mimeType });
      this.currentBlobUrl = URL.createObjectURL(blob);

      // Update store to indicate playback starting
      audioStore.startPlayback(id);

      // Return a promise that resolves when audio ends
      return new Promise((resolve, reject) => {
        audio.onended = () => {
          audioStore.stopPlayback();
          if (this.currentBlobUrl) {
            URL.revokeObjectURL(this.currentBlobUrl);
            this.currentBlobUrl = null;
          }
          resolve();
        };

        audio.onerror = (error) => {
          console.error('Audio playback error:', error);
          audioStore.stopPlayback();
          if (this.currentBlobUrl) {
            URL.revokeObjectURL(this.currentBlobUrl);
            this.currentBlobUrl = null;
          }
          reject(error);
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
        audio.src = this.currentBlobUrl!;
        audio.play().catch(reject);
      });
    } catch (error) {
      console.error('Failed to play audio:', error);
      audioStore.stopPlayback();
      throw error;
    }
  }

  /**
   * Get MIME type from format string
   */
  private getMimeType(format?: string): string {
    if (!format) {
      return 'audio/mpeg'; // Default fallback
    }

    const formatLower = format.toLowerCase();

    // Handle explicit MIME types
    if (formatLower.startsWith('audio/')) {
      return formatLower;
    }

    // Map common format strings to MIME types
    switch (formatLower) {
      case 'opus':
        return 'audio/ogg; codecs=opus';
      case 'pcm':
      case 'pcm16':
      case 'pcm_s16le':
        return 'audio/wav';
      case 'mp3':
      case 'mpeg':
        return 'audio/mpeg';
      case 'wav':
        return 'audio/wav';
      case 'aac':
        return 'audio/aac';
      case 'flac':
        return 'audio/flac';
      case 'ogg':
        return 'audio/ogg';
      default:
        // Try to extract format from compound strings like 'pcm_s16le_24000'
        if (formatLower.includes('opus')) {
          return 'audio/ogg; codecs=opus';
        }
        if (formatLower.includes('pcm')) {
          return 'audio/wav';
        }
        return 'audio/mpeg';
    }
  }

  /**
   * Stop current audio playback
   */
  stop(): void {
    if (this.audioElement) {
      this.audioElement.pause();
      this.audioElement.currentTime = 0;
    }
    if (this.currentBlobUrl) {
      URL.revokeObjectURL(this.currentBlobUrl);
      this.currentBlobUrl = null;
    }
    // Clear queue
    this.audioQueue = [];
    this.isProcessingQueue = false;
    useAudioStore.getState().stopPlayback();
  }
}

// Export singleton instance
export const audioManager = new AudioManagerClass();
