import { useEffect } from 'react';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { useAsync } from './useAsync';
import { useSync } from './useSync';

export function useMessages(conversationId: string | null) {
  const { messages, setMessages, addMessage, clearMessages } = useMessageContext();

  // Use sync hook for multi-device sync
  const { isSyncing, lastSyncTime, syncError, syncNow } = useSync(conversationId);

  // Fetch messages with loading and error handling
  const {
    loading,
    error,
    execute: fetchMessages,
  } = useAsync(
    async (id: string) => api.getMessages(id),
    {
      onSuccess: (data) => setMessages(data),
      errorMessage: 'Failed to fetch messages',
    }
  );

  // Send message with separate loading state
  const {
    loading: sending,
    execute: executeSendMessage,
  } = useAsync(
    async (id: string, content: string) => api.sendMessage(id, { contents: content }),
    {
      onSuccess: (newMessage) => addMessage(newMessage),
      errorMessage: 'Failed to send message',
    }
  );

  useEffect(() => {
    if (!conversationId) {
      clearMessages();
      return;
    }

    fetchMessages(conversationId);
  }, [conversationId, fetchMessages, clearMessages]);

  const sendMessage = async (content: string): Promise<boolean> => {
    if (!conversationId || !content.trim()) return false;

    const result = await executeSendMessage(conversationId, content);
    return result !== null;
  };

  return {
    messages,
    loading,
    error,
    sending,
    sendMessage,
    // Sync state
    isSyncing,
    lastSyncTime,
    syncError,
    syncNow,
  };
}
