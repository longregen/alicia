import { useEffect, useCallback, useState, useRef } from 'react';
import { api } from '../services/api';
import { useMessageContext } from '../contexts/MessageContext';
import { useAsync } from './useAsync';
import { useSync } from './useSync';
import { Message } from '../types/models';
import { messageRepository } from '../db/repository';
import { useBranchStore } from '../stores/branchStore';

// Generate a temporary local ID for optimistic updates
function generateLocalId(): string {
  return `local_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

export function useMessages(conversationId: string | null) {
  const [messages, setMessages] = useState<Message[]>([]);
  // Store server messages with their related entities (tool_uses, memory_usages)
  // These are not stored in SQLite, so we keep them in memory
  // Using a ref to avoid circular dependency in refreshMessages
  const serverMessagesCacheRef = useRef<Map<string, Message>>(new Map());
  // Ref to access current conversationId in callbacks without re-creating them
  const conversationIdRef = useRef<string | null>(conversationId);
  conversationIdRef.current = conversationId;

  const { clearMessages: clearProtocolMessages } = useMessageContext();
  const [sending, setSending] = useState(false);
  const [refreshCounter, setRefreshCounter] = useState(0);

  // Branch store for initializing sibling data from fetched messages
  const { initializeFromMessages, clearBranches } = useBranchStore();

  // Callback to refresh UI when WebSocket sync occurs
  const handleSyncComplete = useCallback(() => {
    setRefreshCounter(prev => prev + 1);
  }, []);

  // Use sync hook for multi-device sync
  const { isSyncing, lastSyncTime, syncError, syncNow } = useSync(conversationId, {
    onSync: handleSyncComplete,
    onMessage: handleSyncComplete, // Also refresh UI when new messages arrive via WebSocket
  });

  // Refresh messages from SQLite and merge with server cache for related entities
  const refreshMessages = useCallback(() => {
    if (!conversationId) {
      setMessages([]);
      return;
    }
    const dbMessages = messageRepository.findByConversation(conversationId);

    // Merge SQLite messages with cached server data for related entities
    const mergedMessages = dbMessages.map(dbMsg => {
      const cachedMsg = serverMessagesCacheRef.current.get(dbMsg.id);
      if (cachedMsg) {
        // Preserve tool_uses and memory_usages from server response
        return {
          ...dbMsg,
          tool_uses: cachedMsg.tool_uses,
          memory_usages: cachedMsg.memory_usages,
        };
      }
      return dbMsg;
    });

    setMessages(mergedMessages);
  }, [conversationId]);

  // Wrap onSuccess callback to merge server messages into SQLite
  const handleFetchSuccess = useCallback((data: Message[]) => {
    // Cache server messages with their related entities
    const newCache = new Map<string, Message>();
    data.forEach(msg => {
      newCache.set(msg.id, msg);
    });
    serverMessagesCacheRef.current = newCache;

    // Save server messages to SQLite (core fields only)
    data.forEach(msg => {
      messageRepository.upsert({
        ...msg,
        sync_status: 'synced',
      });
    });

    // Initialize branch store from fetched messages.
    // The backend now returns all siblings at each branch point,
    // so initializeFromMessages will detect them by grouping on previous_id.
    const convId = conversationIdRef.current;
    if (convId) {
      initializeFromMessages(convId, data);
    }

    // Refresh from SQLite to get merged state
    setRefreshCounter(prev => prev + 1);
  }, [initializeFromMessages]);

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
      serverMessagesCacheRef.current = new Map();
      clearProtocolMessages();
      clearBranches();
      return;
    }

    // Clear cache and branch state when switching conversations
    serverMessagesCacheRef.current = new Map();
    clearBranches();

    // Load from SQLite immediately
    refreshMessages();

    // Then fetch from server to sync
    fetchMessages(conversationId);
  }, [conversationId, fetchMessages, clearProtocolMessages, clearBranches, refreshMessages]);

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
      const serverMessage = await api.sendMessage(conversationId, { contents: content, local_id: localId });

      // Replace the local message ID with the server-assigned ID
      // This prevents duplicates when WebSocket broadcasts arrive with the server ID
      messageRepository.replaceId(localId, serverMessage.id, {
        ...serverMessage,
        local_id: localId,
        server_id: serverMessage.id,
        sync_status: 'synced',
      });

      setRefreshCounter(prev => prev + 1);
      return true;
    } catch {
      // Keep message as pending for retry via WebSocket sync
      // WebSocket reconnection will retry pending messages
      messageRepository.update(localId, {
        sync_status: 'pending',
      });
      messageRepository.incrementRetryCount(localId);
      setRefreshCounter(prev => prev + 1);
      return false;
    } finally {
      setSending(false);
    }
  }, [conversationId]);

  // Refetch messages from server - used after branch switching
  const refetch = useCallback(() => {
    if (conversationId) {
      fetchMessages(conversationId);
    }
  }, [conversationId, fetchMessages]);

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
    // Manual refresh (from SQLite only)
    refresh: () => setRefreshCounter(prev => prev + 1),
    // Refetch from server - use after branch switching
    refetch,
  };
}
