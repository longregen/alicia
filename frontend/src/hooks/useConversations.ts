import { useState, useEffect } from 'react';
import { Conversation } from '../types/models';
import { api } from '../services/api';
import { useAsync } from './useAsync';

export function useConversations() {
  const [conversations, setConversations] = useState<Conversation[]>([]);

  // Fetch conversations with loading and error handling
  const {
    loading,
    error,
    execute: fetchConversations,
  } = useAsync(
    async () => api.getConversations(),
    {
      onSuccess: (data) => setConversations(data),
      errorMessage: 'Failed to fetch conversations',
    }
  );

  // Create conversation
  const {
    execute: createConversation,
  } = useAsync(
    async (title?: string) => api.createConversation({ title }),
    {
      onSuccess: (newConversation) => setConversations(prev => [newConversation, ...prev]),
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

  // Update conversation in local state
  const updateConversation = (updatedConversation: Conversation) => {
    setConversations(prev =>
      prev.map(c => c.id === updatedConversation.id ? updatedConversation : c)
    );
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
