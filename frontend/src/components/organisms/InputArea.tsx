import React, { useState, useRef, useCallback } from 'react';
import RecordingButtonForInput from '../atoms/RecordingButtonForInput';
import InputSendButton from '../atoms/InputSendButton';
import MicrophoneVAD from '../molecules/MicrophoneVAD';
import { cls } from '../../utils/cls';
import { RECORDING_STATES } from '../../mockData';
import { MicrophoneStatus } from '../../types/streaming';
import { useVAD } from '../../hooks/useVAD';
import type { RecordingState } from '../../types/components';

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
  /** Callback to publish audio track to LiveKit (when useSileroVAD is true) */
  onPublishAudioTrack?: (track: MediaStreamTrack) => Promise<void>;
  /** Whether input is disabled */
  disabled?: boolean;
  /** Placeholder text for input */
  placeholder?: string;
  /** Whether to use Silero VAD for voice input */
  useSileroVAD?: boolean;
  className?: string;
}

const InputArea: React.FC<InputAreaProps> = ({
  onSend,
  onPublishAudioTrack,
  disabled = false,
  placeholder = 'Type a message...',
  useSileroVAD = false,
  className = '',
}) => {
  const [inputValue, setInputValue] = useState('');
  const [recordingState, setRecordingState] = useState<RecordingState>(RECORDING_STATES.IDLE);
  const trackPublishedRef = useRef<boolean>(false);

  // Initialize VAD hook with LiveKit audio streaming
  const { status: microphoneStatus, vadManager } = useVAD({
    onStatusChange: (status) => {
      console.log('VAD status changed:', status);
    },
    onSpeechEnd: (audioData: Float32Array) => {
      console.log('Speech segment captured:', audioData.length, 'samples');
      // Speech segment captured - audio streaming handled by VAD bridge
      // Server responds with Transcription protocol messages
    },
    onTrackReady: useCallback(async (track: MediaStreamTrack) => {
      // Publish the audio track to LiveKit when ready
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

  const handleTextChange = (value: string) => {
    setInputValue(value);
  };

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
    if (!useSileroVAD) {
      // Toggle recording state for non-VAD mode
      const newState = recordingState === RECORDING_STATES.IDLE
        ? RECORDING_STATES.RECORDING
        : RECORDING_STATES.IDLE;
      setRecordingState(newState);
    }
    // VAD mode handles its own state
  };

  const canSend = inputValue.trim().length > 0;
  const isRecording = microphoneStatus === MicrophoneStatus.Recording || recordingState === RECORDING_STATES.RECORDING;

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleTextSubmit(inputValue);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    handleTextChange(e.target.value);
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
      {useSileroVAD ? (
        <MicrophoneVAD
          useSileroVAD={true}
          vadManager={vadManager || undefined}
          disabled={disabled}
          className="flex-shrink-0"
        />
      ) : (
        <RecordingButtonForInput
          state={recordingState}
          onClick={handleVoiceClick}
          disabled={disabled}
          size="md"
          className="flex-shrink-0"
        />
      )}

      {/* Text input - using simple input for e2e test compatibility */}
      <div className="flex-1">
        <input
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
