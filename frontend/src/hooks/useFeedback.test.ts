import { renderHook, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useFeedback } from './useFeedback';
import { useFeedbackStore } from '../stores/feedbackStore';
import { api } from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  api: {
    voteOnMessage: vi.fn(),
    voteOnToolUse: vi.fn(),
    voteOnMemory: vi.fn(),
    voteOnReasoning: vi.fn(),
    removeMessageVote: vi.fn(),
    removeToolUseVote: vi.fn(),
    removeMemoryVote: vi.fn(),
    removeReasoningVote: vi.fn(),
    getMessageVotes: vi.fn(),
    getToolUseVotes: vi.fn(),
    getMemoryVotes: vi.fn(),
    getReasoningVotes: vi.fn(),
  },
}));

describe('useFeedback', () => {
  beforeEach(() => {
    // Reset store state before each test
    useFeedbackStore.getState().clearFeedback();
    vi.clearAllMocks();
  });

  it('should initialize with no vote and empty counts', async () => {
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    expect(result.current.currentVote).toBeNull();
    expect(result.current.counts).toEqual({ up: 0, down: 0, critical: 0 });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();

    // Wait for async effects to complete
    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });
  });

  it('should fetch aggregates on mount for message target', async () => {
    const mockResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 5, downvotes: 2, user_vote: null, special: {} };
    vi.mocked(api.getMessageVotes).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    expect(api.getMessageVotes).toHaveBeenCalledWith('msg-1');
    expect(result.current.counts.up).toBe(5);
    expect(result.current.counts.down).toBe(2);
  });

  it('should fetch aggregates on mount for tool_use target', async () => {
    const mockResponse = { target_id: 'tool-1', target_type: 'tool_use', upvotes: 3, downvotes: 1, user_vote: null, special: {} };
    vi.mocked(api.getToolUseVotes).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('tool_use', 'tool-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    expect(api.getToolUseVotes).toHaveBeenCalledWith('tool-1');
    expect(result.current.counts.up).toBe(3);
    expect(result.current.counts.down).toBe(1);
  });

  it('should handle successful upvote on message', async () => {
    const mockResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 1, downvotes: 0, user_vote: 'up', special: {} };
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.vote('up');
    });

    expect(api.voteOnMessage).toHaveBeenCalledWith('msg-1', 'up', undefined);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Check vote was added to store
    const voteInStore = useFeedbackStore.getState().getVote('message', 'msg-1');
    expect(voteInStore?.vote).toBe('up');
    expect(result.current.counts.up).toBe(1);
    expect(result.current.error).toBeNull();
  });

  it('should handle successful downvote on tool_use', async () => {
    const mockResponse = { target_id: 'tool-1', target_type: 'tool_use', upvotes: 0, downvotes: 1, user_vote: 'down', special: {} };
    vi.mocked(api.getToolUseVotes).mockResolvedValue({ target_id: 'tool-1', target_type: 'tool_use', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnToolUse).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('tool_use', 'tool-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.vote('down');
    });

    expect(api.voteOnToolUse).toHaveBeenCalledWith('tool-1', 'down', undefined);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const voteInStore = useFeedbackStore.getState().getVote('tool_use', 'tool-1');
    expect(voteInStore?.vote).toBe('down');
    expect(result.current.counts.down).toBe(1);
  });

  it('should toggle vote off when clicking same vote', async () => {
    const mockVoteResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 1, downvotes: 0, user_vote: 'up', special: {} };
    const mockUnvoteResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} };
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockResolvedValue(mockVoteResponse);
    vi.mocked(api.removeMessageVote).mockResolvedValue(mockUnvoteResponse);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    // First vote
    await act(async () => {
      await result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('up');
    });

    // Second vote (toggle off)
    await act(async () => {
      await result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBeNull();
    });

    expect(api.removeMessageVote).toHaveBeenCalledWith('msg-1');
    expect(result.current.counts.up).toBe(0);
  });

  it('should switch vote from up to down', async () => {
    const mockUpResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 1, downvotes: 0, user_vote: 'up', special: {} };
    const mockDownResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 1, user_vote: 'down', special: {} };
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockResolvedValueOnce(mockUpResponse).mockResolvedValueOnce(mockDownResponse);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    // Vote up
    await act(async () => {
      await result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('up');
    });

    // Vote down (should switch)
    await act(async () => {
      await result.current.vote('down');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('down');
    });

    expect(result.current.counts.down).toBe(1);
  });

  it('should handle critical vote on memory', async () => {
    const mockResponse = { target_id: 'mem-1', target_type: 'memory', upvotes: 0, downvotes: 0, user_vote: 'critical', special: { critical: 1 } };
    vi.mocked(api.getMemoryVotes).mockResolvedValue({ target_id: 'mem-1', target_type: 'memory', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMemory).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('memory', 'mem-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.vote('critical');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('critical');
    });

    expect(api.voteOnMemory).toHaveBeenCalledWith('mem-1', 'critical');
    expect(result.current.counts.critical).toBe(1);
  });

  it('should handle quick feedback for tool_use', async () => {
    const mockResponse = { target_id: 'tool-1', target_type: 'tool_use', upvotes: 0, downvotes: 1, user_vote: 'down', special: {} };
    vi.mocked(api.getToolUseVotes).mockResolvedValue({ target_id: 'tool-1', target_type: 'tool_use', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnToolUse).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('tool_use', 'tool-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.setQuickFeedback('Too slow');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('down');
    });

    expect(api.voteOnToolUse).toHaveBeenCalledWith('tool-1', 'down', 'Too slow');
    expect(result.current.currentQuickFeedback).toBe('Too slow');
  });

  it('should handle unvote operation', async () => {
    const mockVoteResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 1, downvotes: 0, user_vote: 'up', special: {} };
    const mockUnvoteResponse = { target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} };
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockResolvedValue(mockVoteResponse);
    vi.mocked(api.removeMessageVote).mockResolvedValue(mockUnvoteResponse);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    // First vote
    await act(async () => {
      await result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('up');
    });

    // Unvote
    await act(async () => {
      await result.current.unvote();
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBeNull();
    });

    expect(api.removeMessageVote).toHaveBeenCalledWith('msg-1');
  });

  it('should handle API error during vote', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const error = new Error('Server error');
    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockRejectedValue(error);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.vote('up');
    });

    expect(result.current.error).toBe('Server error');
    expect(result.current.currentVote).toBeNull();
    expect(result.current.isLoading).toBe(false);
    consoleErrorSpy.mockRestore();
  });

  it('should silently handle 404 when fetching aggregates', async () => {
    const error = new Error('404');
    vi.mocked(api.getMessageVotes).mockRejectedValue(error);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    // Should not show error for 404
    expect(result.current.counts).toEqual({ up: 0, down: 0, critical: 0 });
  });

  it('should not fetch aggregates when targetId is empty', () => {
    const { result } = renderHook(() => useFeedback('message', ''));

    expect(result.current.loadingAggregates).toBe(false);
    expect(api.getMessageVotes).not.toHaveBeenCalled();
  });

  it('should handle reasoning target type', async () => {
    const mockResponse = { target_id: 'reason-1', target_type: 'reasoning', upvotes: 2, downvotes: 1, user_vote: 'up', special: {} };
    vi.mocked(api.getReasoningVotes).mockResolvedValue({ target_id: 'reason-1', target_type: 'reasoning', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnReasoning).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useFeedback('reasoning', 'reason-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    await act(async () => {
      await result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.currentVote).toBe('up');
    });

    expect(api.voteOnReasoning).toHaveBeenCalledWith('reason-1', 'up');
  });

  it('should set isLoading to true during vote operation', async () => {
    let resolveVote: (value: any) => void;
    const votePromise = new Promise((resolve) => {
      resolveVote = resolve;
    });

    vi.mocked(api.getMessageVotes).mockResolvedValue({ target_id: 'msg-1', target_type: 'message', upvotes: 0, downvotes: 0, user_vote: null, special: {} });
    vi.mocked(api.voteOnMessage).mockReturnValue(votePromise as any);

    const { result } = renderHook(() => useFeedback('message', 'msg-1'));

    await waitFor(() => {
      expect(result.current.loadingAggregates).toBe(false);
    });

    let votePromiseResult: Promise<void>;
    act(() => {
      votePromiseResult = result.current.vote('up');
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(true);
    });

    await act(async () => {
      resolveVote!({ target_id: 'msg-1', target_type: 'message', upvotes: 1, downvotes: 0, user_vote: 'up', special: {} });
      await votePromiseResult!;
    });

    expect(result.current.isLoading).toBe(false);
  });
});
