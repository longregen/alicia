import React, { useEffect, useRef, useMemo } from 'react';
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
  const messagesMap = useConversationStore(selectMessages);
  const currentStreamingMessageId = useConversationStore((state) => state.currentStreamingMessageId);
  const currentConversationId = useConversationStore((state) => state.currentConversationId);
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  // Memoize sorted messages to avoid creating new arrays on every render
  // Sort messages using previousId chain for correct conversation order,
  // falling back to timestamp when no previousId chain exists
  const messages = useMemo(() => {
    // Filter messages by current conversation ID
    const messageArray = Object.values(messagesMap).filter(
      (msg) => !currentConversationId || msg.conversationId === currentConversationId
    );

    // Build a map for quick lookup
    const messageById = new Map(messageArray.map(m => [m.id, m]));

    // Track which messages have been positioned
    const positioned = new Set<string>();
    const result: typeof messageArray = [];

    // Helper to recursively position a message and all its dependencies
    const positionMessage = (msg: typeof messageArray[0]): void => {
      if (positioned.has(msg.id)) return;

      // If this message has a previousId, position that first
      if (msg.previousId && messageById.has(msg.previousId)) {
        const prevMsg = messageById.get(msg.previousId)!;
        positionMessage(prevMsg);
      }

      // Now position this message
      if (!positioned.has(msg.id)) {
        result.push(msg);
        positioned.add(msg.id);
      }
    };

    // First, sort all messages by timestamp as a baseline
    const sorted = [...messageArray].sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());

    // Then process each message, respecting previousId chains
    for (const msg of sorted) {
      positionMessage(msg);
    }

    return result;
  }, [messagesMap, currentConversationId]);

  // Calculate total items including streaming message
  const totalItems = messages.length + (currentStreamingMessageId ? 1 : 0);

  // Scroll to bottom when messages change or streaming state changes
  // Note: This scrolls on any change, not just additions
  useEffect(() => {
    if (virtuosoRef.current && totalItems > 0) {
      virtuosoRef.current.scrollToIndex({
        index: totalItems - 1,
        behavior: 'smooth',
        align: 'end',
      });
    }
  }, [totalItems]);

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
      <div className={`flex items-center justify-center h-full text-muted-foreground ${className}`}>
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
