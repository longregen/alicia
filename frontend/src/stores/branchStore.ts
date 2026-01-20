import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { enableMapSet } from 'immer';
import type { MessageId } from '../types/streaming';
import type { Message } from '../types/models';

// Enable Immer MapSet plugin for Map/Set support in stores
enableMapSet();

const API_BASE = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL}/api/v1`
  : '/api/v1';

/**
 * Sibling message from backend.
 * Represents an alternative version of a message at the same position in conversation.
 */
export interface SiblingMessage {
  id: string;
  content: string;
  createdAt: string;
}

/**
 * Branch state for a message.
 * Tracks sibling messages from the backend.
 */
interface MessageBranchState {
  siblings: SiblingMessage[];
  currentIndex: number;
  loading: boolean;
  error: string | null;
}

/**
 * Branch update notification from WebSocket.
 * Sent when a new sibling is created (e.g., via edit operation).
 */
export interface BranchUpdateNotification {
  conversationId: string;
  parentMessageId: string;
  newSibling: SiblingMessage;
  allSiblings: SiblingMessage[];
  totalCount: number;
}

/**
 * Branch state interface.
 * Tracks message siblings from backend for branch navigation.
 */
interface BranchState {
  // Map of messageId -> branch state with siblings
  branchStates: Map<MessageId, MessageBranchState>;

  // Actions
  fetchSiblings: (conversationId: string, messageId: MessageId) => Promise<void>;
  switchBranch: (
    conversationId: string,
    messageId: MessageId,
    direction: 'prev' | 'next'
  ) => Promise<SiblingMessage | null>;
  selectBranch: (
    conversationId: string,
    messageId: MessageId,
    targetMessageId: string
  ) => Promise<SiblingMessage | null>;
  initializeFromMessages: (conversationId: string, messages: Message[]) => void;
  /**
   * Handle BranchUpdate notification from WebSocket.
   * Updates branch state for all affected siblings when a new sibling is created.
   * @param update - The branch update notification containing all sibling info
   */
  handleBranchUpdate: (update: BranchUpdateNotification) => void;
  getCurrentSibling: (messageId: MessageId) => SiblingMessage | null;
  getSiblingCount: (messageId: MessageId) => number;
  getCurrentIndex: (messageId: MessageId) => number;
  isLoading: (messageId: MessageId) => boolean;
  clearBranches: () => void;
}

/**
 * Fetch siblings for a message from the backend.
 */
async function fetchSiblingsFromBackend(
  conversationId: string,
  messageId: string
): Promise<SiblingMessage[]> {
  const response = await fetch(
    `${API_BASE}/conversations/${conversationId}/messages/${messageId}/siblings`
  );

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Failed to fetch siblings: ${response.status}`);
  }

  const data = await response.json();
  // Backend returns { siblings: Message[] }
  const messages: Message[] = data.siblings || [];

  return messages.map((msg) => ({
    id: msg.id,
    content: msg.contents,
    createdAt: msg.created_at,
  }));
}

/**
 * Switch the conversation tip to a different branch.
 */
