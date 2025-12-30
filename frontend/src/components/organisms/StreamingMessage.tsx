import React from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useConversationStore, selectCurrentStreamingMessage } from '../../stores/conversationStore';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import type { MessageAddon } from '../../types/components';
import type { MessageSentence } from '../../types/streaming';

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
  const getSentences = useConversationStore((state) => state.getMessageSentences);

  if (!streamingMessage) {
    return null;
  }

  // Get sentences for the streaming message
  const sentences: MessageSentence[] = getSentences(streamingMessage.id);

  // Combine all complete sentences for streaming display
  const streamingText = sentences
    .filter((s) => s.isComplete)
    .sort((a, b) => a.sequence - b.sequence)
    .map((s) => s.content)
    .join(' ');

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
    <ChatBubble
      type={MESSAGE_TYPES.ASSISTANT}
      content=""
      state={MESSAGE_STATES.STREAMING}
      timestamp={streamingMessage.createdAt}
      streamingText={streamingText}
      addons={addons}
      showTyping={true}
      className={className}
    />
  );
};

export default StreamingMessage;
