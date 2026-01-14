import React, { useState } from 'react';
import MessageList from './MessageList';
import InputArea from './InputArea';
import ResponseControls from './ResponseControls';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { useAudioStore } from '../../stores/audioStore';
import { useLiveKit } from '../../hooks/useLiveKit';
import { cls } from '../../utils/cls';

/**
 * ChatWindow organism component.
 *
 * Main chat container combining:
 * - Chat header with title, conversation ID, and audio controls
 * - MessageList for conversation history
 * - InputArea for user input (text + voice)
 * - ResponseControls for stop/regenerate
 */

export interface ChatWindowProps {
  onSendMessage?: (message: string, isVoice: boolean) => void;
  onStopStreaming?: () => void;
  onRegenerateResponse?: () => void;
  conversationId?: string | null;
  conversationTitle?: string;
  showControls?: boolean;
  className?: string;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  onSendMessage,
  onStopStreaming,
  onRegenerateResponse,
  conversationId = null,
  conversationTitle = 'Conversation',
  showControls = true,
  className = '',
}) => {
  // Voice input state - controlled by mic button in InputArea
  const [voiceActive, setVoiceActive] = useState(false);

  // Connection state
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;

  // Audio output state from store
  const audioOutputEnabled = useAudioStore((state) => state.playback.audioOutputEnabled);
  const toggleAudioOutput = useAudioStore((state) => state.toggleAudioOutput);

  // LiveKit connection - only connect when voice is active
  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
    sendStop,
  } = useLiveKit(voiceActive ? conversationId : null);

  const handleSendMessage = (message: string, isVoice: boolean) => {
    onSendMessage?.(message, isVoice);
  };

  const handleStop = async () => {
    onStopStreaming?.();
    if (sendStop && conversationId) {
      try {
        await sendStop();
      } catch (error) {
        console.error('Failed to send stop command:', error);
      }
    }
  };

  const handleRegenerate = () => {
    onRegenerateResponse?.();
  };

  const handleVoiceActiveChange = (active: boolean) => {
    setVoiceActive(active);
  };

  // Connection status display logic
  const getConnectionStatusText = () => {
    if (voiceActive) {
      if (liveKitConnected) return 'Voice connected';
      if (liveKitError) return 'Voice error';
      return 'Connecting voice...';
    }
    switch (connectionStatus) {
      case ConnectionStatus.Connecting: return 'Connecting';
      case ConnectionStatus.Reconnecting: return 'Reconnecting';
      case ConnectionStatus.Disconnected: return 'Disconnected';
      case ConnectionStatus.Connected: return null; // Don't show when connected
      case ConnectionStatus.Error: return 'Connection error';
      default: return null;
    }
  };

  const connectionStatusText = getConnectionStatusText();
  const showConnectionStatus = connectionStatusText !== null;

  return (
    <div className={cls('stack h-full bg-background', className)}>
      {/* Header */}
      <header className="h-14 flex-between px-4 border-b border-border shrink-0">
        <div className="row-3">
          <h2 className="font-medium text-foreground">{conversationTitle}</h2>
          <span className="badge badge-default">{conversationId || 'No ID'}</span>
        </div>
        <button
          onClick={toggleAudioOutput}
          className={cls(
            'btn-icon',
            audioOutputEnabled && 'text-accent bg-accent/10'
          )}
          aria-label={audioOutputEnabled ? 'Disable audio output' : 'Enable audio output'}
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
            {audioOutputEnabled ? (
              <path
                fillRule="evenodd"
                d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414zm-2.829 2.828a1 1 0 011.415 0A5.983 5.983 0 0115 10a5.984 5.984 0 01-1.757 4.243 1 1 0 01-1.415-1.415A3.984 3.984 0 0013 10a3.983 3.983 0 00-1.172-2.828 1 1 0 010-1.415z"
                clipRule="evenodd"
              />
            ) : (
              <path
                fillRule="evenodd"
                d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM12.293 7.293a1 1 0 011.414 0L15 8.586l1.293-1.293a1 1 0 111.414 1.414L16.414 10l1.293 1.293a1 1 0 01-1.414 1.414L15 11.414l-1.293 1.293a1 1 0 01-1.414-1.414L13.586 10l-1.293-1.293a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            )}
          </svg>
        </button>
      </header>

      {/* Connection status - only show when not fully connected */}
      {showConnectionStatus && (
        <div className={cls(
          'status-bar',
          voiceActive && liveKitError ? 'status-bar-error' :
          voiceActive && liveKitConnected ? 'status-bar-success' :
          'status-bar-warning'
        )}>
          <svg className="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <span>{connectionStatusText}</span>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-hidden">
        <MessageList />
      </div>

      {/* Controls */}
      {showControls && (
        <ResponseControls
          onStop={handleStop}
          onRegenerate={handleRegenerate}
          show={isConnected}
        />
      )}

      {/* Input */}
      <InputArea
        onSend={handleSendMessage}
        onPublishAudioTrack={voiceActive ? publishAudioTrack : undefined}
        onVoiceActiveChange={handleVoiceActiveChange}
        voiceActive={voiceActive}
        disabled={!isConnected}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        conversationId={conversationId}
      />
    </div>
  );
};

export default ChatWindow;
