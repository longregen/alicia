import React, { useEffect } from 'react';
import ChatWindow from './ChatWindow';
import { useConversationStore } from '../../stores/conversationStore';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import {
  createMessageId,
  createConversationId,
  MessageStatus,
  NormalizedMessage,
} from '../../types/streaming';
import { Message } from '../../types/models';

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
  conversationId: string | null;
  syncError?: string | null;
}

/**
 * Convert legacy Message to new streaming Message format
 */
function convertToStreamingMessage(legacyMessage: Message): NormalizedMessage {
  return {
    id: createMessageId(legacyMessage.id),
    conversationId: createConversationId(legacyMessage.conversation_id),
    role: legacyMessage.role,
    content: legacyMessage.contents,
    status: MessageStatus.Complete, // Legacy messages are always complete
    createdAt: new Date(legacyMessage.created_at),
    sentenceIds: [],
    toolCallIds: [],
    memoryTraceIds: [],
    sync_status: legacyMessage.sync_status,
  };
}

const ChatWindowBridge: React.FC<ChatWindowBridgeProps> = ({
  messages,
  loading,
  sending,
  onSendMessage,
  conversationId,
  syncError = null,
}) => {
  const mergeMessages = useConversationStore((state) => state.mergeMessages);
  const setConnectionStatus = useConnectionStore((state) => state.setConnectionStatus);
  const setError = useConnectionStore((state) => state.setError);

  // Synchronize conversationId and messages to conversationStore
  useEffect(() => {
    if (conversationId && messages) {
      const streamingMessages = messages.map(convertToStreamingMessage);
      mergeMessages(createConversationId(conversationId), streamingMessages);
    }
  }, [conversationId, messages, mergeMessages]);

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
      conversationId={conversationId}
      useSileroVAD={false}
      showControls={false} // Disable controls in bridge mode since stop/regenerate not supported
    />
  );
};

export default ChatWindowBridge;
