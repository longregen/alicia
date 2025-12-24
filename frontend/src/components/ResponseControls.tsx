import { useState } from 'react';
import { useMessageContext } from '../contexts/MessageContext';
import { Message } from '../types/models';

interface ResponseControlsProps {
  onStop: () => void;
  onRegenerate: (targetId: string) => void;
  disabled?: boolean;
}

export function ResponseControls({ onStop, onRegenerate, disabled = false }: ResponseControlsProps) {
  const { isGenerating, messages } = useMessageContext();
  const [isStopping, setIsStopping] = useState(false);

  // Find the last assistant message
  const lastAssistantMessage = messages
    .filter((msg: Message) => msg.role === 'assistant')
    .sort((a: Message, b: Message) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    )[0];

  const handleStop = async () => {
    setIsStopping(true);
    await onStop();
    // Keep visual feedback for a moment
    setTimeout(() => setIsStopping(false), 1000);
  };

  const handleRegenerate = () => {
    if (lastAssistantMessage) {
      onRegenerate(lastAssistantMessage.id);
    }
  };

  // Don't show anything if disabled or no messages
  if (disabled || messages.length === 0) {
    return null;
  }

  return (
    <div className="response-controls" style={{
      display: 'flex',
      gap: '8px',
      padding: '8px',
      justifyContent: 'center',
      alignItems: 'center',
    }}>
      {isGenerating ? (
        <button
          onClick={handleStop}
          disabled={isStopping}
          className="control-button stop-button"
          style={{
            padding: '8px 16px',
            border: 'none',
            borderRadius: '8px',
            background: isStopping ? '#ccc' : '#ff5252',
            color: 'white',
            cursor: isStopping ? 'not-allowed' : 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            fontSize: '14px',
            fontWeight: '500',
            transition: 'all 0.2s ease',
            opacity: isStopping ? 0.6 : 1,
          }}
          onMouseEnter={(e) => {
            if (!isStopping) {
              e.currentTarget.style.background = '#ff1744';
            }
          }}
          onMouseLeave={(e) => {
            if (!isStopping) {
              e.currentTarget.style.background = '#ff5252';
            }
          }}
          title={isStopping ? 'Stopping...' : 'Stop generation'}
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <rect x="3" y="3" width="10" height="10" rx="1" />
          </svg>
          {isStopping ? 'Stopping...' : 'Stop'}
        </button>
      ) : (
        lastAssistantMessage && (
          <button
            onClick={handleRegenerate}
            className="control-button regenerate-button"
            style={{
              padding: '8px 16px',
              border: '1px solid #ddd',
              borderRadius: '8px',
              background: 'white',
              color: '#666',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              fontSize: '14px',
              fontWeight: '500',
              transition: 'all 0.2s ease',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = '#f5f5f5';
              e.currentTarget.style.borderColor = '#999';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'white';
              e.currentTarget.style.borderColor = '#ddd';
            }}
            title="Regenerate response"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M13.65 2.35a7.5 7.5 0 0 0-11.3 0L1 1v4h4L3.5 3.5a5.5 5.5 0 1 1-.8 5.6l-1.5.8a7.5 7.5 0 1 0 12.45-7.55z" />
            </svg>
            Regenerate
          </button>
        )
      )}
    </div>
  );
}
