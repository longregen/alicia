import { useEffect, useCallback, useState } from 'react';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { useAsync } from './useAsync';
import { useSync } from './useSync';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';

// Generate a temporary local ID for optimistic updates
function generateLocalId(): string {
  return `local_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

export function useMessages(conversationId: string | null) {
  const [messages, setMessages] = useState<Message[]>([]);
  const { clearMessages: clearProtocolMessages } = useMessageContext();
  const [sending, setSending] = useState(false);
  const [refreshCounter, setRefreshCounter] = useState(0);

  // Use sync hook for multi-device sync
  const { isSyncing, lastSyncTime, syncError, syncNow } = useSync(conversationId);

  // Refresh messages from SQLite
  const refreshMessages = useCallback(() => {
    if (!conversationId) {
      setMessages([]);
      return;
    }
    const dbMessages = messageRepository.findByConversation(conversationId);
    setMessages(dbMessages);
  }, [conversationId]);

  // Wrap onSuccess callback to merge server messages into SQLite
  const handleFetchSuccess = useCallback((data: Message[]) => {
    // Save server messages to SQLite
    data.forEach(msg => {
      messageRepository.upsert({
        ...msg,
        sync_status: 'synced',
      });
    });

    // Refresh from SQLite to get merged state
    setRefreshCounter(prev => prev + 1);
  }, []);

  // Fetch messages with loading and error handling
  const {
    loading,
    error,
    execute: fetchMessages,
  } = useAsync(
    async (id: string) => api.getMessages(id),
    {
      onSuccess: handleFetchSuccess,
      errorMessage: 'Failed to fetch messages',
    }
  );

  // Load messages from SQLite when conversation changes
  useEffect(() => {
    if (!conversationId) {
      setMessages([]);
      clearProtocolMessages();
      return;
    }

    // Load from SQLite immediately
    refreshMessages();

    // Then fetch from server to sync
    fetchMessages(conversationId);
  }, [conversationId, fetchMessages, clearProtocolMessages, refreshMessages]);

  // Refresh from SQLite when counter changes
  useEffect(() => {
    refreshMessages();
  }, [refreshCounter, refreshMessages]);

  // Optimistic message sending
  const sendMessage = useCallback(async (content: string): Promise<boolean> => {
    if (!conversationId || !content.trim()) return false;

    const localId = generateLocalId();

    // Create optimistic message
    const optimisticMessage: Message = {
      id: localId,
      conversation_id: conversationId,
      sequence_number: -1, // Will be updated when server responds
      role: 'user',
      contents: content.trim(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      local_id: localId,
      sync_status: 'pending',
    };

    // Save to SQLite immediately (optimistic update)
    messageRepository.insert(optimisticMessage);
    setRefreshCounter(prev => prev + 1);
    setSending(true);

    try {
      // Send to server
      const serverMessage = await api.sendMessage(conversationId, { contents: content });

      // Update SQLite with server response
      messageRepository.update(localId, {
        ...serverMessage,
        local_id: localId,
        sync_status: 'synced',
      });

      setRefreshCounter(prev => prev + 1);
      return true;
    } catch {
      // Mark message as failed but keep it visible
      messageRepository.update(localId, {
        sync_status: 'conflict',
      });
      setRefreshCounter(prev => prev + 1);
      return false;
    } finally {
      setSending(false);
    }
  }, [conversationId]);

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
    // Manual refresh
    refresh: () => setRefreshCounter(prev => prev + 1),
  };
}
