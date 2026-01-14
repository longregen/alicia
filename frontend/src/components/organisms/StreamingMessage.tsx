import React, { useMemo } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useConversationStore, selectCurrentStreamingMessage, selectSentences } from '../../stores/conversationStore';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import type { MessageAddon } from '../../types/components';

/**
 * StreamingMessage organism component.
 *
 * Displays currently streaming assistant responses with:
 * - Sentences as they arrive (from store)
 * - Typing cursor animation
 * - "Streaming" status indicator
 * - Data fetched from Zustand store's currentStreamingMessageId
 */

export interface StreamingMessageProps {
  className?: string;
}

const StreamingMessage: React.FC<StreamingMessageProps> = ({ className = '' }) => {
  const streamingMessage = useConversationStore(selectCurrentStreamingMessage);
  const sentencesMap = useConversationStore(selectSentences);

  // Memoize sentences derivation to avoid creating new arrays on every render
  const streamingText = useMemo(() => {
    if (!streamingMessage) return '';

    return streamingMessage.sentenceIds
      .map(id => sentencesMap[id])
      .filter(s => s && s.isComplete)
      .sort((a, b) => a.sequence - b.sequence)
      .map(s => s.content)
      .join(' ');
  }, [streamingMessage, sentencesMap]);

  if (!streamingMessage) {
    return null;
  }

  // Build streaming status addon
  const addons: MessageAddon[] = [
    {
      id: `${streamingMessage.id}-streaming`,
      type: 'icon',
      position: 'inline',
      emoji: '⚡',
      tooltip: 'Streaming response...',
    }
  ];

  return (
    <div className={`flex flex-col items-start ${className}`}>
      <ChatBubble
        type={MESSAGE_TYPES.ASSISTANT}
        content=""
        state={MESSAGE_STATES.STREAMING}
        timestamp={streamingMessage.createdAt}
        streamingText={streamingText}
        addons={addons}
        showTyping={true}
      />
    </div>
  );
};

export default StreamingMessage;
