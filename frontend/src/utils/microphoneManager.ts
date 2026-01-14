/**
 * MicrophoneManager - Singleton for managing microphone and VAD resources
 *
 * This class provides a single instance that manages:
 * - SileroVADManager (voice activity detection)
 * - VADLiveKitBridge (audio pipeline to LiveKit)
 * - MediaStreamTrack (reusable across rooms)
 *
 * Benefits:
 * - Microphone permission requested once
 * - VAD model loaded once (~5-10MB)
 * - Single AudioContext (browser limits exist)
 * - Same track can be published to multiple LiveKit rooms
 */

import { MicrophoneStatus } from '../types/streaming';
import { SileroVADManager, SileroVADCallbacks } from './sileroVAD';
import { VADLiveKitBridge } from '../adapters/vadLiveKitBridge';

export interface MicrophoneManagerCallbacks {
  onStatusChange?: (status: MicrophoneStatus) => void;
  onSpeechProbability?: (probability: number, isSpeaking: boolean) => void;
  onSpeechStart?: () => void;
  onSpeechEnd?: (audioData: Float32Array) => void;
  onTrackReady?: (track: MediaStreamTrack) => void;
  onError?: (error: Error) => void;
}

class MicrophoneManagerClass {
  private static instance: MicrophoneManagerClass | null = null;

  private vadManager: SileroVADManager | null = null;
  private bridge: VADLiveKitBridge | null = null;
  private audioTrack: MediaStreamTrack | null = null;
  private status: MicrophoneStatus = MicrophoneStatus.Inactive;
  private subscribers: Set<MicrophoneManagerCallbacks> = new Set();
  private isInitialized = false;
  private isStarted = false;
  private speechProbability = 0;
  private isSpeaking = false;

  private constructor() {
    // Private constructor for singleton
  }

  /**
   * Get the singleton instance
   */
  static getInstance(): MicrophoneManagerClass {
    if (!MicrophoneManagerClass.instance) {
      MicrophoneManagerClass.instance = new MicrophoneManagerClass();
    }
    return MicrophoneManagerClass.instance;
  }

  /**
   * Subscribe to microphone manager events
   * Returns unsubscribe function
   */
  subscribe(callbacks: MicrophoneManagerCallbacks): () => void {
    this.subscribers.add(callbacks);

    // Immediately notify subscriber of current state
    if (this.status !== MicrophoneStatus.Inactive) {
      callbacks.onStatusChange?.(this.status);
    }
    if (this.audioTrack) {
      callbacks.onTrackReady?.(this.audioTrack);
    }

    return () => {
      this.subscribers.delete(callbacks);
    };
  }

  /**
   * Start VAD and return the audio track
   * Idempotent - safe to call multiple times
   */
  async start(): Promise<MediaStreamTrack> {
    // If already started, return existing track
    if (this.isStarted && this.audioTrack) {
      return this.audioTrack;
    }

    // If already initialized but paused, just resume
    if (this.isInitialized && this.vadManager && this.audioTrack) {
      await this.vadManager.start();
      this.isStarted = true;
      this.updateStatus(MicrophoneStatus.Recording);
      return this.audioTrack;
    }

    // First time initialization
    try {
      this.updateStatus(MicrophoneStatus.Loading);

      // Create VAD manager with callbacks
      const vadCallbacks: SileroVADCallbacks = {
        onStatusChange: (status) => {
          // Map SileroVAD status to our status
          // Don't override Loading status while we're still initializing
          if (this.status !== MicrophoneStatus.Loading) {
            this.updateStatus(status);
          }
        },
        onSpeechProbability: (probability, speaking) => {
          this.speechProbability = probability;
          this.isSpeaking = speaking;
          this.notifySubscribers('onSpeechProbability', probability, speaking);
        },
        onFrameProcessed: (audioFrame) => {
          this.bridge?.pushAudioFrame(audioFrame);
        },
        onSpeechStart: () => {
          this.notifySubscribers('onSpeechStart');
        },
        onSpeechEnd: (audioData) => {
          this.bridge?.pushSpeechSegment(audioData);
          this.notifySubscribers('onSpeechEnd', audioData);
          // Briefly show sending state
          this.updateStatus(MicrophoneStatus.Sending);
          // Return to recording after a short delay
          setTimeout(() => {
            if (this.isStarted) {
              this.updateStatus(MicrophoneStatus.Recording);
            }
          }, 500);
        },
        onError: (error) => {
          this.updateStatus(MicrophoneStatus.Error);
          this.notifySubscribers('onError', error);
        },
      };

      this.vadManager = new SileroVADManager(vadCallbacks);

      // Initialize the bridge (creates AudioContext and worklet)
      this.bridge = new VADLiveKitBridge();
      this.audioTrack = await this.bridge.initialize();

      // Notify subscribers that track is ready
      this.notifySubscribers('onTrackReady', this.audioTrack);

      // Initialize and start VAD (connects to microphone)
      await this.vadManager.initialize();
      await this.vadManager.start();

      this.isInitialized = true;
      this.isStarted = true;
      this.updateStatus(MicrophoneStatus.Recording);

      return this.audioTrack;
    } catch (error) {
      this.updateStatus(MicrophoneStatus.Error);
      this.notifySubscribers('onError', error as Error);
      // Cleanup on error
      this.cleanupResources();
      throw error;
    }
  }

