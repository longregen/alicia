import React, { useState, useRef, useCallback, useEffect } from 'react';
import InputSendButton from '../atoms/InputSendButton';
import MicrophoneVAD from '../molecules/MicrophoneVAD';
import { cls } from '../../utils/cls';
import { MicrophoneStatus } from '../../types/streaming';
import { useVAD } from '../../hooks/useVAD';

/**
 * InputArea organism component.
 *
 * Input area with voice and text support:
 * - Text input with resize support
 * - Voice input via MicrophoneVAD with Silero VAD
 * - Send button
 * - Handles both text and voice submission
 * - Streams audio to LiveKit for server-side transcription
 */

export interface InputAreaProps {
  /** Callback when user submits a message */
  onSend?: (message: string, isVoice: boolean) => void;
  /** Callback to publish audio track to LiveKit */
  onPublishAudioTrack?: (track: MediaStreamTrack) => Promise<void>;
  /** Callback when voice input is toggled */
  onVoiceActiveChange?: (active: boolean) => void;
  /** Whether voice input is currently active (LiveKit connected) */
  voiceActive?: boolean;
  /** Whether input is disabled */
  disabled?: boolean;
  /** Placeholder text for input */
  placeholder?: string;
  /** Current conversation ID - used to autofocus when switching conversations */
  conversationId?: string | null;
  className?: string;
}

const InputArea: React.FC<InputAreaProps> = ({
  onSend,
  onPublishAudioTrack,
  onVoiceActiveChange,
  voiceActive = false,
  disabled = false,
  placeholder = 'Type a message...',
  conversationId,
  className = '',
}) => {
  const [inputValue, setInputValue] = useState('');
  const trackPublishedRef = useRef<boolean>(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Autofocus input when conversation changes
  useEffect(() => {
    if (conversationId && inputRef.current) {
      inputRef.current.focus();
    }
  }, [conversationId]);

  // Reset track published state when voice becomes inactive
  useEffect(() => {
    if (!voiceActive) {
      trackPublishedRef.current = false;
    }
  }, [voiceActive]);

  // Initialize VAD hook with LiveKit audio streaming
  const { status: microphoneStatus, startVAD, stopVAD, isSpeaking, speechProbability } = useVAD({
    onStatusChange: (status) => {
      console.log('VAD status changed:', status);
    },
    onSpeechEnd: (audioData: Float32Array) => {
      console.log('Speech segment captured:', audioData.length, 'samples');
    },
    onTrackReady: useCallback(async (track: MediaStreamTrack) => {
      if (onPublishAudioTrack && !trackPublishedRef.current) {
        try {
          await onPublishAudioTrack(track);
          trackPublishedRef.current = true;
          console.log('Audio track published to LiveKit for transcription');
        } catch (error) {
          console.error('Failed to publish audio track:', error);
        }
      }
    }, [onPublishAudioTrack]),
    onError: (error) => {
      console.error('VAD error:', error);
    },
  });

  const handleTextSubmit = (value: string) => {
    if (value.trim() && onSend) {
      onSend(value.trim(), false);
      setInputValue('');
    }
  };

  const handleSendClick = () => {
    handleTextSubmit(inputValue);
  };

  const handleVoiceClick = () => {
    onVoiceActiveChange?.(!voiceActive);
  };

  const canSend = inputValue.trim().length > 0;
  const isRecording = voiceActive && microphoneStatus === MicrophoneStatus.Recording;

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleTextSubmit(inputValue);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
  };

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleTextSubmit(inputValue);
    }
  };

  return (
    <form
      className={cls('input-bar flex items-end gap-3 p-4 md:p-5 bg-elevated', className)}
      onSubmit={handleFormSubmit}
    >
      {/* Voice input button */}
      <MicrophoneVAD
        microphoneStatus={microphoneStatus}
        isSpeaking={isSpeaking}
        speechProbability={speechProbability}
        onStartVAD={startVAD}
        onStopVAD={stopVAD}
        disabled={disabled}
        onClick={handleVoiceClick}
        className="flex-shrink-0"
      />

      {/* Text input - using simple input for e2e test compatibility */}
      <div className="flex-1">
        <input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleInputKeyDown}
          placeholder={placeholder}
          disabled={disabled || isRecording}
          autoFocus
          className="input rounded-3xl"
          aria-label="Message input"
        />
      </div>

      {/* Send button */}
      <InputSendButton
        onSend={handleSendClick}
        canSend={canSend}
        disabled={disabled || isRecording}
        tooltipText={canSend ? 'Send message' : 'Type a message to send'}
        className="flex-shrink-0"
      />
    </form>
  );
};

export default InputArea;
