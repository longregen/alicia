import React from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useChatStore } from '../../stores/chatStore';
import type { MessageId, ConversationId, ToolCall, MemoryTrace } from '../../types/chat';
import type { MessageAddon, ToolData } from '../../types/components';

export interface AssistantMessageProps {
  conversationId: ConversationId | null;
  messageId: MessageId;
  onBranchSwitch?: (targetMessageId: string) => void;
  onRetry?: (messageId: MessageId) => void;
  className?: string;
}

const AssistantMessage: React.FC<AssistantMessageProps> = ({ conversationId, messageId, onBranchSwitch, onRetry, className = '' }) => {
  const message = useChatStore((state) => {
    if (!conversationId) return undefined;
    return state.conversations.get(conversationId)?.messages.get(messageId);
  });

  if (!message) return null;

  const tools: ToolData[] = message.tool_calls.map((tc: ToolCall) => ({
    id: tc.id as string,
    name: tc.tool_name,
    description: `Arguments: ${JSON.stringify(tc.arguments)}`,
    status: tc.status === 'success' ? 'completed' : tc.status === 'error' ? 'error' : 'running',
    result: tc.status === 'success' ? String(tc.result) : tc.status === 'error' ? tc.error : undefined,
  }));

  const addons: MessageAddon[] = [];

  tools.forEach((tool) => {
    const getToolEmoji = (name: string) => {
      const n = name.toLowerCase();
      if (n.includes('memory')) return '🧠';
      if (n.includes('web') || n.includes('search')) return '🌐';
      if (n.includes('file')) return '📄';
      if (n.includes('code')) return '⚙️';
      return '🔧';
    };

    addons.push({
      id: tool.id,
      type: 'tool',
      position: 'inline',
      emoji: getToolEmoji(tool.name),
      tooltip: tool.name,
    });
  });

  if (message.memory_traces.length > 0) {
    addons.push({
      id: 'memory-traces',
      type: 'memory',
      position: 'inline',
      memoryData: message.memory_traces.map((trace: MemoryTrace) => ({
        id: trace.id as string,
        content: trace.content,
        relevance: trace.relevance,
      })),
    });
  }

  return (
    <div className={`flex flex-col items-start gap-2 ${className}`}>
      <ChatBubble
        type="assistant"
        content={message.content}
        state="completed"
        timestamp={new Date(message.created_at)}
        addons={addons}
        tools={tools}
        messageId={messageId}
        conversationId={conversationId || undefined}
        onBranchSwitch={onBranchSwitch}
        onRetry={onRetry}
      />
    </div>
  );
};

export default AssistantMessage;
