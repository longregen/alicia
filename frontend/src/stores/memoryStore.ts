import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type MemoryCategory = 'preference' | 'fact' | 'context' | 'instruction';

export interface Memory {
  id: string;
  content: string;
  category: MemoryCategory;
  tags: string[];
  importance: number;
  pinned: boolean;
  archived: boolean;
  createdAt: number;
  updatedAt: number;
  usageCount: number;
}

interface MemoryStoreState {
  memories: Record<string, Memory>;
}

interface MemoryStoreActions {
  // Memory actions
  createMemory: (
    content: string,
    category: MemoryCategory,
    pinned?: boolean
  ) => void;
  updateMemory: (id: string, updates: Partial<Omit<Memory, 'id' | 'createdAt'>>) => void;
  deleteMemory: (id: string) => void;
  pinMemory: (id: string, pinned: boolean) => void;
  archiveMemory: (id: string) => void;

  // Query actions
  searchMemories: (query: string) => Memory[];
  getPinnedMemories: () => Memory[];
  getMemoriesByCategory: (category: MemoryCategory) => Memory[];

  // Bulk operations
  clearMemories: () => void;
}

type MemoryStore = MemoryStoreState & MemoryStoreActions;

const initialState: MemoryStoreState = {
  memories: {},
};

export const useMemoryStore = create<MemoryStore>()(
  immer((set, get) => ({
    ...initialState,

    // Memory actions
    createMemory: (content, category, pinned = false) =>
      set((state) => {
        const id = crypto.randomUUID();
        const timestamp = Date.now();
        state.memories[id] = {
          id,
          content,
          category,
          tags: [],
          importance: 0.5,
          pinned,
          archived: false,
          createdAt: timestamp,
          updatedAt: timestamp,
          usageCount: 0,
        };
      }),

    updateMemory: (id, updates) =>
      set((state) => {
        if (state.memories[id]) {
          Object.assign(state.memories[id], updates, {
            updatedAt: Date.now(),
          });
        }
      }),

    deleteMemory: (id) =>
      set((state) => {
        delete state.memories[id];
      }),

    pinMemory: (id, pinned) =>
      set((state) => {
        if (state.memories[id]) {
          state.memories[id].pinned = pinned;
          state.memories[id].updatedAt = Date.now();
        }
      }),

    archiveMemory: (id) =>
      set((state) => {
        if (state.memories[id]) {
          state.memories[id].archived = true;
          state.memories[id].updatedAt = Date.now();
        }
      }),

    // Query actions
    searchMemories: (query) => {
      const lowerQuery = query.toLowerCase();
      return Object.values(get().memories)
        .filter(
          (m) => !m.archived && m.content.toLowerCase().includes(lowerQuery)
        )
        .sort((a, b) => b.updatedAt - a.updatedAt);
    },

    getPinnedMemories: () => {
      return Object.values(get().memories)
        .filter((m) => m.pinned && !m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt);
    },

    getMemoriesByCategory: (category) => {
      return Object.values(get().memories)
        .filter((m) => m.category === category && !m.archived)
        .sort((a, b) => b.updatedAt - a.updatedAt);
    },

    // Bulk operations
    clearMemories: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);

// Utility selectors
export const selectAllMemories = (state: MemoryStore) =>
  Object.values(state.memories)
    .filter((m) => !m.archived)
    .sort((a, b) => b.updatedAt - a.updatedAt);

export const selectArchivedMemories = (state: MemoryStore) =>
  Object.values(state.memories)
    .filter((m) => m.archived)
    .sort((a, b) => b.updatedAt - a.updatedAt);

export const selectMemoriesByCategory = (
  state: MemoryStore,
  category: MemoryCategory
) =>
  Object.values(state.memories)
    .filter((m) => m.category === category && !m.archived)
    .sort((a, b) => b.updatedAt - a.updatedAt);

export const selectRecentMemories = (state: MemoryStore, limit: number = 10) =>
  Object.values(state.memories)
    .filter((m) => !m.archived)
    .sort((a, b) => b.updatedAt - a.updatedAt)
    .slice(0, limit);
