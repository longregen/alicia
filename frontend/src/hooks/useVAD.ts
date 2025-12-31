import { useState, useCallback, useRef, useEffect } from 'react';
import { SileroVADManager, SileroVADCallbacks } from '../utils/sileroVAD';
import { MicrophoneStatus } from '../types/streaming';
import { VADLiveKitBridge } from '../adapters/vadLiveKitBridge';

export interface UseVADOptions {
  /** Callback when speech starts */
  onSpeechStart?: () => void;
  /** Callback when speech ends with audio data */
  onSpeechEnd?: (audioData: Float32Array) => void;
  /** Callback when speech probability changes */
  onSpeechProbability?: (probability: number, isSpeaking: boolean) => void;
  /** Callback when VAD status changes */
  onStatusChange?: (status: MicrophoneStatus) => void;
  /** Callback when error occurs */
  onError?: (error: Error) => void;
  /** Callback when audio track is ready for LiveKit publishing */
  onTrackReady?: (track: MediaStreamTrack) => void;
}

export interface UseVADReturn {
  /** Start VAD processing and microphone capture */
  startVAD: () => Promise<void>;
  /** Stop VAD processing */
  stopVAD: () => void;
  /** Destroy VAD instance and cleanup resources */
  destroyVAD: () => void;
  /** Current microphone status */
  status: MicrophoneStatus;
  /** Whether VAD is currently recording */
  isRecording: boolean;
  /** Current speech probability (0-1) */
  speechProbability: number;
  /** Whether speech is currently detected */
  isSpeaking: boolean;
  /** MediaStreamTrack for LiveKit publishing (null until started) */
  audioTrack: MediaStreamTrack | null;
  /** VAD manager instance (for advanced usage) */
  vadManager: SileroVADManager | null;
}

/**
 * Hook for managing Silero VAD integration with LiveKit audio streaming.
 *
 * This hook:
 * - Initializes and manages SileroVADManager
 * - Creates MediaStreamTrack from microphone input
 * - Provides start/stop controls for voice activity detection
 * - Exposes speech detection events and probability
 * - Returns audio track for publishing to LiveKit
 *
 * @example
 * ```tsx
 * const { startVAD, stopVAD, audioTrack, isSpeaking } = useVAD({
 *   onSpeechEnd: (audio) => console.log('Speech ended:', audio.length),
 *   onTrackReady: (track) => liveKit.publishAudioTrack(track),
 * });
 *
 * // Start VAD when user clicks microphone button
 * <button onClick={startVAD}>Start Recording</button>
 * ```
 */
export function useVAD(options: UseVADOptions = {}): UseVADReturn {
  const [status, setStatus] = useState<MicrophoneStatus>(MicrophoneStatus.Inactive);
  const [speechProbability, setSpeechProbability] = useState(0);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [audioTrack, setAudioTrack] = useState<MediaStreamTrack | null>(null);

  const vadManagerRef = useRef<SileroVADManager | null>(null);
  const bridgeRef = useRef<VADLiveKitBridge | null>(null);
  const optionsRef = useRef<UseVADOptions>(options);

  // Keep options ref updated
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  // Initialize VAD manager on mount
  useEffect(() => {
    const callbacks: SileroVADCallbacks = {
      onStatusChange: (newStatus) => {
        setStatus(newStatus);
        optionsRef.current.onStatusChange?.(newStatus);
      },
      onSpeechProbability: (probability, speaking) => {
        setSpeechProbability(probability);
        setIsSpeaking(speaking);
        optionsRef.current.onSpeechProbability?.(probability, speaking);
      },
      onFrameProcessed: (audioFrame) => {
        bridgeRef.current?.pushAudioFrame(audioFrame);
      },
      onSpeechStart: () => {
        optionsRef.current.onSpeechStart?.();
      },
      onSpeechEnd: (audioData) => {
        bridgeRef.current?.pushSpeechSegment(audioData);
        optionsRef.current.onSpeechEnd?.(audioData);
      },
      onError: (error) => {
        optionsRef.current.onError?.(error);
      },
    };

    vadManagerRef.current = new SileroVADManager(callbacks);

    return () => {
      // Cleanup on unmount
      if (vadManagerRef.current) {
        vadManagerRef.current.destroy();
        vadManagerRef.current = null;
      }
      if (bridgeRef.current) {
        bridgeRef.current.cleanup();
        bridgeRef.current = null;
      }
    };
  }, []);

  /**
   * Start VAD processing and create MediaStreamTrack for LiveKit.
   *
   * This will:
   * 1. Initialize VADLiveKitBridge (creates AudioContext and worklet)
   * 2. Get MediaStreamTrack from bridge for LiveKit publishing
   * 3. Initialize Silero VAD (connects to microphone)
   * 4. Start VAD processing (routes audio frames through bridge)
   * 5. Return bridge's MediaStreamTrack to LiveKit
   */
  const startVAD = useCallback(async () => {
    if (!vadManagerRef.current) {
      throw new Error('VAD manager not initialized');
    }

    try {
      // Initialize VADLiveKitBridge
      bridgeRef.current = new VADLiveKitBridge();
      const bridgeTrack = await bridgeRef.current.initialize();

      setAudioTrack(bridgeTrack);

      // Initialize and start VAD (this will start calling onFrameProcessed/onSpeechEnd)
      await vadManagerRef.current.initialize();
      await vadManagerRef.current.start();

      // Notify that bridge track is ready for LiveKit publishing
      try {
        optionsRef.current.onTrackReady?.(bridgeTrack);
      } catch (callbackError) {
        // Cleanup on callback error
        if (bridgeRef.current) {
          bridgeRef.current.cleanup();
          bridgeRef.current = null;
        }
        optionsRef.current.onError?.(callbackError as Error);
        throw callbackError;
      }
    } catch (error) {
      // Cleanup bridge on any error
      if (bridgeRef.current) {
        bridgeRef.current.cleanup();
        bridgeRef.current = null;
      }
      setAudioTrack(null);
      console.error('Failed to start VAD:', error);
      optionsRef.current.onError?.(error as Error);
      throw error;
    }
  }, []);

  /**
   * Stop VAD processing but keep bridge and VAD resources allocated for quick restart.
   */
  const stopVAD = useCallback(() => {
    if (vadManagerRef.current) {
      vadManagerRef.current.stop();
    }
  }, []);

  /**
   * Destroy VAD instance and cleanup all resources including bridge.
   * Use this when done with VAD completely (e.g., component unmounting).
   */
  const destroyVAD = useCallback(() => {
    if (vadManagerRef.current) {
      vadManagerRef.current.destroy();
      vadManagerRef.current = null;
    }
    if (bridgeRef.current) {
      bridgeRef.current.cleanup();
      bridgeRef.current = null;
    }
    setAudioTrack(null);
    setStatus(MicrophoneStatus.Inactive);
    setSpeechProbability(0);
    setIsSpeaking(false);
  }, []);

  const isRecording = status === MicrophoneStatus.Recording;

  return {
    startVAD,
    stopVAD,
    destroyVAD,
    status,
    isRecording,
    speechProbability,
    isSpeaking,
    audioTrack,
    vadManager: vadManagerRef.current,
  };
}
