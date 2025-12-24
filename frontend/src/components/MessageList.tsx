import { useEffect, useRef } from 'react';
import { Message } from '../types/models';
import { MessageBubble } from './MessageBubble';
import { useMessageContext } from '../contexts/MessageContext';

interface MessageListProps {
  messages: Message[];
  loading: boolean;
}

export function MessageList({ messages, loading }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const { toolUsages } = useMessageContext();

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  if (loading) {
    return <div className="message-list loading">Loading messages...</div>;
  }

  if (messages.length === 0) {
    return (
      <div className="message-list empty">
        <div className="empty-state">
          Start a conversation with Alicia
        </div>
      </div>
    );
  }

  return (
    <div className="message-list">
      {messages.map((message, index) => {
        // Associate tool usages with their corresponding message using messageId
        const messageToolUsages = toolUsages.filter(
          usage => usage.request.messageId === message.id
        );

        // Check if this is the latest message
        const isLatestMessage = index === messages.length - 1;

        return (
          <MessageBubble
            key={message.id}
            message={message}
            toolUsages={messageToolUsages}
            isLatestMessage={isLatestMessage}
          />
        );
      })}
      <div ref={bottomRef} />
    </div>
  );
}
