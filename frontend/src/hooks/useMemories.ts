import { useCallback, useState, useEffect, useMemo } from 'react';
import {
  useMemoryStore,
  type Memory,
  type MemoryCategory,
} from '../stores/memoryStore';

/** Reasons for deleting a memory */
export type MemoryDeletionReason = 'wrong' | 'useless' | 'old' | 'duplicate' | 'other';

export interface MemoryAPIResponse {
  id: string;
  content: string;
  importance: number;
  confidence: number;
  user_rating?: number;
  tags: string[];
  source_type?: string;
  pinned: boolean;
  archived: boolean;
  created_at: number;
  updated_at: number;
}

export interface MemoryListResponse {
  memories: MemoryAPIResponse[];
  total: number;
}

export interface SearchResultResponse {
  memory: MemoryAPIResponse;
  similarity: number;
}

export interface SearchResultsResponse {
  results: SearchResultResponse[];
  total: number;
}

/**
 * Convert API response to store memory format.
 * Moved outside hook since it has no dependencies on hook state.
 */
function apiToStoreMemory(apiMemory: MemoryAPIResponse): Memory {
  // Map tags to category (use first tag or default to 'fact')
  const category: MemoryCategory = apiMemory.tags[0] as MemoryCategory || 'fact';

  return {
    id: apiMemory.id,
    content: apiMemory.content,
    category,
    tags: apiMemory.tags || [],
    importance: apiMemory.importance || 0.5,
    createdAt: apiMemory.created_at * 1000, // Convert to milliseconds
    updatedAt: apiMemory.updated_at * 1000,
    pinned: apiMemory.pinned || false,
    archived: apiMemory.archived || false,
    usageCount: 0,
  };
}

/**
 * Hook for managing global memories with API integration.
 * Wraps the memoryStore with CRUD operations and server synchronization.
 *
 * @example
 * ```tsx
 * function MemoryManager() {
 *   const { memories, create, update, remove, pin, archive, search, isLoading, error } = useMemories();
 *
 *   const handleCreate = async () => {
 *     await create('User prefers dark mode', 'preference');
 *   };
 *
 *   return <div>...</div>;
 * }
 * ```
 */
