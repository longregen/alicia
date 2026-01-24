import { useState, useEffect, useCallback } from 'react';
import { Conversation } from '../types/models';
import { api } from '../services/api';
import { useAsync } from './useAsync';
import { registerTitleUpdateHandler } from '../contexts/WebSocketContext';

export function useConversations() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [hasFetched, setHasFetched] = useState(false);

  const handleFetchSuccess = useCallback((data: Conversation[]) => {
    setConversations(data);
    setHasFetched(true);
  }, []);

  const handleCreateSuccess = useCallback((newConversation: Conversation) => {
    setConversations(prev => [newConversation, ...prev]);
  }, []);

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

  const {
    execute: createConversation,
  } = useAsync(
    async (title?: string) => api.createConversation({ title }),
    {
      onSuccess: handleCreateSuccess,
      errorMessage: 'Failed to create conversation',
    }
  );

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
  }, []);

  useEffect(() => {
    return registerTitleUpdateHandler((conversationId, title) => {
      setConversations(prev =>
        prev.map(c => c.id === conversationId ? { ...c, title } : c)
      );
    });
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
