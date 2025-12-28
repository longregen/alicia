import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  useMemoryStore,
  selectAllMemories,
  selectArchivedMemories,
  selectMemoriesByCategory,
  selectRecentMemories,
  type MemoryCategory,
} from './memoryStore';

describe('memoryStore', () => {
  beforeEach(() => {
    useMemoryStore.getState().clearMemories();
    vi.stubGlobal('crypto', {
      randomUUID: () => 'test-uuid-' + Math.random(),
    });
  });

  describe('createMemory', () => {
    it('should create a memory with required fields', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');

      const state = useMemoryStore.getState();
      const memory = Object.values(state.memories)[0];
      expect(memory).toBeDefined();
      expect(memory.content).toBe('Test memory');
      expect(memory.category).toBe('fact');
      expect(memory.pinned).toBe(false);
      expect(memory.archived).toBe(false);
    });

    it('should create memory with pinned flag', () => {
      useMemoryStore.getState().createMemory('Important memory', 'preference', true);

      const state = useMemoryStore.getState();
      const memory = Object.values(state.memories)[0];
      expect(memory.pinned).toBe(true);
    });

    it('should set timestamps on creation', () => {
      const before = Date.now();
      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const after = Date.now();

      const state = useMemoryStore.getState();
      const memory = Object.values(state.memories)[0];
      expect(memory.createdAt).toBeGreaterThanOrEqual(before);
      expect(memory.createdAt).toBeLessThanOrEqual(after);
      expect(memory.updatedAt).toBe(memory.createdAt);
    });

    it('should handle all memory categories', () => {
      const categories: MemoryCategory[] = ['preference', 'fact', 'context', 'instruction'];

      categories.forEach((category) => {
        useMemoryStore.getState().createMemory(`Memory for ${category}`, category);
      });

      const state = useMemoryStore.getState();
      expect(Object.keys(state.memories)).toHaveLength(4);
    });
  });

  describe('updateMemory', () => {
    it('should update memory content', () => {
      useMemoryStore.getState().createMemory('Original content', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().updateMemory(memoryId, {
        content: 'Updated content',
      });

      const updatedMemory = useMemoryStore.getState().memories[memoryId];
      expect(updatedMemory.content).toBe('Updated content');
    });

    it('should update memory category', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().updateMemory(memoryId, {
        category: 'preference',
      });

      const updatedMemory = useMemoryStore.getState().memories[memoryId];
      expect(updatedMemory.category).toBe('preference');
    });

    it('should update timestamp on update', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      const originalTimestamp = state.memories[memoryId].updatedAt;

      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().updateMemory(memoryId, {
        content: 'Updated',
      });

      const updatedMemory = useMemoryStore.getState().memories[memoryId];
      expect(updatedMemory.updatedAt).toBeGreaterThan(originalTimestamp);

      vi.useRealTimers();
    });

    it('should not update non-existent memory', () => {
      expect(() => {
        useMemoryStore.getState().updateMemory('non-existent', {
          content: 'Updated',
        });
      }).not.toThrow();

      const state = useMemoryStore.getState();
      expect(Object.keys(state.memories)).toHaveLength(0);
    });

    it('should update multiple fields at once', () => {
      useMemoryStore.getState().createMemory('Original', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().updateMemory(memoryId, {
        content: 'Updated content',
        category: 'preference',
        pinned: true,
      });

      const updatedMemory = useMemoryStore.getState().memories[memoryId];
      expect(updatedMemory.content).toBe('Updated content');
      expect(updatedMemory.category).toBe('preference');
      expect(updatedMemory.pinned).toBe(true);
    });
  });

  describe('deleteMemory', () => {
    it('should delete a memory', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().deleteMemory(memoryId);

      const updatedState = useMemoryStore.getState();
      expect(Object.keys(updatedState.memories)).toHaveLength(0);
    });

    it('should not throw when deleting non-existent memory', () => {
      expect(() => {
        useMemoryStore.getState().deleteMemory('non-existent');
      }).not.toThrow();
    });
  });

  describe('pinMemory', () => {
    it('should pin a memory', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().pinMemory(memoryId, true);

      const memory = useMemoryStore.getState().memories[memoryId];
      expect(memory.pinned).toBe(true);
    });

    it('should unpin a memory', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact', true);
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().pinMemory(memoryId, false);

      const memory = useMemoryStore.getState().memories[memoryId];
      expect(memory.pinned).toBe(false);
    });

    it('should update timestamp when pinning', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      const originalTimestamp = state.memories[memoryId].updatedAt;

      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().pinMemory(memoryId, true);

      const memory = useMemoryStore.getState().memories[memoryId];
      expect(memory.updatedAt).toBeGreaterThan(originalTimestamp);

      vi.useRealTimers();
    });

    it('should not throw for non-existent memory', () => {
      expect(() => {
        useMemoryStore.getState().pinMemory('non-existent', true);
      }).not.toThrow();
    });
  });

  describe('archiveMemory', () => {
    it('should archive a memory', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];

      useMemoryStore.getState().archiveMemory(memoryId);

      const memory = useMemoryStore.getState().memories[memoryId];
      expect(memory.archived).toBe(true);
    });

    it('should update timestamp when archiving', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Test memory', 'fact');
      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      const originalTimestamp = state.memories[memoryId].updatedAt;

      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().archiveMemory(memoryId);

      const memory = useMemoryStore.getState().memories[memoryId];
      expect(memory.updatedAt).toBeGreaterThan(originalTimestamp);

      vi.useRealTimers();
    });

    it('should not throw for non-existent memory', () => {
      expect(() => {
        useMemoryStore.getState().archiveMemory('non-existent');
      }).not.toThrow();
    });
  });

  describe('searchMemories', () => {
    it('should find memories by content', () => {
      useMemoryStore.getState().createMemory('Python programming', 'fact');
      useMemoryStore.getState().createMemory('JavaScript tutorial', 'fact');
      useMemoryStore.getState().createMemory('Python best practices', 'fact');

      const results = useMemoryStore.getState().searchMemories('Python');
      expect(results).toHaveLength(2);
      expect(results.every((m) => m.content.includes('Python'))).toBe(true);
    });

    it('should be case insensitive', () => {
      useMemoryStore.getState().createMemory('Python Programming', 'fact');

      const results1 = useMemoryStore.getState().searchMemories('python');
      const results2 = useMemoryStore.getState().searchMemories('PYTHON');

      expect(results1).toHaveLength(1);
      expect(results2).toHaveLength(1);
    });

    it('should exclude archived memories from search', () => {
      useMemoryStore.getState().createMemory('Test memory 1', 'fact');
      useMemoryStore.getState().createMemory('Test memory 2', 'fact');

      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      useMemoryStore.getState().archiveMemory(memoryId);

      const results = useMemoryStore.getState().searchMemories('Test');
      expect(results).toHaveLength(1);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Old memory', 'fact');
      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().createMemory('New memory', 'fact');

      const results = useMemoryStore.getState().searchMemories('memory');
      expect(results[0].content).toBe('New memory');
      expect(results[1].content).toBe('Old memory');

      vi.useRealTimers();
    });

    it('should return empty array when no matches', () => {
      useMemoryStore.getState().createMemory('Test memory', 'fact');

      const results = useMemoryStore.getState().searchMemories('nonexistent');
      expect(results).toEqual([]);
    });
  });

  describe('getPinnedMemories', () => {
    it('should return only pinned memories', () => {
      useMemoryStore.getState().createMemory('Pinned 1', 'fact', true);
      useMemoryStore.getState().createMemory('Not pinned', 'fact', false);
      useMemoryStore.getState().createMemory('Pinned 2', 'preference', true);

      const pinnedMemories = useMemoryStore.getState().getPinnedMemories();
      expect(pinnedMemories).toHaveLength(2);
      expect(pinnedMemories.every((m) => m.pinned)).toBe(true);
    });

    it('should exclude archived memories even if pinned', () => {
      useMemoryStore.getState().createMemory('Pinned memory', 'fact', true);

      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      useMemoryStore.getState().archiveMemory(memoryId);

      const pinnedMemories = useMemoryStore.getState().getPinnedMemories();
      expect(pinnedMemories).toHaveLength(0);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Old pinned', 'fact', true);
      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().createMemory('New pinned', 'fact', true);

      const pinnedMemories = useMemoryStore.getState().getPinnedMemories();
      expect(pinnedMemories[0].content).toBe('New pinned');
      expect(pinnedMemories[1].content).toBe('Old pinned');

      vi.useRealTimers();
    });
  });

  describe('getMemoriesByCategory', () => {
    it('should return memories for specific category', () => {
      useMemoryStore.getState().createMemory('Preference 1', 'preference');
      useMemoryStore.getState().createMemory('Fact 1', 'fact');
      useMemoryStore.getState().createMemory('Preference 2', 'preference');

      const preferences = useMemoryStore.getState().getMemoriesByCategory('preference');
      expect(preferences).toHaveLength(2);
      expect(preferences.every((m) => m.category === 'preference')).toBe(true);
    });

    it('should exclude archived memories', () => {
      useMemoryStore.getState().createMemory('Memory 1', 'fact');
      useMemoryStore.getState().createMemory('Memory 2', 'fact');

      const state = useMemoryStore.getState();
      const memoryId = Object.keys(state.memories)[0];
      useMemoryStore.getState().archiveMemory(memoryId);

      const facts = useMemoryStore.getState().getMemoriesByCategory('fact');
      expect(facts).toHaveLength(1);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Old fact', 'fact');
      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().createMemory('New fact', 'fact');

      const facts = useMemoryStore.getState().getMemoriesByCategory('fact');
      expect(facts[0].content).toBe('New fact');
      expect(facts[1].content).toBe('Old fact');

      vi.useRealTimers();
    });

    it('should return empty array when no memories in category', () => {
      useMemoryStore.getState().createMemory('Test', 'fact');

      const contexts = useMemoryStore.getState().getMemoriesByCategory('context');
      expect(contexts).toEqual([]);
    });
  });

  describe('clearMemories', () => {
    it('should reset all state to initial values', () => {
      useMemoryStore.getState().createMemory('Memory 1', 'fact');
      useMemoryStore.getState().createMemory('Memory 2', 'preference');

      useMemoryStore.getState().clearMemories();

      const state = useMemoryStore.getState();
      expect(Object.keys(state.memories)).toHaveLength(0);
    });
  });

  describe('selectAllMemories', () => {
    it('should return all non-archived memories', () => {
      useMemoryStore.getState().createMemory('Memory 1', 'fact');
      useMemoryStore.getState().createMemory('Memory 2', 'preference');
      useMemoryStore.getState().createMemory('Memory 3', 'context');

      const memories = selectAllMemories(useMemoryStore.getState());
      expect(memories).toHaveLength(3);
    });

    it('should exclude archived memories', () => {
      useMemoryStore.getState().createMemory('Active', 'fact');
      useMemoryStore.getState().createMemory('Archived', 'fact');

      const state = useMemoryStore.getState();
      const memoryIds = Object.keys(state.memories);
      useMemoryStore.getState().archiveMemory(memoryIds[1]);

      const memories = selectAllMemories(useMemoryStore.getState());
      expect(memories).toHaveLength(1);
      expect(memories[0].content).toBe('Active');
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Old', 'fact');
      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().createMemory('New', 'fact');

      const memories = selectAllMemories(useMemoryStore.getState());
      expect(memories[0].content).toBe('New');
      expect(memories[1].content).toBe('Old');

      vi.useRealTimers();
    });
  });

  describe('selectArchivedMemories', () => {
    it('should return only archived memories', () => {
      useMemoryStore.getState().createMemory('Active', 'fact');
      useMemoryStore.getState().createMemory('Archived 1', 'fact');
      useMemoryStore.getState().createMemory('Archived 2', 'preference');

      const state = useMemoryStore.getState();
      const memoryIds = Object.keys(state.memories);
      useMemoryStore.getState().archiveMemory(memoryIds[1]);
      useMemoryStore.getState().archiveMemory(memoryIds[2]);

      const archived = selectArchivedMemories(useMemoryStore.getState());
      expect(archived).toHaveLength(2);
      expect(archived.every((m) => m.archived)).toBe(true);
    });

    it('should return empty array when no archived memories', () => {
      useMemoryStore.getState().createMemory('Active', 'fact');

      const archived = selectArchivedMemories(useMemoryStore.getState());
      expect(archived).toEqual([]);
    });
  });

  describe('selectMemoriesByCategory', () => {
    it('should filter by category and exclude archived', () => {
      useMemoryStore.getState().createMemory('Fact 1', 'fact');
      useMemoryStore.getState().createMemory('Preference 1', 'preference');
      useMemoryStore.getState().createMemory('Fact 2', 'fact');

      const facts = selectMemoriesByCategory(useMemoryStore.getState(), 'fact');
      expect(facts).toHaveLength(2);
      expect(facts.every((m) => m.category === 'fact')).toBe(true);
    });
  });

  describe('selectRecentMemories', () => {
    it('should return most recent memories up to limit', () => {
      for (let i = 0; i < 15; i++) {
        useMemoryStore.getState().createMemory(`Memory ${i}`, 'fact');
      }

      const recent = selectRecentMemories(useMemoryStore.getState(), 10);
      expect(recent).toHaveLength(10);
    });

    it('should default to 10 memories', () => {
      for (let i = 0; i < 15; i++) {
        useMemoryStore.getState().createMemory(`Memory ${i}`, 'fact');
      }

      const recent = selectRecentMemories(useMemoryStore.getState());
      expect(recent).toHaveLength(10);
    });

    it('should exclude archived memories', () => {
      useMemoryStore.getState().createMemory('Active', 'fact');
      useMemoryStore.getState().createMemory('Archived', 'fact');

      const state = useMemoryStore.getState();
      const memoryIds = Object.keys(state.memories);
      useMemoryStore.getState().archiveMemory(memoryIds[1]);

      const recent = selectRecentMemories(useMemoryStore.getState());
      expect(recent).toHaveLength(1);
    });

    it('should sort by updated timestamp descending', () => {
      vi.useFakeTimers();

      useMemoryStore.getState().createMemory('Old', 'fact');
      vi.advanceTimersByTime(1000);

      useMemoryStore.getState().createMemory('New', 'fact');

      const recent = selectRecentMemories(useMemoryStore.getState());
      expect(recent[0].content).toBe('New');

      vi.useRealTimers();
    });
  });
});
