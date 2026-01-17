import React, { useEffect } from 'react';
import ChatWindow from './ChatWindow';
import { useConversationStore } from '../../stores/conversationStore';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import {
  createMessageId,
  createConversationId,
  createToolCallId,
  createMemoryTraceId,
  MessageStatus,
  NormalizedMessage,
  ToolCall,
  MemoryTrace,
} from '../../types/streaming';
import { Message, ToolUseResponse, MemoryUsageResponse } from '../../types/models';

/**
 * ChatWindowBridge component.
 *
 * Bridges the legacy prop-based data flow from App.tsx to the new
 * organisms/ChatWindow component that uses Zustand stores.
 *
 * This allows gradual migration without rewriting App.tsx's data flow.
 */

export interface ChatWindowBridgeProps {
  // Legacy ChatWindow props from App.tsx
  messages: Message[];
  loading: boolean;
  sending: boolean;
  onSendMessage: (content: string) => void;
  onArchive?: () => void;
  onDelete?: () => void;
  conversationId: string | null;
  syncError?: string | null;
}

/**
 * Convert REST API MemoryUsageResponse to store MemoryTrace format
 */
function convertToMemoryTrace(memoryUsage: MemoryUsageResponse): MemoryTrace {
  return {
    id: createMemoryTraceId(memoryUsage.id),
    messageId: createMessageId(memoryUsage.message_id),
    content: memoryUsage.memory_content || '',
    relevance: memoryUsage.similarity_score,
    source: memoryUsage.memory_id,
  };
}

/**
 * Convert REST API ToolUseResponse to store ToolCall format
 */
function convertToToolCall(toolUse: ToolUseResponse, messageId: string): ToolCall {
  const baseFields = {
    id: createToolCallId(toolUse.id),
    toolName: toolUse.tool_name,
    arguments: toolUse.arguments || {},
    messageId: createMessageId(messageId),
    startTimeMs: new Date(toolUse.created_at).getTime(),
  };

  if (toolUse.status === 'success') {
    return {
      ...baseFields,
      status: 'success',
      endTimeMs: toolUse.completed_at ? new Date(toolUse.completed_at).getTime() : Date.now(),
      resultContent: typeof toolUse.result === 'string' ? toolUse.result : JSON.stringify(toolUse.result),
    };
  } else if (toolUse.status === 'error') {
    return {
      ...baseFields,
      status: 'error',
      endTimeMs: toolUse.completed_at ? new Date(toolUse.completed_at).getTime() : Date.now(),
      error: toolUse.error_message || 'Unknown error',
    };
  } else if (toolUse.status === 'running') {
    return {
      ...baseFields,
      status: 'executing',
    };
  } else {
    // pending or cancelled
    return {
      ...baseFields,
      status: 'pending',
    };
  }
}

/**
 * Convert legacy Message to new streaming Message format
 */
function convertToStreamingMessage(legacyMessage: Message): NormalizedMessage {
  // Extract tool call IDs if present
  const toolCallIds = (legacyMessage.tool_uses || []).map(tu => createToolCallId(tu.id));
  // Extract memory trace IDs if present
  const memoryTraceIds = (legacyMessage.memory_usages || []).map(mu => createMemoryTraceId(mu.id));

  return {
    id: createMessageId(legacyMessage.id),
    conversationId: createConversationId(legacyMessage.conversation_id),
    role: legacyMessage.role,
    content: legacyMessage.contents,
    status: MessageStatus.Complete, // Legacy messages are always complete
    createdAt: new Date(legacyMessage.created_at),
    sentenceIds: [],
    toolCallIds,
    memoryTraceIds,
    sync_status: legacyMessage.sync_status,
    local_id: legacyMessage.local_id, // Required for deduplication when server responds
  };
}

const ChatWindowBridge: React.FC<ChatWindowBridgeProps> = ({
  messages,
  loading,
  sending,
  onSendMessage,
  onArchive,
  onDelete,
  conversationId,
  syncError = null,
}) => {
  const mergeMessages = useConversationStore((state) => state.mergeMessages);
  const addToolCall = useConversationStore((state) => state.addToolCall);
  const addMemoryTrace = useConversationStore((state) => state.addMemoryTrace);
  const setCurrentConversationId = useConversationStore((state) => state.setCurrentConversationId);
  const clearConversation = useConversationStore((state) => state.clearConversation);
  const setConnectionStatus = useConnectionStore((state) => state.setConnectionStatus);
  const setError = useConnectionStore((state) => state.setError);

  // Clear store and set current conversation when switching conversations
  useEffect(() => {
    if (conversationId) {
      clearConversation();
      setCurrentConversationId(createConversationId(conversationId));
    } else {
      clearConversation();
      setCurrentConversationId(null);
    }
  }, [conversationId, clearConversation, setCurrentConversationId]);

  // Synchronize messages to conversationStore
  useEffect(() => {
    if (conversationId && messages) {
      const streamingMessages = messages.map(convertToStreamingMessage);
      mergeMessages(createConversationId(conversationId), streamingMessages);

      // Add tool calls to the store (after mergeMessages so messages exist)
      messages.forEach(msg => {
        if (msg.tool_uses) {
          msg.tool_uses.forEach(toolUse => {
            addToolCall(convertToToolCall(toolUse, msg.id));
          });
        }
      });

      // Add memory traces to the store (after mergeMessages so messages exist)
      messages.forEach(msg => {
        if (msg.memory_usages) {
          msg.memory_usages.forEach(memoryUsage => {
            addMemoryTrace(convertToMemoryTrace(memoryUsage));
          });
        }
      });
    }
  }, [conversationId, messages, mergeMessages, addToolCall, addMemoryTrace]);

  // Synchronize connection state to connectionStore
  // Skip in E2E tests where connection is mocked
  useEffect(() => {
    // Don't override connection status in E2E tests
    if (typeof window !== 'undefined' && (window as unknown as { __E2E_CONNECTION_MOCK__?: unknown }).__E2E_CONNECTION_MOCK__) {
      return;
    }

    if (loading || sending) {
      setConnectionStatus(ConnectionStatus.Connecting);
    } else if (syncError) {
      setConnectionStatus(ConnectionStatus.Error);
      setError(syncError);
    } else if (conversationId) {
      setConnectionStatus(ConnectionStatus.Connected);
      setError(null);
    } else {
      setConnectionStatus(ConnectionStatus.Disconnected);
    }
  }, [loading, sending, syncError, conversationId, setConnectionStatus, setError]);

  // Handle message sending - bridge to legacy callback
  // Legacy handler doesn't support voice flag - parameter prefixed with underscore
  const handleSendMessage = (message: string, _isVoice: boolean) => {
    onSendMessage(message);
  };

  return (
    <ChatWindow
      onSendMessage={handleSendMessage}
      onArchive={onArchive}
      onDelete={onDelete}
      conversationId={conversationId}
      showControls={false} // Disable controls in bridge mode since stop/regenerate not supported
    />
  );
};

export default ChatWindowBridge;
