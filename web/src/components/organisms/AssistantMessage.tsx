import React, { useMemo } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import { useChatStore, selectConversationStreamingMessage } from '../../stores/chatStore';
import { getToolEmoji } from '../atoms/ComplexAddons';
import type { ToolDetail } from '../atoms/ComplexAddons';
import type { MessageId, ConversationId, ToolCall, MemoryTrace } from '../../types/chat';
import type { MessageAddon } from '../../types/components';

export interface AssistantMessageProps {
  conversationId: ConversationId | null;
  messageId?: MessageId;
  streaming?: boolean;
  onBranchSwitch?: (targetMessageId: string) => void;
  onRetry?: (messageId: MessageId) => void;
  className?: string;
}

function buildToolDetails(toolCalls: ToolCall[]): ToolDetail[] {
  return toolCalls.map((tc) => ({
    id: tc.id as string,
    name: tc.tool_name,
    description: `Arguments: ${JSON.stringify(tc.arguments)}`,
    status: tc.status === 'success' ? 'completed' as const : tc.status === 'error' ? 'error' as const : 'running' as const,
    result: tc.status === 'success' ? String(tc.result) : tc.status === 'error' ? tc.error : undefined,
    emoji: getToolEmoji(tc.tool_name),
  }));
}

function buildMemoryAddons(memoryTraces: MemoryTrace[]): MessageAddon[] {
  if (memoryTraces.length === 0) return [];
  return [{
    id: 'memory-traces',
    type: 'memory',
    position: 'inline',
    memoryData: memoryTraces.map((trace) => ({
      id: trace.id as string,
      content: trace.content,
      relevance: trace.relevance,
    })),
  }];
}

const AssistantMessage: React.FC<AssistantMessageProps> = ({ conversationId, messageId, streaming, onBranchSwitch, onRetry, className = '' }) => {
  // Streaming path
  const streamingSelector = useMemo(() => selectConversationStreamingMessage(conversationId), [conversationId]);
  const streamingMessage = useChatStore(streamingSelector);

  // Completed path
  const completedMessage = useChatStore((state) => {
    if (streaming || !conversationId || !messageId) return undefined;
    return state.conversations.get(conversationId)?.messages.get(messageId);
  });

  const message = streaming ? streamingMessage : completedMessage;
  if (!message) return null;

  const toolDetails = buildToolDetails(message.tool_calls);
  const addons = buildMemoryAddons(message.memory_traces);

  return (
    <div className={`flex flex-col items-start gap-2 ${className}`}>
      <ChatBubble
        type="assistant"
        content={streaming ? '' : message.content}
        state={streaming ? 'streaming' : message.status === 'error' ? 'error' : 'completed'}
        timestamp={new Date(message.created_at)}
        streamingText={streaming ? message.content : undefined}
        addons={addons}
        toolDetails={toolDetails}
        messageId={streaming ? undefined : messageId}
        conversationId={conversationId || undefined}
        onBranchSwitch={onBranchSwitch}
        onRetry={onRetry}
        thinkingEntries={message.thinking}
        reasoningSteps={message.reasoning_steps}
      />
    </div>
  );
};

export default AssistantMessage;
