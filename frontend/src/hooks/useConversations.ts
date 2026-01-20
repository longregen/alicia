import { useState, useEffect, useCallback } from 'react';
import { Conversation } from '../types/models';
import { api } from '../services/api';
import { useAsync } from './useAsync';
import { registerConversationUpdateHandler } from '../contexts/WebSocketContext';
import { ConversationUpdate } from '../types/protocol';

export function useConversations() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [hasFetched, setHasFetched] = useState(false);

  // Stable callbacks to prevent infinite re-renders
  const handleFetchSuccess = useCallback((data: Conversation[]) => {
    setConversations(data);
    setHasFetched(true);
  }, []);

  const handleCreateSuccess = useCallback((newConversation: Conversation) => {
    setConversations(prev => [newConversation, ...prev]);
  }, []);

  // Fetch conversations with loading and error handling
  const {
    loading,
    error,
    execute: fetchConversations,
  } = useAsync(
    async () => api.getConversations(),
    {
      onSuccess: handleFetchSuccess,
      errorMessage: 'Failed to fetch conversations',
    }
  );

  // Create conversation
  const {
    execute: createConversation,
  } = useAsync(
    async (title?: string) => api.createConversation({ title }),
    {
      onSuccess: handleCreateSuccess,
      errorMessage: 'Failed to create conversation',
    }
  );

  // Delete conversation
  const {
    execute: executeDelete,
  } = useAsync(
    async (id: string) => api.deleteConversation(id),
    {
      errorMessage: 'Failed to delete conversation',
    }
  );

  const deleteConversation = async (id: string) => {
    const result = await executeDelete(id);
    if (result !== null) {
      setConversations(prev => prev.filter(c => c.id !== id));
    }
  };

  // Update conversation
  const {
    execute: executeUpdate,
  } = useAsync(
    async (id: string, data: Partial<Conversation>) => api.updateConversation(id, data),
    {
      errorMessage: 'Failed to update conversation',
    }
  );

  const updateConversation = async (id: string, data: Partial<Conversation>) => {
    const result = await executeUpdate(id, data);
    if (result) {
      setConversations(prev =>
        prev.map(c => c.id === id ? result : c)
      );
    }
  };

  useEffect(() => {
    fetchConversations();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Only fetch on mount

  // Listen for real-time conversation updates via WebSocket
  useEffect(() => {
    const handleConversationUpdate = (update: ConversationUpdate) => {
      setConversations(prev =>
        prev.map(c => {
          if (c.id === update.conversationId) {
            return {
              ...c,
              title: update.title ?? c.title,
              status: (update.status as Conversation['status']) ?? c.status,
              updated_at: update.updatedAt,
            };
          }
          return c;
        })
      );
    };

    const unregister = registerConversationUpdateHandler(handleConversationUpdate);
    return () => unregister();
  }, []);

  return {
    conversations,
    loading,
    hasFetched,
    error,
    createConversation,
    deleteConversation,
    updateConversation,
    refetch: fetchConversations,
  };
}
