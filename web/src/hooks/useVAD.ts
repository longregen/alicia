import { useState, useCallback, useEffect, useRef } from 'react';
import { MicrophoneStatus } from '../types/streaming';
import { getMicrophoneManager, MicrophoneManagerCallbacks } from '../utils/microphoneManager';

export interface UseVADOptions {
  onSpeechStart?: () => void;
  onSpeechEnd?: (audioData: Float32Array) => void;
  onSpeechProbability?: (probability: number, isSpeaking: boolean) => void;
  onStatusChange?: (status: MicrophoneStatus) => void;
  onError?: (error: Error) => void;
  onTrackReady?: (track: MediaStreamTrack) => void;
}

export interface UseVADReturn {
  startVAD: () => Promise<void>;
  stopVAD: () => void;
  status: MicrophoneStatus;
  isRecording: boolean;
  speechProbability: number;
  isSpeaking: boolean;
  audioTrack: MediaStreamTrack | null;
}

export function useVAD(options: UseVADOptions = {}): UseVADReturn {
  const [status, setStatus] = useState<MicrophoneStatus>(MicrophoneStatus.Inactive);
  const [speechProbability, setSpeechProbability] = useState(0);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [audioTrack, setAudioTrack] = useState<MediaStreamTrack | null>(null);

  const optionsRef = useRef<UseVADOptions>(options);

  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  useEffect(() => {
    const manager = getMicrophoneManager();
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

  const startVAD = useCallback(async () => {
    const manager = getMicrophoneManager();
    await manager.start();
  }, []);

  const stopVAD = useCallback(() => {
    const manager = getMicrophoneManager();
    manager.stop();
  }, []);

  const isRecording = status === MicrophoneStatus.Recording || status === MicrophoneStatus.Sending;

  return { startVAD, stopVAD, status, isRecording, speechProbability, isSpeaking, audioTrack };
}
