import React, { useMemo } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import MemoryTraceAddon from '../atoms/MemoryTraceAddon';
import { useConversationStore, selectCurrentStreamingMessage, selectSentences } from '../../stores/conversationStore';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import type { MessageAddon, ToolData } from '../../types/components';

/**
 * StreamingMessage organism component.
 *
 * Displays currently streaming assistant responses with:
 * - Sentences as they arrive (from store)
 * - Typing cursor animation
 * - "Streaming" status indicator
 * - Tool calls in progress (running, completed, error states)
 * - Memory traces being used for context
 * - Data fetched from Zustand store's currentStreamingMessageId
 */

export interface StreamingMessageProps {
  className?: string;
}

const StreamingMessage: React.FC<StreamingMessageProps> = ({ className = '' }) => {
  const streamingMessage = useConversationStore(selectCurrentStreamingMessage);
  const sentencesMap = useConversationStore(selectSentences);
  const toolCallsMap = useConversationStore((state) => state.toolCalls);
  const memoryTracesMap = useConversationStore((state) => state.memoryTraces);

  // Memoize sentences derivation to avoid creating new arrays on every render
  const streamingText = useMemo(() => {
    if (!streamingMessage) return '';

    return streamingMessage.sentenceIds
      .map(id => sentencesMap[id])
      .filter(s => s && s.isComplete)
      .sort((a, b) => a.sequence - b.sequence)
      .map(s => s.content)
      .join(' ');
  }, [streamingMessage, sentencesMap]);

  // Memoize tool calls for this streaming message
  const toolCalls = useMemo(() => {
    if (!streamingMessage) return [];
    return streamingMessage.toolCallIds
      .map(id => toolCallsMap[id])
      .filter(Boolean);
  }, [streamingMessage, toolCallsMap]);

  // Memoize memory traces for this streaming message
  const memoryTraces = useMemo(() => {
    if (!streamingMessage) return [];
    return streamingMessage.memoryTraceIds
      .map(id => memoryTracesMap[id])
      .filter(Boolean)
      .sort((a, b) => b.relevance - a.relevance);
  }, [streamingMessage, memoryTracesMap]);

  if (!streamingMessage) {
    return null;
  }

  // Convert tool calls to ToolData format for ChatBubble
  const tools: ToolData[] = toolCalls.map(toolCall => ({
    id: toolCall.id,
    name: toolCall.toolName,
    description: `Arguments: ${JSON.stringify(toolCall.arguments)}`,
    status: toolCall.status === 'pending' ? 'running' :
            toolCall.status === 'executing' ? 'running' :
            toolCall.status === 'success' ? 'completed' : 'error',
    result: toolCall.status === 'success' ? toolCall.resultContent :
            toolCall.status === 'error' ? toolCall.error : undefined,
  }));

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
    <div className={`flex flex-col items-start gap-2 ${className}`}>
      <ChatBubble
        type={MESSAGE_TYPES.ASSISTANT}
        content=""
        state={MESSAGE_STATES.STREAMING}
        timestamp={streamingMessage.createdAt}
        streamingText={streamingText}
        addons={addons}
        tools={tools}
        showTyping={true}
      />
      {memoryTraces.length > 0 && (
        <MemoryTraceAddon traces={memoryTraces} />
      )}
    </div>
  );
};

export default StreamingMessage;
