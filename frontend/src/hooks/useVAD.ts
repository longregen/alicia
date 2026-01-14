import { useState, useCallback, useEffect, useRef } from 'react';
import { MicrophoneStatus } from '../types/streaming';
import { getMicrophoneManager, MicrophoneManagerCallbacks } from '../utils/microphoneManager';

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
}

/**
 * Hook for managing Silero VAD integration with LiveKit audio streaming.
 *
 * This hook uses a singleton MicrophoneManager that:
 * - Initializes microphone and VAD once
 * - Reuses the same audio track across all LiveKit rooms
 * - Provides consistent speech detection events
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

  const optionsRef = useRef<UseVADOptions>(options);

  // Keep options ref updated
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  // Subscribe to MicrophoneManager on mount
  useEffect(() => {
    const manager = getMicrophoneManager();

    // Initialize state from manager's current state
    setStatus(manager.getStatus());
    setSpeechProbability(manager.getSpeechProbability());
    setIsSpeaking(manager.getIsSpeaking());
    setAudioTrack(manager.getAudioTrack());

    const callbacks: MicrophoneManagerCallbacks = {
      onStatusChange: (newStatus) => {
        setStatus(newStatus);
        optionsRef.current.onStatusChange?.(newStatus);
      },
      onSpeechProbability: (probability, speaking) => {
        setSpeechProbability(probability);
        setIsSpeaking(speaking);
        optionsRef.current.onSpeechProbability?.(probability, speaking);
      },
      onSpeechStart: () => {
        optionsRef.current.onSpeechStart?.();
      },
      onSpeechEnd: (audioData) => {
        optionsRef.current.onSpeechEnd?.(audioData);
      },
      onTrackReady: (track) => {
        setAudioTrack(track);
        optionsRef.current.onTrackReady?.(track);
      },
      onError: (error) => {
        optionsRef.current.onError?.(error);
      },
    };

    const unsubscribe = manager.subscribe(callbacks);

    return () => {
      unsubscribe();
    };
  }, []);

  /**
   * Start VAD processing and create MediaStreamTrack for LiveKit.
   */
  const startVAD = useCallback(async () => {
    const manager = getMicrophoneManager();
    await manager.start();
  }, []);

  /**
   * Stop VAD processing but keep resources allocated for quick restart.
   */
  const stopVAD = useCallback(() => {
    const manager = getMicrophoneManager();
    manager.stop();
  }, []);

  const isRecording = status === MicrophoneStatus.Recording || status === MicrophoneStatus.Sending;

  return {
    startVAD,
    stopVAD,
    status,
    isRecording,
    speechProbability,
    isSpeaking,
    audioTrack,
  };
}
