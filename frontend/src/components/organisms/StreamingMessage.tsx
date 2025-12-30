import React from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { shallow } from 'zustand/shallow';
import { useConversationStore, selectCurrentStreamingMessage } from '../../stores/conversationStore';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import type { MessageAddon } from '../../types/components';

// Stable empty array to avoid infinite re-renders
const EMPTY_SENTENCES: never[] = [];

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
  // Use shallow comparison to avoid infinite re-renders from array selector
  const sentences = useConversationStore(
    (state) => streamingMessage ? state.getMessageSentences(streamingMessage.id) : EMPTY_SENTENCES,
    shallow
  );

  if (!streamingMessage) {
    return null;
  }

  // Combine all complete sentences for streaming display
  const streamingText = sentences
    .filter(s => s.isComplete)
    .sort((a, b) => a.sequence - b.sequence)
    .map(s => s.content)
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
