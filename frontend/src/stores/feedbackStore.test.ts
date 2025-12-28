import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  useFeedbackStore,
  selectAllVotes,
  selectVotesByType,
  selectRecentVotes,
  type VoteType,
  type VotableType,
  type VoteAggregates,
} from './feedbackStore';

describe('feedbackStore', () => {
  beforeEach(() => {
    useFeedbackStore.getState().clearFeedback();
    // Mock crypto.randomUUID for consistent test results
    vi.stubGlobal('crypto', {
      randomUUID: () => 'test-uuid-' + Date.now(),
    });
  });

  describe('addVote', () => {
    it('should add a vote to the store', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');

      const state = useFeedbackStore.getState();
      const vote = Object.values(state.votes)[0];
      expect(vote).toBeDefined();
      expect(vote.targetType).toBe('message');
      expect(vote.targetId).toBe('msg-1');
      expect(vote.vote).toBe('up');
    });

    it('should add vote with quick feedback', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'down', 'Not helpful');

      const state = useFeedbackStore.getState();
      const vote = Object.values(state.votes)[0];
      expect(vote.quickFeedback).toBe('Not helpful');
    });

    it('should create vote with timestamp', () => {
      const before = Date.now();
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      const after = Date.now();

      const state = useFeedbackStore.getState();
      const vote = Object.values(state.votes)[0];
      expect(vote.timestamp).toBeGreaterThanOrEqual(before);
      expect(vote.timestamp).toBeLessThanOrEqual(after);
    });

    it('should overwrite existing vote for same target', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().addVote('message', 'msg-1', 'down');

      const state = useFeedbackStore.getState();
      const votes = Object.values(state.votes).filter(
        (v) => v.targetType === 'message' && v.targetId === 'msg-1'
      );
      expect(votes).toHaveLength(1);
      expect(votes[0].vote).toBe('down');
    });

    it('should handle different target types', () => {
      const targetTypes: VotableType[] = ['message', 'tool_use', 'memory', 'reasoning'];

      targetTypes.forEach((type) => {
        useFeedbackStore.getState().addVote(type, `${type}-1`, 'up');
      });

      const state = useFeedbackStore.getState();
      expect(Object.keys(state.votes)).toHaveLength(4);
    });

    it('should handle different vote types', () => {
      const voteTypes: VoteType[] = ['up', 'down', 'critical'];

      voteTypes.forEach((type, i) => {
        useFeedbackStore.getState().addVote('message', `msg-${i}`, type);
      });

      const state = useFeedbackStore.getState();
      const votes = Object.values(state.votes);
      expect(votes).toHaveLength(3);
      expect(votes.map((v) => v.vote)).toContain('up');
      expect(votes.map((v) => v.vote)).toContain('down');
      expect(votes.map((v) => v.vote)).toContain('critical');
    });
  });

  describe('removeVote', () => {
    it('should remove a vote from the store', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().removeVote('message', 'msg-1');

      const state = useFeedbackStore.getState();
      expect(Object.keys(state.votes)).toHaveLength(0);
    });

    it('should not throw when removing non-existent vote', () => {
      expect(() => {
        useFeedbackStore.getState().removeVote('message', 'non-existent');
      }).not.toThrow();
    });

    it('should only remove vote for specific target', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');

      useFeedbackStore.getState().removeVote('message', 'msg-1');

      const state = useFeedbackStore.getState();
      expect(Object.keys(state.votes)).toHaveLength(1);
      const remainingVote = Object.values(state.votes)[0];
      expect(remainingVote.targetId).toBe('msg-2');
    });
  });

  describe('getVote', () => {
    it('should return vote for target', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');

      const vote = useFeedbackStore.getState().getVote('message', 'msg-1');
      expect(vote).toBeDefined();
      expect(vote?.targetId).toBe('msg-1');
      expect(vote?.vote).toBe('up');
    });

    it('should return undefined for non-existent vote', () => {
      const vote = useFeedbackStore.getState().getVote('message', 'non-existent');
      expect(vote).toBeUndefined();
    });
  });

  describe('getVoteCounts', () => {
    it('should return vote counts from server aggregates if available', () => {
      const aggregates: VoteAggregates = {
        upvotes: 5,
        downvotes: 2,
        special: { critical: 1 },
      };

      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates);

      const counts = useFeedbackStore.getState().getVoteCounts('message', 'msg-1');
      expect(counts.up).toBe(5);
      expect(counts.down).toBe(2);
      expect(counts.critical).toBe(1);
    });

    it('should return zero for missing special votes', () => {
      const aggregates: VoteAggregates = {
        upvotes: 3,
        downvotes: 1,
      };

      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates);

      const counts = useFeedbackStore.getState().getVoteCounts('message', 'msg-1');
      expect(counts.critical).toBe(0);
    });

    it('should fall back to local votes when no aggregates exist', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');

      const counts = useFeedbackStore.getState().getVoteCounts('message', 'msg-1');
      expect(counts.up).toBe(1);
      expect(counts.down).toBe(0);
      expect(counts.critical).toBe(0);
    });

    it('should return zero counts for non-existent target', () => {
      const counts = useFeedbackStore.getState().getVoteCounts('message', 'non-existent');
      expect(counts.up).toBe(0);
      expect(counts.down).toBe(0);
      expect(counts.critical).toBe(0);
    });
  });

  describe('setAggregates', () => {
    it('should set server aggregates for target', () => {
      const aggregates: VoteAggregates = {
        upvotes: 10,
        downvotes: 3,
        special: { critical: 2 },
      };

      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates);

      const storedAggregates = useFeedbackStore.getState().getAggregates('message', 'msg-1');
      expect(storedAggregates).toEqual(aggregates);
    });

    it('should overwrite existing aggregates', () => {
      const aggregates1: VoteAggregates = {
        upvotes: 5,
        downvotes: 2,
      };

      const aggregates2: VoteAggregates = {
        upvotes: 10,
        downvotes: 4,
      };

      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates1);
      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates2);

      const storedAggregates = useFeedbackStore.getState().getAggregates('message', 'msg-1');
      expect(storedAggregates).toEqual(aggregates2);
    });
  });

  describe('getAggregates', () => {
    it('should return aggregates for target', () => {
      const aggregates: VoteAggregates = {
        upvotes: 7,
        downvotes: 3,
      };

      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates);

      const result = useFeedbackStore.getState().getAggregates('message', 'msg-1');
      expect(result).toEqual(aggregates);
    });

    it('should return undefined for non-existent aggregates', () => {
      const result = useFeedbackStore.getState().getAggregates('message', 'non-existent');
      expect(result).toBeUndefined();
    });
  });

  describe('clearFeedback', () => {
    it('should reset all state to initial values', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().addVote('tool_use', 'tool-1', 'down');

      const aggregates: VoteAggregates = {
        upvotes: 5,
        downvotes: 2,
      };
      useFeedbackStore.getState().setAggregates('message', 'msg-1', aggregates);

      useFeedbackStore.getState().clearFeedback();

      const state = useFeedbackStore.getState();
      expect(Object.keys(state.votes)).toHaveLength(0);
      expect(Object.keys(state.aggregates)).toHaveLength(0);
    });
  });

  describe('selectAllVotes', () => {
    it('should return all votes sorted by timestamp descending', () => {
      vi.useFakeTimers();

      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      vi.advanceTimersByTime(1000);

      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');
      vi.advanceTimersByTime(1000);

      useFeedbackStore.getState().addVote('tool_use', 'tool-1', 'up');

      const votes = selectAllVotes(useFeedbackStore.getState());
      expect(votes).toHaveLength(3);
      expect(votes[0].targetType).toBe('tool_use');
      expect(votes[1].targetType).toBe('message');
      expect(votes[1].targetId).toBe('msg-2');
      expect(votes[2].targetId).toBe('msg-1');

      vi.useRealTimers();
    });

    it('should return empty array when no votes exist', () => {
      const votes = selectAllVotes(useFeedbackStore.getState());
      expect(votes).toEqual([]);
    });
  });

  describe('selectVotesByType', () => {
    it('should return votes filtered by target type', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');
      useFeedbackStore.getState().addVote('tool_use', 'tool-1', 'up');

      const messageVotes = selectVotesByType(useFeedbackStore.getState(), 'message');
      expect(messageVotes).toHaveLength(2);
      expect(messageVotes.every((v) => v.targetType === 'message')).toBe(true);

      const toolVotes = selectVotesByType(useFeedbackStore.getState(), 'tool_use');
      expect(toolVotes).toHaveLength(1);
      expect(toolVotes[0].targetType).toBe('tool_use');
    });

    it('should return empty array when no votes match type', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');

      const votes = selectVotesByType(useFeedbackStore.getState(), 'memory');
      expect(votes).toEqual([]);
    });

    it('should sort by timestamp descending', () => {
      vi.useFakeTimers();

      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      vi.advanceTimersByTime(1000);

      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');

      const votes = selectVotesByType(useFeedbackStore.getState(), 'message');
      expect(votes[0].targetId).toBe('msg-2');
      expect(votes[1].targetId).toBe('msg-1');

      vi.useRealTimers();
    });
  });

  describe('selectRecentVotes', () => {
    it('should return most recent votes up to limit', () => {
      for (let i = 0; i < 15; i++) {
        useFeedbackStore.getState().addVote('message', `msg-${i}`, 'up');
      }

      const recentVotes = selectRecentVotes(useFeedbackStore.getState(), 10);
      expect(recentVotes).toHaveLength(10);
    });

    it('should default to 10 votes', () => {
      for (let i = 0; i < 15; i++) {
        useFeedbackStore.getState().addVote('message', `msg-${i}`, 'up');
      }

      const recentVotes = selectRecentVotes(useFeedbackStore.getState());
      expect(recentVotes).toHaveLength(10);
    });

    it('should return all votes when less than limit', () => {
      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');

      const recentVotes = selectRecentVotes(useFeedbackStore.getState(), 10);
      expect(recentVotes).toHaveLength(2);
    });

    it('should sort by timestamp descending', () => {
      vi.useFakeTimers();

      useFeedbackStore.getState().addVote('message', 'msg-1', 'up');
      vi.advanceTimersByTime(1000);

      useFeedbackStore.getState().addVote('message', 'msg-2', 'down');
      vi.advanceTimersByTime(1000);

      useFeedbackStore.getState().addVote('message', 'msg-3', 'up');

      const recentVotes = selectRecentVotes(useFeedbackStore.getState(), 2);
      expect(recentVotes[0].targetId).toBe('msg-3');
      expect(recentVotes[1].targetId).toBe('msg-2');

      vi.useRealTimers();
    });
  });
});
