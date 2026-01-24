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
  tags: string[];
  pinned: boolean;
  archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface MemoryListResponse {
  memories: MemoryAPIResponse[];
  total: number;
}


// Outside hook: no dependencies on hook state
function apiToStoreMemory(apiMemory: MemoryAPIResponse): Memory {
  const category: MemoryCategory = apiMemory.tags?.[0] as MemoryCategory || 'fact';

  return {
    id: apiMemory.id,
    content: apiMemory.content,
    category,
    tags: apiMemory.tags || [],
    importance: apiMemory.importance || 0.5,
    createdAt: new Date(apiMemory.created_at).getTime(),
    updatedAt: new Date(apiMemory.updated_at).getTime(),
    pinned: apiMemory.pinned || false,
    archived: apiMemory.archived || false,
    usageCount: 0,
  };
}

export function useMemories() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isFetching, setIsFetching] = useState(false);

  const setMemory = useMemoryStore((state) => state.setMemory);
  const updateMemory = useMemoryStore((state) => state.updateMemory);
  const deleteMemory = useMemoryStore((state) => state.deleteMemory);
  const pinMemory = useMemoryStore((state) => state.pinMemory);
  const archiveMemory = useMemoryStore((state) => state.archiveMemory);
  const clearMemories = useMemoryStore((state) => state.clearMemories);

  // Subscribe to raw object and compute locally to avoid infinite loops from derived selectors
  const rawMemories = useMemoryStore((state) => state.memories);

  const memories = useMemo(
    () =>
      Object.values(rawMemories)
        .filter((m) => !m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt),
    [rawMemories]
  );

  const deletedMemories = useMemo(
    () =>
      Object.values(rawMemories)
        .filter((m) => m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt),
    [rawMemories]
  );

  const fetchMemories = useCallback(async () => {
    setIsFetching(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/memories?limit=500');
      if (!response.ok) {
        throw new Error(`Failed to fetch memories: ${response.status}`);
      }

      const data: MemoryListResponse = await response.json();

      clearMemories();
      if (data.memories) {
        data.memories.forEach((apiMemory) => {
          const memory = apiToStoreMemory(apiMemory);
          setMemory(memory);
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch memories';
      setError(message);
      console.error('Fetch memories error:', err);
    } finally {
      setIsFetching(false);
    }
  }, [setMemory, clearMemories]);

  useEffect(() => {
    fetchMemories();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const create = useCallback(async (content: string, category?: MemoryCategory) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/memories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create memory: ${response.status}`);
      }

      const apiMemory: MemoryAPIResponse = await response.json();
      const memory = apiToStoreMemory(apiMemory);

      setMemory(memory);

      if (category) {
        try {
          await fetch(`/api/v1/memories/${apiMemory.id}/tags`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tag: category }),
          });
        } catch {
          console.warn('Failed to add category tag to memory');
        }
      }

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

      if (tags) {
        // Simplified: in production you'd manage tags properly (add/remove delta)
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

  const remove = useCallback(async (id: string, reason?: MemoryDeletionReason) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/memories/${id}/archive`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      });

      if (!response.ok) {
        throw new Error(`Failed to delete memory: ${response.status}`);
      }

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

  const setRating = useCallback(async (id: string, stars: number) => {
    setIsLoading(true);
    setError(null);

    // Stars (1-5) -> importance (0.2-1.0)
    const importance = stars <= 0 ? 0.5 : Math.min(1.0, Math.max(0.2, stars * 0.2));

    try {
      const response = await fetch(`/api/v1/memories/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ importance }),
      });

      if (!response.ok) {
        throw new Error(`Failed to set memory importance: ${response.status}`);
      }

      await response.json();

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

  const setCategory = useCallback(async (id: string, category: MemoryCategory) => {
    setIsLoading(true);
    setError(null);

    try {
      const memory = rawMemories[id];
      const oldCategory = memory?.category;

      const addResponse = await fetch(`/api/v1/memories/${id}/tags`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tag: category }),
      });

      if (!addResponse.ok) {
        throw new Error(`Failed to add category tag: ${addResponse.status}`);
      }

      if (oldCategory && oldCategory !== category) {
        await fetch(`/api/v1/memories/${id}/tags/${oldCategory}`, {
          method: 'DELETE',
        });
      }

      updateMemory(id, { category, tags: [category] });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to set category';
      setError(message);
      console.error('Set category error:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [updateMemory, rawMemories]);

  const search = useCallback((query: string): Memory[] => {
    if (!query.trim()) {
      return memories;
    }
    const lowerQuery = query.toLowerCase();
    return memories.filter((m) => m.content.toLowerCase().includes(lowerQuery));
  }, [memories]);

  const filterByCategory = useCallback((category: MemoryCategory): Memory[] => {
    return memories.filter((m) => m.category === category);
  }, [memories]);

  return {
    memories,
    deletedMemories,
    isLoading,
    isFetching,
    error,
    create,
    update,
    remove,
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
