import { useEffect, useCallback, useState } from 'react';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { useAsync } from './useAsync';
import { useSync } from './useSync';
import { Message } from '../types/models';

// Generate a temporary local ID for optimistic updates
function generateLocalId(): string {
  return `local_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

export function useMessages(conversationId: string | null) {
  const { messages, setMessages, addMessage, updateMessage, clearMessages } = useMessageContext();
  const [sending, setSending] = useState(false);

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
      onSuccess: (data) => {
        // Use setMessages when we have no local/pending messages (e.g., initial load)
        // This ensures we don't lose optimistic messages that were added during the fetch
        setMessages((prevMessages: Message[]) => {
          // Keep any pending/local messages that aren't yet on the server
          const pendingMessages = prevMessages.filter(
            (m) => m.sync_status === 'pending' || m.sync_status === 'conflict'
          );

          // If no pending messages, just use server data
          if (pendingMessages.length === 0) {
            return data;
          }

          // Merge: server messages + pending messages (avoiding duplicates)
          const serverIds = new Set(data.map((m) => m.id));
          const uniquePending = pendingMessages.filter((m) => !serverIds.has(m.id));

          // Sort by sequence_number, with pending messages at the end
          return [...data, ...uniquePending].sort((a, b) => {
            if (a.sequence_number === -1) return 1;
            if (b.sequence_number === -1) return -1;
            return a.sequence_number - b.sequence_number;
          });
        });
      },
      errorMessage: 'Failed to fetch messages',
    }
  );

  useEffect(() => {
    if (!conversationId) {
      clearMessages();
      return;
    }

    // Clear messages immediately when switching conversations to prevent message leakage
    // The new messages will be loaded by fetchMessages
    clearMessages();
    fetchMessages(conversationId);
  }, [conversationId, fetchMessages, clearMessages]);

  // Optimistic message sending
  const sendMessage = useCallback(async (content: string): Promise<boolean> => {
    if (!conversationId || !content.trim()) return false;

    const localId = generateLocalId();

    // Create optimistic message that appears immediately
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

    // Add message to UI immediately (optimistic update)
    addMessage(optimisticMessage);
    setSending(true);

    try {
      // Send to server
      const serverMessage = await api.sendMessage(conversationId, { contents: content });

      // Replace optimistic message with server response
      updateMessage(localId, {
        ...serverMessage,
        local_id: localId,
        sync_status: 'synced',
      });

      return true;
    } catch (err) {
      // Mark message as failed but keep it visible
      updateMessage(localId, {
        sync_status: 'conflict',
      });
      return false;
    } finally {
      setSending(false);
    }
  }, [conversationId, addMessage, updateMessage]);

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