  /**
   * Stop recording and release the microphone.
   * This destroys the VAD instance to hide the browser recording indicator.
   * Scripts remain loaded for fast restart (~500ms to reinitialize).
   */
  stop(): void {
    if (!this.isStarted) return;

    if (this.vadManager) {
      this.vadManager.stop();
    }

    if (this.bridge) {
      this.bridge.cleanup();
      this.bridge = null;
    }

    this.audioTrack = null;
    this.isInitialized = false;
    this.isStarted = false;
    this.speechProbability = 0;
    this.isSpeaking = false;
    this.updateStatus(MicrophoneStatus.Inactive);
  }

  /**
   * Fully destroy all resources
   * Call this when the app is closing or user explicitly disables audio
   */
  destroy(): void {
    this.cleanupResources();
    this.updateStatus(MicrophoneStatus.Inactive);
    MicrophoneManagerClass.instance = null;
  }

  /**
   * Get current status
   */
  getStatus(): MicrophoneStatus {
    return this.status;
  }

  /**
   * Get current audio track (may be null if not started)
   */
  getAudioTrack(): MediaStreamTrack | null {
    return this.audioTrack;
  }

  /**
   * Get current speech probability
   */
  getSpeechProbability(): number {
    return this.speechProbability;
  }

  /**
   * Get whether speech is currently detected
   */
  getIsSpeaking(): boolean {
    return this.isSpeaking;
  }

  /**
   * Check if microphone manager is currently recording
   */
  isRecording(): boolean {
    return this.isStarted;
  }

  /**
   * Clean up internal resources
   */
  private cleanupResources(): void {
    if (this.vadManager) {
      this.vadManager.destroy();
      this.vadManager = null;
    }

    if (this.bridge) {
      this.bridge.cleanup();
      this.bridge = null;
    }

    this.audioTrack = null;
    this.isInitialized = false;
    this.isStarted = false;
    this.speechProbability = 0;
    this.isSpeaking = false;
  }

  /**
   * Update status and notify subscribers
   */
  private updateStatus(status: MicrophoneStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.notifySubscribers('onStatusChange', status);
  }

  /**
   * Notify all subscribers of an event
   */
  private notifySubscribers<K extends keyof MicrophoneManagerCallbacks>(
    event: K,
    ...args: Parameters<NonNullable<MicrophoneManagerCallbacks[K]>>
  ): void {
    this.subscribers.forEach((callbacks) => {
      const callback = callbacks[event];
      if (callback) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (callback as (...args: any[]) => void)(...args);
      }
    });
  }
}

// Export singleton getter
export const getMicrophoneManager = (): MicrophoneManagerClass => {
  return MicrophoneManagerClass.getInstance();
};

// Export type for the manager
export type MicrophoneManager = MicrophoneManagerClass;
