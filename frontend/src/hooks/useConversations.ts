import { useState, useEffect, useCallback } from 'react';
import { Conversation } from '../types/models';
import { api } from '../services/api';
import { useAsync } from './useAsync';

export function useConversations() {
  const [conversations, setConversations] = useState<Conversation[]>([]);

  // Stable callbacks to prevent infinite re-renders
  const handleFetchSuccess = useCallback((data: Conversation[]) => {
    setConversations(data);
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
  }, [fetchConversations]);

  return {
    conversations,
    loading,
    error,
    createConversation,
    deleteConversation,
    updateConversation,
    refetch: fetchConversations,
  };
}
