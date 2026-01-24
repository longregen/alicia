import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type MemoryCategory = 'preference' | 'fact' | 'context' | 'instruction' | 'history';

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
  createMemory: (
    content: string,
    category: MemoryCategory,
    pinned?: boolean
  ) => void;
  setMemory: (memory: Memory) => void;
  updateMemory: (id: string, updates: Partial<Omit<Memory, 'id' | 'createdAt'>>) => void;
  deleteMemory: (id: string) => void;
  pinMemory: (id: string, pinned: boolean) => void;
  archiveMemory: (id: string) => void;

  clearMemories: () => void;
}

type MemoryStore = MemoryStoreState & MemoryStoreActions;

const initialState: MemoryStoreState = {
  memories: {},
};

export const useMemoryStore = create<MemoryStore>()(
  immer((set) => ({
    ...initialState,

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

    setMemory: (memory) =>
      set((state) => {
        state.memories[memory.id] = memory;
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

    clearMemories: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);
