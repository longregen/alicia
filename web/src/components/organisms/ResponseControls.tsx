import React from 'react';
import { useChatStore } from '../../stores/chatStore';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { cls } from '../../utils/cls';
import type { ConversationId } from '../../types/chat';

export interface ResponseControlsProps {
  /** Conversation ID to check streaming state for */
  conversationId?: ConversationId | null;
  /** Callback when stop is clicked */
  onStop?: () => void;
  /** Whether to show the controls */
  show?: boolean;
  className?: string;
}

const ResponseControls: React.FC<ResponseControlsProps> = ({
  conversationId = null,
  onStop,
  show = true,
  className = '',
}) => {
  const streamingMessageId = useChatStore((state) => {
    if (!conversationId) return null;
    return state.conversations.get(conversationId)?.streamingMessageId ?? null;
  });
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;
  const isStreaming = !!streamingMessageId;

  // Only show when streaming (Stop button only)
  if (!show || !isConnected || !isStreaming) {
    return null;
  }

  const handleStop = () => {
    if (onStop) {
      onStop();
    }
  };

  return (
    <div className={cls('flex items-center justify-center gap-3 p-2', className)}>
      <button
        onClick={handleStop}
        className={cls(
          'layout-center-gap px-4 py-2 rounded-lg',
          'bg-destructive hover:bg-destructive/90',
          'text-destructive-foreground font-medium',
          'transition-all duration-200',
          'focus:outline-none focus:ring-2 focus:ring-destructive/50',
          'active:scale-95'
        )}
        aria-label="Stop response"
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z"
            clipRule="evenodd"
          />
        </svg>
        <span>Stop</span>
      </button>
    </div>
  );
};

export default ResponseControls;