async function switchBranchOnBackend(
  conversationId: string,
  tipMessageId: string
): Promise<void> {
  const response = await fetch(
    `${API_BASE}/conversations/${conversationId}/switch-branch`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tipMessageId }),
    }
  );

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Failed to switch branch: ${response.status}`);
  }
}

/**
 * Zustand store for managing message branches.
 * Branches are synced with the backend - siblings are fetched from server
 * and branch switching updates the conversation tip on the server.
 */
export const useBranchStore = create<BranchState>()(
  immer((set, get) => ({
    branchStates: new Map<MessageId, MessageBranchState>(),

    fetchSiblings: async (conversationId: string, messageId: MessageId) => {
      // Set loading state
      set((state) => {
        const existing = state.branchStates.get(messageId);
        state.branchStates.set(messageId, {
          siblings: existing?.siblings || [],
          currentIndex: existing?.currentIndex ?? 0,
          loading: true,
          error: null,
        });
      });

      try {
        const siblings = await fetchSiblingsFromBackend(conversationId, messageId);

        // Find the index of the current message in siblings
        const currentIndex = siblings.findIndex((s) => s.id === messageId);

        set((state) => {
          state.branchStates.set(messageId, {
            siblings,
            currentIndex: currentIndex >= 0 ? currentIndex : 0,
            loading: false,
            error: null,
          });
        });
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to fetch siblings';
        set((state) => {
          const existing = state.branchStates.get(messageId);
          state.branchStates.set(messageId, {
            siblings: existing?.siblings || [],
            currentIndex: existing?.currentIndex ?? 0,
            loading: false,
            error: errorMessage,
          });
        });
      }
    },

    switchBranch: async (
      conversationId: string,
      messageId: MessageId,
      direction: 'prev' | 'next'
    ) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);

      if (!branchState || branchState.siblings.length <= 1) {
        return null;
      }

      const { siblings, currentIndex } = branchState;
      let newIndex: number;

      if (direction === 'prev') {
        newIndex = Math.max(0, currentIndex - 1);
      } else {
        newIndex = Math.min(siblings.length - 1, currentIndex + 1);
      }

      // Only proceed if index actually changed
      if (newIndex === currentIndex) {
        return null;
      }

      const targetSibling = siblings[newIndex];
      if (!targetSibling) {
        return null;
      }

      // Set loading state during switch
      set((s) => {
        const existing = s.branchStates.get(messageId);
        if (existing) {
          existing.loading = true;
          existing.error = null;
        }
      });

      try {
        // Call backend to switch the conversation tip
        await switchBranchOnBackend(conversationId, targetSibling.id);

        // Update local state with new index
        set((s) => {
          const existing = s.branchStates.get(messageId);
          if (existing) {
            existing.currentIndex = newIndex;
            existing.loading = false;
          }
        });

        return targetSibling;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to switch branch';
        set((s) => {
          const existing = s.branchStates.get(messageId);
          if (existing) {
            existing.loading = false;
            existing.error = errorMessage;
          }
        });
        return null;
      }
    },

    selectBranch: async (
      conversationId: string,
      messageId: MessageId,
      targetMessageId: string
    ) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);

      if (!branchState) {
        return null;
      }

      const { siblings } = branchState;
      const targetIndex = siblings.findIndex((s) => s.id === targetMessageId);

      if (targetIndex < 0) {
        return null;
      }

      const targetSibling = siblings[targetIndex];
      if (!targetSibling) {
        return null;
      }

      // Set loading state during switch
      set((s) => {
        const existing = s.branchStates.get(messageId);
        if (existing) {
          existing.loading = true;
          existing.error = null;
        }
      });

      try {
        // Call backend to switch the conversation tip
        await switchBranchOnBackend(conversationId, targetMessageId);

        // Update local state with new index
        set((s) => {
          const existing = s.branchStates.get(messageId);
          if (existing) {
            existing.currentIndex = targetIndex;
            existing.loading = false;
          }
        });

        return targetSibling;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to switch branch';
        set((s) => {
          const existing = s.branchStates.get(messageId);
          if (existing) {
            existing.loading = false;
            existing.error = errorMessage;
          }
        });
        return null;
      }
    },

    initializeFromMessages: (_conversationId: string, messages: Message[]) => {
      // Group messages by their previous_id to find siblings
      // Messages with the same previous_id are siblings
      const siblingGroups = new Map<string, Message[]>();

      for (const msg of messages) {
        const parentId = msg.previous_id || 'root';
        const group = siblingGroups.get(parentId) || [];
        group.push(msg);
        siblingGroups.set(parentId, group);
      }

      set((state) => {
        // For each message, check if it has siblings
        for (const msg of messages) {
          const parentId = msg.previous_id || 'root';
          const siblings = siblingGroups.get(parentId) || [];

          if (siblings.length > 1) {
            // This message has siblings - initialize branch state
            const siblingData: SiblingMessage[] = siblings.map((s) => ({
              id: s.id,
              content: s.contents,
              createdAt: s.created_at,
            }));

            const currentIndex = siblingData.findIndex((s) => s.id === msg.id);

            state.branchStates.set(msg.id as MessageId, {
              siblings: siblingData,
              currentIndex: currentIndex >= 0 ? currentIndex : 0,
              loading: false,
              error: null,
            });
          }
        }
      });
    },

    getCurrentSibling: (messageId: MessageId) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);
      if (!branchState || branchState.siblings.length === 0) {
        return null;
      }
      return branchState.siblings[branchState.currentIndex] ?? null;
    },

    getSiblingCount: (messageId: MessageId) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);
      return branchState?.siblings.length ?? 0;
    },

    getCurrentIndex: (messageId: MessageId) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);
      return branchState?.currentIndex ?? 0;
    },

    isLoading: (messageId: MessageId) => {
      const state = get();
      const branchState = state.branchStates.get(messageId);
      return branchState?.loading ?? false;
    },

    clearBranches: () => {
      set((state) => {
        state.branchStates.clear();
      });
    },

    handleBranchUpdate: (update: BranchUpdateNotification) => {
      // When a new sibling is created, update branch state for ALL siblings.
      // Each sibling message needs to know about all its siblings so the
      // BranchNavigator UI can display the correct count and navigation.
      set((state) => {
        // The new sibling is the current one (user just created it via edit)
        const newSiblingIndex = update.allSiblings.findIndex(
          (s) => s.id === update.newSibling.id
        );

        // Update branch state for each sibling
        for (const sibling of update.allSiblings) {
          const siblingId = sibling.id as MessageId;
          const existing = state.branchStates.get(siblingId);

          // For the new sibling, set it as the current index
          // For existing siblings, preserve their current index if they had one,
          // otherwise use their position in the sibling list
          let currentIndex: number;
          if (sibling.id === update.newSibling.id) {
            currentIndex = newSiblingIndex >= 0 ? newSiblingIndex : update.allSiblings.length - 1;
          } else if (existing) {
            // Try to preserve existing current index, but clamp to valid range
            currentIndex = Math.min(existing.currentIndex, update.allSiblings.length - 1);
          } else {
            // New entry - find this sibling's position
            const siblingIndex = update.allSiblings.findIndex((s) => s.id === sibling.id);
            currentIndex = siblingIndex >= 0 ? siblingIndex : 0;
          }

          state.branchStates.set(siblingId, {
            siblings: update.allSiblings,
            currentIndex,
            loading: false,
            error: null,
          });
        }
      });
    },
  }))
);
