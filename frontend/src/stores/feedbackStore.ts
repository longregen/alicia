import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type VoteType = 'up' | 'down' | 'critical';
export type VotableType = 'message' | 'tool_use' | 'memory' | 'reasoning' | 'memory_usage' | 'memory_extraction';
// TODO: Remove if TargetType is not used for backward compatibility
export type TargetType = VotableType;

export interface Vote {
  id: string;
  targetType: VotableType;
  targetId: string;
  vote: VoteType;
  quickFeedback?: string;
  timestamp: number;
}

// Server-validated vote aggregates
export interface VoteAggregates {
  upvotes: number;
  downvotes: number;
  special?: Record<string, number>;
}

interface FeedbackStoreState {
  votes: Record<string, Vote>;
  // Server-validated aggregates per target
  aggregates: Record<string, VoteAggregates>;
}

interface FeedbackStoreActions {
  // Vote actions
  addVote: (
    targetType: VotableType,
    targetId: string,
    vote: VoteType,
    quickFeedback?: string
  ) => void;
  removeVote: (targetType: VotableType, targetId: string) => void;
  getVote: (targetType: VotableType, targetId: string) => Vote | undefined;
  getVoteCounts: (
    targetType: VotableType,
    targetId: string
  ) => { up: number; down: number; critical: number };

  // Server aggregates actions
  setAggregates: (
    targetType: VotableType,
    targetId: string,
    aggregates: VoteAggregates
  ) => void;
  setBatchAggregates: (
    targetType: VotableType,
    aggregatesMap: Record<string, VoteAggregates>
  ) => void;
  getAggregates: (
    targetType: VotableType,
    targetId: string
  ) => VoteAggregates | undefined;

  // Bulk operations
  clearFeedback: () => void;
}

type FeedbackStore = FeedbackStoreState & FeedbackStoreActions;

const initialState: FeedbackStoreState = {
  votes: {},
  aggregates: {},
};

const getVoteKey = (targetType: VotableType, targetId: string): string =>
  `${targetType}:${targetId}`;

export const useFeedbackStore = create<FeedbackStore>()(
  immer((set, get) => ({
    ...initialState,

    // Vote actions
    addVote: (targetType, targetId, vote, quickFeedback) =>
      set((state) => {
        const key = getVoteKey(targetType, targetId);
        state.votes[key] = {
          id: crypto.randomUUID(),
          targetType,
          targetId,
          vote,
          quickFeedback,
          timestamp: Date.now(),
        };
      }),

    removeVote: (targetType, targetId) =>
      set((state) => {
        const key = getVoteKey(targetType, targetId);
        delete state.votes[key];
      }),

    getVote: (targetType, targetId) => {
      const key = getVoteKey(targetType, targetId);
      return get().votes[key];
    },

    getVoteCounts: (targetType, targetId) => {
      const key = getVoteKey(targetType, targetId);
      const aggregates = get().aggregates[key];

      // Prefer server aggregates if available
      if (aggregates) {
        const critical = aggregates.special?.['critical'] || 0;
        return {
          up: aggregates.upvotes,
          down: aggregates.downvotes,
          critical,
        };
      }

      // Fall back to local vote counts (single user perspective)
      const votes = Object.values(get().votes).filter(
        (v) => v.targetType === targetType && v.targetId === targetId
      );

      return {
        up: votes.filter((v) => v.vote === 'up').length,
        down: votes.filter((v) => v.vote === 'down').length,
        critical: votes.filter((v) => v.vote === 'critical').length,
      };
    },

    // Server aggregates actions
    setAggregates: (targetType, targetId, aggregates) =>
      set((state) => {
        const key = getVoteKey(targetType, targetId);
        state.aggregates[key] = aggregates;
      }),

    setBatchAggregates: (targetType, aggregatesMap) =>
      set((state) => {
        for (const [targetId, aggregates] of Object.entries(aggregatesMap)) {
          const key = getVoteKey(targetType, targetId);
          state.aggregates[key] = aggregates;
        }
      }),

    getAggregates: (targetType, targetId) => {
      const key = getVoteKey(targetType, targetId);
      return get().aggregates[key];
    },

    // Bulk operations
    clearFeedback: () =>
      set((state) => {
        Object.assign(state, initialState);
      }),
  }))
);

// Utility selectors
export const selectAllVotes = (state: FeedbackStore) =>
  Object.values(state.votes).sort((a, b) => b.timestamp - a.timestamp);

export const selectVotesByType = (state: FeedbackStore, targetType: VotableType) =>
  Object.values(state.votes)
    .filter((v) => v.targetType === targetType)
    .sort((a, b) => b.timestamp - a.timestamp);

export const selectRecentVotes = (state: FeedbackStore, limit: number = 10) =>
  Object.values(state.votes)
    .sort((a, b) => b.timestamp - a.timestamp)
    .slice(0, limit);
