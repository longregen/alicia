import React from 'react';
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
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;

  // Initialize LiveKit only when using Silero VAD
  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
  } = useLiveKit(useSileroVAD ? conversationId : null);

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

  return (
    <div className={cls('flex flex-col h-full bg-main-bg', className)}>
      {/* Connection status indicator */}
      {!isConnected && (
        <div className="flex items-center justify-center gap-2 px-4 py-2 bg-warning/20 text-warning border-b border-warning/50">
          <svg className="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <span className="text-sm font-medium">
            {connectionStatus === ConnectionStatus.Connecting && 'Connecting...'}
            {connectionStatus === ConnectionStatus.Reconnecting && 'Reconnecting...'}
            {connectionStatus === ConnectionStatus.Disconnected && 'Disconnected'}
            {connectionStatus === ConnectionStatus.Error && 'Connection error'}
          </span>
        </div>
      )}

      {/* LiveKit error indicator (when using VAD) */}
      {useSileroVAD && liveKitError && (
        <div className="flex items-center justify-center gap-2 px-4 py-2 bg-error/20 text-error border-b border-error/50">
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

      {/* Input area */}
      <InputArea
        onSend={handleSendMessage}
        onPublishAudioTrack={useSileroVAD ? publishAudioTrack : undefined}
        disabled={!isConnected || (useSileroVAD && !liveKitConnected)}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        useSileroVAD={useSileroVAD}
      />
    </div>
  );
};

export default ChatWindow;