export function useMemories() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isFetching, setIsFetching] = useState(false);

  // Store actions
  const setMemory = useMemoryStore((state) => state.setMemory);
  const updateMemory = useMemoryStore((state) => state.updateMemory);
  const deleteMemory = useMemoryStore((state) => state.deleteMemory);
  const pinMemory = useMemoryStore((state) => state.pinMemory);
  const archiveMemory = useMemoryStore((state) => state.archiveMemory);
  const clearMemories = useMemoryStore((state) => state.clearMemories);

  // Subscribe to raw memories object and compute locally to avoid infinite loops
  const rawMemories = useMemoryStore((state) => state.memories);

  // Compute active memories (not archived)
  const memories = useMemo(
    () =>
      Object.values(rawMemories)
        .filter((m) => !m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt),
    [rawMemories]
  );

  // Compute deleted/archived memories
  const deletedMemories = useMemo(
    () =>
      Object.values(rawMemories)
        .filter((m) => m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt),
    [rawMemories]
  );

  // Fetch all memories from server
  const fetchMemories = useCallback(async () => {
    setIsFetching(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/memories?limit=500');
      if (!response.ok) {
        throw new Error(`Failed to fetch memories: ${response.status}`);
      }

      const data: MemoryListResponse = await response.json();

      // Clear and repopulate store with correct IDs from API
      clearMemories();
      data.memories.forEach((apiMemory) => {
        const memory = apiToStoreMemory(apiMemory);
        setMemory(memory);
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch memories';
      setError(message);
      console.error('Fetch memories error:', err);
    } finally {
      setIsFetching(false);
    }
  }, [setMemory, clearMemories]);

  // Load memories on mount
  useEffect(() => {
    fetchMemories();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Only fetch on mount

  // Create new memory
  const create = useCallback(async (content: string, category: MemoryCategory) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/memories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content,
          tags: [category],
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create memory: ${response.status}`);
      }

      const apiMemory: MemoryAPIResponse = await response.json();
      const memory = apiToStoreMemory(apiMemory);

      // Use setMemory to preserve the API-assigned ID
      setMemory(memory);

      return apiMemory.id;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create memory';
      setError(message);
      console.error('Create memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [setMemory]);

  // Update existing memory
  const update = useCallback(async (id: string, content: string, category?: MemoryCategory) => {
    setIsLoading(true);
    setError(null);

    try {
      const body: { content: string } = { content };
      const tags = category ? [category] : undefined;

      const response = await fetch(`/api/v1/memories/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`Failed to update memory: ${response.status}`);
      }

      await response.json();

      // Update tags if category changed
      if (tags) {
        // This is a simplified approach - in production you'd manage tags properly
        await fetch(`/api/v1/memories/${id}/tags`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ tag: category }),
        });
      }

      updateMemory(id, {
        content,
        category,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update memory';
      setError(message);
      console.error('Update memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [updateMemory]);

  // Soft delete memory (archive with optional reason)
  const remove = useCallback(async (id: string, reason?: MemoryDeletionReason) => {
    setIsLoading(true);
    setError(null);

    try {
      // Archive the memory (soft delete)
      const response = await fetch(`/api/v1/memories/${id}/archive`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      });

      if (!response.ok) {
        throw new Error(`Failed to delete memory: ${response.status}`);
      }

      // Update local store to mark as archived
      archiveMemory(id);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete memory';
      setError(message);
      console.error('Delete memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [archiveMemory]);

  // Restore archived memory
  const restore = useCallback(async (id: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/memories/${id}/restore`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(`Failed to restore memory: ${response.status}`);
      }

      // Update local store to unarchive
      updateMemory(id, { archived: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to restore memory';
      setError(message);
      console.error('Restore memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [updateMemory]);

  // Permanently delete memory
  const permanentDelete = useCallback(async (id: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/memories/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error(`Failed to permanently delete memory: ${response.status}`);
      }

      deleteMemory(id);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to permanently delete memory';
      setError(message);
      console.error('Permanent delete memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [deleteMemory]);

  // Pin/unpin memory
  const pin = useCallback(async (id: string, pinned: boolean) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/memories/${id}/pin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pinned }),
      });

      if (!response.ok) {
        throw new Error(`Failed to pin memory: ${response.status}`);
      }

      await response.json();

      // Update local store
      pinMemory(id, pinned);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to pin memory';
      setError(message);
      console.error('Pin memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [pinMemory]);

  // Archive memory
  const archive = useCallback(async (id: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/memories/${id}/archive`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(`Failed to archive memory: ${response.status}`);
      }

      await response.json();

      // Update local store
      archiveMemory(id);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to archive memory';
      setError(message);
      console.error('Archive memory error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [archiveMemory]);

  // Set importance rating (1-5 stars mapped to 0.2-1.0 importance)
  const setRating = useCallback(async (id: string, stars: number) => {
    setIsLoading(true);
    setError(null);

    // Convert stars (1-5) to importance (0.2-1.0)
    const importance = stars <= 0 ? 0.5 : Math.min(1.0, Math.max(0.2, stars * 0.2));

    try {
      const response = await fetch(`/api/v1/memories/${id}/importance`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ importance }),
      });

      if (!response.ok) {
        throw new Error(`Failed to set memory importance: ${response.status}`);
      }

      await response.json();

      // Update local store
      updateMemory(id, { importance });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to set importance';
      setError(message);
      console.error('Set importance error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [updateMemory]);

  // Set category for a memory
  const setCategory = useCallback(async (id: string, category: MemoryCategory) => {
    setIsLoading(true);
    setError(null);

    try {
      // Update tags via API (category is stored as first tag)
      const response = await fetch(`/api/v1/memories/${id}/tags`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tags: [category] }),
      });

      if (!response.ok) {
        throw new Error(`Failed to set memory category: ${response.status}`);
      }

      // Update local store
      updateMemory(id, { category, tags: [category] });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to set category';
      setError(message);
      console.error('Set category error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [updateMemory]);

  // Search memories locally to avoid creating new references
  const search = useCallback((query: string): Memory[] => {
    if (!query.trim()) {
      return memories;
    }
    const lowerQuery = query.toLowerCase();
    return memories.filter((m) => m.content.toLowerCase().includes(lowerQuery));
  }, [memories]);

  // Filter by category locally to avoid creating new references
  const filterByCategory = useCallback((category: MemoryCategory): Memory[] => {
    return memories.filter((m) => m.category === category);
  }, [memories]);

  return {
    // State
    memories,
    deletedMemories,
    isLoading,
    isFetching,
    error,

    // Actions
    create,
    update,
    remove,
    restore,
    permanentDelete,
    pin,
    archive,
    setRating,
    setCategory,
    search,
    filterByCategory,
    refresh: fetchMemories,
  };
}
