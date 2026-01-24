import React, { useMemo } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useChatStore, selectConversationStreamingMessage } from '../../stores/chatStore';
import type { ConversationId } from '../../types/chat';
import type { ToolData } from '../../types/components';

export interface StreamingMessageProps {
  conversationId: ConversationId | null;
  className?: string;
}

const StreamingMessage: React.FC<StreamingMessageProps> = ({ conversationId, className = '' }) => {
  const streamingSelector = useMemo(() => selectConversationStreamingMessage(conversationId), [conversationId]);
  const streamingMessage = useChatStore(streamingSelector);

  if (!streamingMessage) return null;

  const tools: ToolData[] = streamingMessage.tool_calls.map((tc) => ({
    id: tc.id as string,
    name: tc.tool_name,
    description: `Arguments: ${JSON.stringify(tc.arguments)}`,
    status: tc.status === 'success' ? 'completed' : tc.status === 'error' ? 'error' : 'running',
    result: tc.status === 'success' ? String(tc.result) : tc.status === 'error' ? tc.error : undefined,
  }));

  return (
    <div className={`flex flex-col items-start gap-2 ${className}`}>
      <ChatBubble
        type="assistant"
        content=""
        state="streaming"
        timestamp={new Date(streamingMessage.created_at)}
        streamingText={streamingMessage.content}
        tools={tools}
      />
    </div>
  );
};

export default StreamingMessage;
