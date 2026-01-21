import React, { useEffect, useRef, useMemo } from 'react';
import { Virtuoso, VirtuosoHandle } from 'react-virtuoso';
import UserMessage from './UserMessage';
import AssistantMessage from './AssistantMessage';
import StreamingMessage from './StreamingMessage';
import { useConversationStore, selectMessages } from '../../stores/conversationStore';
import type { MessageId, NormalizedMessage } from '../../types/streaming';

/**
 * MessageList organism component.
 *
 * Main message list with:
 * - Virtualized scrolling via react-virtuoso
 * - Renders UserMessage/AssistantMessage based on role
 * - Renders StreamingMessage at end when streaming
 * - Auto-scrolls to bottom on new messages
 * - Only shows messages in the active branch (from tip to root)
 */

export interface MessageListProps {
  className?: string;
}

/**
 * Build the active branch by walking backwards from the tip message via previousId to the root.
 *
 * When tipMessageId is provided (from the conversation's tip_message_id), it uses that
 * as the authoritative tip. This ensures the correct branch is displayed when navigating
 * between siblings.
 *
 * When tipMessageId is not provided, falls back to finding the most recently created
 * leaf message (for backwards compatibility with older conversations).
 *
 * If no messages have previousId links (legacy conversations or simple chats),
 * falls back to showing all messages sorted by timestamp.
 */
function buildActiveBranch(messages: NormalizedMessage[], tipMessageId: MessageId | null): NormalizedMessage[] {
  if (messages.length === 0) return [];

  // Check if any message has a previousId link - this indicates branch structure
  const hasBranchStructure = messages.some(m => m.previousId);
  console.log('[buildActiveBranch] hasBranchStructure:', hasBranchStructure, 'messages with previousId:', messages.filter(m => m.previousId).map(m => ({ id: m.id, previousId: m.previousId })));

  if (!hasBranchStructure) {
    // No branch structure - fall back to timestamp sorting for all messages
    console.log('[buildActiveBranch] No branch structure, returning all messages sorted by timestamp');
    return [...messages].sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());
  }

  // Build a map for quick lookup by ID
  const messageById = new Map(messages.map(m => [m.id, m]));

  // Determine the tip message
  let tip: NormalizedMessage | undefined;

  if (tipMessageId) {
    // Use the authoritative tip from the conversation
    tip = messageById.get(tipMessageId);
  }

  if (!tip) {
    // Fallback: find the most recently created leaf message
    // Build a set of all previousIds to find leaf messages (messages with no children)
    const hasChildren = new Set<string>();
    for (const msg of messages) {
      if (msg.previousId) {
        hasChildren.add(msg.previousId);
      }
    }

    // Find all leaf messages (no other message points to them as previousId)
    const leafMessages = messages.filter(m => !hasChildren.has(m.id));

    if (leafMessages.length === 0) {
      // No leaf found - this shouldn't happen in a valid conversation tree
      // Fall back to showing all messages sorted by creation time
      return [...messages].sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());
    }

    // Select the tip: the most recently created leaf message
    tip = leafMessages.reduce((latest, msg) =>
      msg.createdAt.getTime() > latest.createdAt.getTime() ? msg : latest
    );
  }

  // Walk from tip to root via previousId to build the active branch
  const activeBranch: NormalizedMessage[] = [];
  let current: NormalizedMessage | undefined = tip;

  while (current) {
    activeBranch.unshift(current); // Add to front to maintain order (root -> tip)
    current = current.previousId ? messageById.get(current.previousId) : undefined;
  }

  return activeBranch;
}

const MessageList: React.FC<MessageListProps> = ({ className = '' }) => {
  const messagesMap = useConversationStore(selectMessages);
  const currentStreamingMessageId = useConversationStore((state) => state.currentStreamingMessageId);
  const currentConversationId = useConversationStore((state) => state.currentConversationId);
  const tipMessageId = useConversationStore((state) => state.tipMessageId);
  const virtuosoRef = useRef<VirtuosoHandle>(null);

  // Memoize messages filtered to the active branch
  // The active branch is determined by the tipMessageId (from conversation's tip_message_id)
  // and walking backwards via previousId to the root
  const messages = useMemo(() => {
    // Filter messages by current conversation ID
    const messageArray = Object.values(messagesMap).filter(
      (msg) => !currentConversationId || msg.conversationId === currentConversationId
    );

    console.log('[MessageList] Building active branch with tipMessageId:', tipMessageId, 'from', messageArray.length, 'messages');
    // Build and return only the active branch using the authoritative tip
    const result = buildActiveBranch(messageArray, tipMessageId);
    console.log('[MessageList] Active branch has', result.length, 'messages:', result.map(m => m.id));
    return result;
  }, [messagesMap, currentConversationId, tipMessageId]);

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
        className="h-full mx-2"
      />
    </div>
  );
};

export default MessageList;
