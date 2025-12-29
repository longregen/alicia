import React from 'react';
import { useConversationStore } from '../../stores/conversationStore';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { cls } from '../../utils/cls';

/**
 * ResponseControls organism component.
 *
 * Controls for managing assistant responses:
 * - Stop button during streaming
 * - Regenerate button after response complete
 * - Uses connection state to determine availability
 */

export interface ResponseControlsProps {
  /** Callback when stop is clicked */
  onStop?: () => void;
  /** Callback when regenerate is clicked */
  onRegenerate?: () => void;
  /** Whether to show the controls */
  show?: boolean;
  className?: string;
}

const ResponseControls: React.FC<ResponseControlsProps> = ({
  onStop,
  onRegenerate,
  show = true,
  className = '',
}) => {
  const currentStreamingMessageId = useConversationStore((state) => state.currentStreamingMessageId);
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;
  const isStreaming = !!currentStreamingMessageId;

  if (!show) {
    return null;
  }

  const handleStop = () => {
    if (isStreaming && onStop) {
      onStop();
    }
  };

  const handleRegenerate = () => {
    if (!isStreaming && onRegenerate) {
      onRegenerate();
    }
  };

  // Don't show controls if not connected
  if (!isConnected) {
    return null;
  }

  return (
    <div className={cls('flex items-center justify-center gap-3 p-2', className)}>
      {isStreaming ? (
        // Stop button during streaming
        <button
          onClick={handleStop}
          className={cls(
            'flex items-center gap-2 px-4 py-2 rounded-lg',
            'bg-error hover:bg-error/80',
            'text-white-text font-medium',
            'transition-all duration-200',
            'focus:outline-none focus:ring-2 focus:ring-error/50',
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
      ) : (
        // Regenerate button when not streaming
        <button
          onClick={handleRegenerate}
          className={cls(
            'flex items-center gap-2 px-4 py-2 rounded-lg',
            'bg-primary-blue hover:bg-primary-blue/80',
            'text-white-text font-medium',
            'transition-all duration-200',
            'focus:outline-none focus:ring-2 focus:ring-primary-blue/50',
            'active:scale-95'
          )}
          aria-label="Regenerate response"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          <span>Regenerate</span>
        </button>
      )}
    </div>
  );
};

export default ResponseControls;
