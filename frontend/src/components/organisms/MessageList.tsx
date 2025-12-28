import React, { useEffect, useRef } from 'react';
import { Virtuoso, VirtuosoHandle } from 'react-virtuoso';
import UserMessage from './UserMessage';
import AssistantMessage from './AssistantMessage';
import StreamingMessage from './StreamingMessage';
import { useConversationStore, selectMessages } from '../../stores/conversationStore';
import type { MessageId } from '../../types/streaming';

/**
 * MessageList organism component.
 *
 * Main message list with:
 * - Virtualized scrolling via react-virtuoso
 * - Renders UserMessage/AssistantMessage based on role
 * - Renders StreamingMessage at end when streaming
 * - Auto-scrolls to bottom on new messages
 */

export interface MessageListProps {
  className?: string;
}

const MessageList: React.FC<MessageListProps> = ({ className = '' }) => {
  const messages = useConversationStore(selectMessages);
  const currentStreamingMessageId = useConversationStore((state) => state.currentStreamingMessageId);
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  // Auto-scroll to bottom when new messages arrive or streaming starts
  useEffect(() => {
    if (virtuosoRef.current) {
      virtuosoRef.current.scrollToIndex({
        index: messages.length - 1,
        behavior: 'smooth',
        align: 'end',
      });
    }
  }, [messages.length, currentStreamingMessageId]);

  // Calculate total items including streaming message
  const totalItems = messages.length + (currentStreamingMessageId ? 1 : 0);

  const renderItem = (index: number) => {
    // Check if this is the streaming message
    if (currentStreamingMessageId && index === messages.length) {
      return <StreamingMessage className="mb-4" />;
    }

    // Render regular message
    const message = messages[index];
    if (!message) return null;

    const messageId = message.id as MessageId;

    if (message.role === 'user') {
      return <UserMessage messageId={messageId} className="mb-4" />;
    } else if (message.role === 'assistant') {
      return <AssistantMessage messageId={messageId} className="mb-4" />;
    }

    return null;
  };

  if (totalItems === 0) {
    return (
      <div className={`flex items-center justify-center h-full text-muted-text ${className}`}>
        <p>No messages yet. Start a conversation!</p>
      </div>
    );
  }

  return (
    <div className={`h-full w-full ${className}`}>
      <Virtuoso
        ref={virtuosoRef}
        totalCount={totalItems}
        itemContent={renderItem}
        followOutput="smooth"
        alignToBottom
        className="h-full"
      />
    </div>
  );
};

export default MessageList;
