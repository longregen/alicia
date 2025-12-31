import React, { useState } from 'react';
import MessageList from './MessageList';
import InputArea from './InputArea';
import ResponseControls from './ResponseControls';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { useLiveKit } from '../../hooks/useLiveKit';
import { cls } from '../../utils/cls';

/**
 * ChatWindow organism component.
 *
 * Main chat container combining:
 * - MessageList for conversation history
 * - InputArea for user input with VAD support
 * - ResponseControls for stop/regenerate
 * - LiveKit integration for voice streaming
 * - Overall layout structure
 */

export interface ChatWindowProps {
  /** Callback when user sends a message */
  onSendMessage?: (message: string, isVoice: boolean) => void;
  /** Callback when user stops streaming */
  onStopStreaming?: () => void;
  /** Callback when user regenerates response */
  onRegenerateResponse?: () => void;
  /** Conversation ID for LiveKit connection (when using VAD) */
  conversationId?: string | null;
  /** Whether to use Silero VAD for voice input */
  useSileroVAD?: boolean;
  /** Whether to show response controls */
  showControls?: boolean;
  className?: string;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  onSendMessage,
  onStopStreaming,
  onRegenerateResponse,
  conversationId = null,
  useSileroVAD = false,
  showControls = true,
  className = '',
}) => {
  const [voiceModeActive, setVoiceModeActive] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const [voiceSelectorOpen, setVoiceSelectorOpen] = useState(false);
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;

  // Initialize LiveKit when voice mode is active or using Silero VAD
  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
  } = useLiveKit((voiceModeActive || useSileroVAD) ? conversationId : null);

  const toggleVoiceMode = () => {
    setVoiceModeActive(!voiceModeActive);
    if (voiceModeActive) {
      setIsRecording(false);
    }
  };

  const toggleRecording = () => {
    setIsRecording(!isRecording);
  };

  const toggleVoiceSelector = () => {
    setVoiceSelectorOpen(!voiceSelectorOpen);
  };

  // Handle keyboard events for closing voice selector
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && voiceSelectorOpen) {
        setVoiceSelectorOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [voiceSelectorOpen]);

  const handleSendMessage = (message: string, isVoice: boolean) => {
    if (onSendMessage) {
      onSendMessage(message, isVoice);
    }
  };

  const handleStop = () => {
    if (onStopStreaming) {
      onStopStreaming();
    }
  };

  const handleRegenerate = () => {
    if (onRegenerateResponse) {
      onRegenerateResponse();
    }
  };

  // Determine voice connection status text
  const getVoiceConnectionText = () => {
    if (liveKitConnected) return 'Connected';
    if (liveKitError) return 'Error';
    if (voiceModeActive) return 'Connecting';
    return 'Disconnected';
  };

  return (
    <div className={cls('flex flex-col h-full bg-elevated', className)}>
      {/* Connection status indicator - always visible when voice mode is active */}
      <div className={cls(
        'flex items-center justify-center gap-2 px-4 py-2 border-b border',
        voiceModeActive
          ? (liveKitConnected ? 'bg-success-subtle text-success' : 'bg-warning-subtle text-warning')
          : (!isConnected ? 'bg-warning-subtle text-warning' : 'hidden')
      )}>
        <svg className="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
        <span className="text-sm font-medium">
          {voiceModeActive ? getVoiceConnectionText() : (
            <>
              {connectionStatus === ConnectionStatus.Connecting && 'Connecting'}
              {connectionStatus === ConnectionStatus.Reconnecting && 'Reconnecting'}
              {connectionStatus === ConnectionStatus.Disconnected && 'Disconnected'}
              {connectionStatus === ConnectionStatus.Connected && 'Connected'}
              {connectionStatus === ConnectionStatus.Error && 'Connection error'}
            </>
          )}
        </span>
      </div>

      {/* LiveKit error indicator (when using VAD) */}
      {(useSileroVAD || voiceModeActive) && liveKitError && (
        <div className="flex items-center justify-center gap-2 px-4 py-2 bg-error-subtle text-error border-b border-error">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <span className="text-sm font-medium">{liveKitError}</span>
        </div>
      )}

      {/* Message list */}
      <div className="flex-1 overflow-hidden">
        <MessageList />
      </div>

      {/* Response controls */}
      {showControls && (
        <ResponseControls
          onStop={handleStop}
          onRegenerate={handleRegenerate}
          show={isConnected}
        />
      )}

      {/* Voice controls - visible when voice mode is active */}
      {voiceModeActive && (
        <div className="flex items-center justify-center gap-4 p-4 border-t border">
          {/* Audio input indicator */}
          <div className="flex items-center gap-2 text-sm text-muted">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
            </svg>
            <span>Voice Input</span>
          </div>

          {/* Record button */}
          <button
            className={cls(
              'w-12 h-12 rounded-full flex items-center justify-center transition-all duration-200 border-2',
              isRecording
                ? 'bg-error border-error text-on-emphasis animate-pulse-recording'
                : 'bg-elevated border hover:border-accent hover:scale-105'
            )}
            onClick={toggleRecording}
            aria-label={isRecording ? 'Stop recording' : 'Start recording'}
          >
            {isRecording ? (
              <div className="w-4 h-4 bg-on-emphasis rounded-sm" />
            ) : (
              <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
              </svg>
            )}
          </button>

          {/* Voice selector toggle */}
          <div className="relative">
            <button
              className="flex items-center gap-2 px-4 py-2 bg-surface border-2 border rounded-full text-sm font-medium hover:bg-sunken hover:border-accent transition-all"
              aria-label="Select voice"
              onClick={toggleVoiceSelector}
              aria-expanded={voiceSelectorOpen}
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
              <span>Voice</span>
            </button>

            {/* Voice selector panel */}
            {voiceSelectorOpen && (
              <div className="absolute bottom-full left-0 mb-2 w-80 bg-elevated border rounded-lg shadow-lg p-2 animate-fade-in">
                <div className="flex justify-between items-center p-3 pb-2 border-b border">
                  <h3 className="text-base font-semibold text-default">Select Voice</h3>
                  <button
                    onClick={toggleVoiceSelector}
                    className="btn-ghost w-6 h-6 flex items-center justify-center rounded text-muted hover:bg-surface text-xl"
                  >
                    ×
                  </button>
                </div>
                <div className="p-4 flex flex-col gap-2">
                  <button className="w-full text-left px-3 py-2 text-sm hover:bg-surface rounded transition-colors">Sarah</button>
                  <button className="w-full text-left px-3 py-2 text-sm hover:bg-surface rounded transition-colors">Adam</button>
                  <button className="w-full text-left px-3 py-2 text-sm hover:bg-surface rounded transition-colors">Nicole</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Voice mode toggle button */}
      <div className="flex items-center justify-between p-2 border-t border">
        <button
          className={cls(
            'flex items-center gap-2 px-4 py-2 rounded-full transition-all duration-200',
            voiceModeActive
              ? 'bg-accent text-on-emphasis'
              : 'bg-surface text-muted hover:bg-sunken'
          )}
          onClick={toggleVoiceMode}
          aria-label={voiceModeActive ? 'Disable voice mode' : 'Enable voice mode'}
          aria-pressed={voiceModeActive}
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
          </svg>
          <span className="text-sm font-medium">Voice Mode</span>
        </button>
      </div>

      {/* Input area */}
      <InputArea
        onSend={handleSendMessage}
        onPublishAudioTrack={(useSileroVAD || voiceModeActive) ? publishAudioTrack : undefined}
        disabled={!isConnected || ((useSileroVAD || voiceModeActive) && !liveKitConnected)}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        useSileroVAD={useSileroVAD || voiceModeActive}
      />
    </div>
  );
};

export default ChatWindow;
