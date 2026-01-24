import React from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useChatStore } from '../../stores/chatStore';
import type { MessageId, ConversationId } from '../../types/chat';

export interface UserMessageProps {
  conversationId: ConversationId | null;
  messageId: MessageId;
  onBranchSwitch?: (targetMessageId: string) => void;
  className?: string;
}

const UserMessage: React.FC<UserMessageProps> = ({ conversationId, messageId, onBranchSwitch, className = '' }) => {
  const message = useChatStore((state) => {
    if (!conversationId) return undefined;
    return state.conversations.get(conversationId)?.messages.get(messageId);
  });

  if (!message) return null;

  return (
    <div className={`flex flex-col items-end ${className}`}>
      <ChatBubble
        type="user"
        content={message.content}
        state="completed"
        timestamp={new Date(message.created_at)}
        messageId={messageId}
        conversationId={conversationId || undefined}
        onBranchSwitch={onBranchSwitch}
      />
    </div>
  );
};

export default UserMessage;
