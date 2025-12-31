import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { enableMapSet } from 'immer';
import type { MessageId } from '../types/streaming';

// Enable Immer MapSet plugin for Map/Set support in stores
enableMapSet();

/**
 * Message branch represents an alternative version of a message.
 * Used for local UI state only - branches are NOT persisted to backend.
 */
export interface MessageBranch {
  content: string;
  createdAt: Date;
}

/**
 * Branch state interface.
 * Tracks multiple versions of messages for local editing/branching.
 */
interface BranchState {
  // Map of messageId -> array of alternative message versions
  branches: Map<MessageId, MessageBranch[]>;

  // Map of messageId -> current version index being displayed
  currentVersionIndex: Map<MessageId, number>;

  // Actions
  initializeBranch: (messageId: MessageId, content: string) => void;
  createBranch: (messageId: MessageId, content: string) => void;
  navigateToBranch: (messageId: MessageId, direction: 'prev' | 'next') => void;
  getCurrentBranch: (messageId: MessageId) => MessageBranch | null;
  getBranchCount: (messageId: MessageId) => number;
  getCurrentIndex: (messageId: MessageId) => number;
}

/**
 * Zustand store for managing message branches.
 * Branches are local UI state only and not synced to backend.
 */
export const useBranchStore = create<BranchState>()(
  immer((set, get) => ({
    branches: new Map<MessageId, MessageBranch[]>(),
    currentVersionIndex: new Map<MessageId, number>(),

    initializeBranch: (messageId: MessageId, content: string) => {
      set((state) => {
        // Only initialize if not already initialized
        if (state.branches.has(messageId)) return;

        const initialBranch: MessageBranch = {
          content,
          createdAt: new Date(),
        };

        state.branches.set(messageId, [initialBranch]);
        state.currentVersionIndex.set(messageId, 0);
      });
    },

    createBranch: (messageId: MessageId, content: string) => {
      set((state) => {
        const existingBranches = state.branches.get(messageId) || [];
        const newBranch: MessageBranch = {
          content,
          createdAt: new Date(),
        };

        // Add new branch to the array
        const updatedBranches = [...existingBranches, newBranch];
        state.branches.set(messageId, updatedBranches);

        // Set current index to the newly created branch
        state.currentVersionIndex.set(messageId, updatedBranches.length - 1);
      });
    },

    navigateToBranch: (messageId: MessageId, direction: 'prev' | 'next') => {
      set((state) => {
        const branches = state.branches.get(messageId);
        if (!branches || branches.length <= 1) return;

        const currentIndex = state.currentVersionIndex.get(messageId) ?? 0;

        if (direction === 'prev') {
          state.currentVersionIndex.set(messageId, Math.max(0, currentIndex - 1));
        } else {
          state.currentVersionIndex.set(messageId, Math.min(branches.length - 1, currentIndex + 1));
        }
      });
    },

    getCurrentBranch: (messageId: MessageId) => {
      const state = get();
      const branches = state.branches.get(messageId);
      if (!branches || branches.length === 0) return null;

      const index = state.currentVersionIndex.get(messageId) ?? 0;
      return branches[index] ?? null;
    },

    getBranchCount: (messageId: MessageId) => {
      const state = get();
      const branches = state.branches.get(messageId);
      return branches?.length ?? 0;
    },

    getCurrentIndex: (messageId: MessageId) => {
      const state = get();
      return state.currentVersionIndex.get(messageId) ?? 0;
    },
  }))
);
