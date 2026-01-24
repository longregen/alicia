import React, { useMemo, useCallback } from 'react';
import { Virtuoso } from 'react-virtuoso';
import UserMessage from './UserMessage';
import AssistantMessage from './AssistantMessage';
import StreamingMessage from './StreamingMessage';
import {
  useChatStore,
  selectConversationActiveBranch,
  selectConversationStreamingMessageId,
} from '../../stores/chatStore';
import type { MessageId, ConversationId } from '../../types/chat';
import { getRandomQuote } from '../../constants/quotes';

export interface MessageListProps {
  conversationId: ConversationId | null;
  className?: string;
  onBranchSwitch?: (targetMessageId: string) => void;
  onRetry?: (messageId: MessageId) => void;
}

const MessageList: React.FC<MessageListProps> = ({ conversationId, className = '', onBranchSwitch, onRetry }) => {
  const activeBranchSelector = useMemo(() => selectConversationActiveBranch(conversationId), [conversationId]);
  const messages = useChatStore(activeBranchSelector);

  const streamingIdSelector = useMemo(() => selectConversationStreamingMessageId(conversationId), [conversationId]);
  const streamingMessageId = useChatStore(streamingIdSelector);

  const streamingInBranch = streamingMessageId && messages.some(m => m.id === streamingMessageId);
  const totalItems = messages.length + (streamingMessageId && !streamingInBranch ? 1 : 0);

  const renderItem = useCallback((index: number) => {
    if (streamingMessageId && index === messages.length) {
      return <StreamingMessage conversationId={conversationId} className="mb-4" />;
    }

    const message = messages[index];
    if (!message) return null;

    const messageId = message.id as MessageId;

    // Render streaming messages via StreamingMessage component
    if (messageId === streamingMessageId) {
      return <StreamingMessage conversationId={conversationId} className="mb-4" />;
    }

    if (message.role === 'user') {
      return <UserMessage conversationId={conversationId} messageId={messageId} onBranchSwitch={onBranchSwitch} className="mb-4" />;
    } else if (message.role === 'assistant') {
      return <AssistantMessage conversationId={conversationId} messageId={messageId} onBranchSwitch={onBranchSwitch} onRetry={onRetry} className="mb-4" />;
    }

    return null;
  }, [messages, streamingMessageId, conversationId, onBranchSwitch, onRetry]);

  const randomQuote = useMemo(() => getRandomQuote(), []);

  if (totalItems === 0) {
    return (
      <div className={`flex items-center justify-center h-full text-muted-foreground ${className}`}>
        <div className="max-w-lg text-center px-6">
          <blockquote className="text-lg italic leading-relaxed">
            "{randomQuote.text}"
          </blockquote>
          <cite className="block mt-3 text-sm opacity-70">— {randomQuote.author}</cite>
        </div>
      </div>
    );
  }

  return (
    <div className={`h-full w-full ${className}`}>
      <Virtuoso
        totalCount={totalItems}
        itemContent={renderItem}
        initialTopMostItemIndex={totalItems - 1}
        followOutput="smooth"
        alignToBottom
        atBottomThreshold={100}
        className="h-full mx-2"
      />
    </div>
  );
};

export default MessageList;
