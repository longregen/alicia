import { useCallback, useState, useEffect, useMemo } from 'react';
import {
  useFeedbackStore,
  type VotableType,
  type VoteType,
  type VoteAggregates,
} from '../stores/feedbackStore';
import { api, type VoteResponse } from '../services/api';

/**
 * Unified hook for feedback operations on a specific target.
 * Wraps the feedbackStore with additional logic and convenience methods.
 * Makes API calls to persist votes on the server.
 *
 * @param targetType - Type of votable element (message, tool_use, memory, reasoning)
 * @param targetId - Unique ID of the target element
 * @returns Object with vote state, handlers, and aggregate counts
 *
 * @example
 * ```tsx
 * function ToolCard({ toolCall }) {
 *   const { currentVote, vote, counts, isLoading } = useFeedback('tool_use', toolCall.id);
 *
 *   return (
 *     <div>
 *       <button onClick={() => vote('up')} disabled={isLoading}>👍 {counts.up}</button>
 *       <button onClick={() => vote('down')} disabled={isLoading}>👎 {counts.down}</button>
 *       {currentVote && <span>You voted: {currentVote}</span>}
 *     </div>
 *   );
 * }
 * ```
 */
export function useFeedback(targetType: VotableType, targetId: string) {
  const [isLoading, setIsLoading] = useState(false);
  const [loadingAggregates, setLoadingAggregates] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Get store actions - stable references
  const addVote = useFeedbackStore((state) => state.addVote);
  const removeVote = useFeedbackStore((state) => state.removeVote);
  const setAggregates = useFeedbackStore((state) => state.setAggregates);

  // Subscribe to raw store state and compute locally to avoid infinite loops
  const voteKey = `${targetType}:${targetId}`;
  const rawVote = useFeedbackStore((state) => state.votes[voteKey]);
  const rawAggregates = useFeedbackStore((state) => state.aggregates[voteKey]);

  // Compute current vote locally
  const currentVote = rawVote;

  // Compute counts locally to avoid new object creation on every render
  const counts = useMemo(() => {
    if (rawAggregates && 'upvotes' in rawAggregates) {
      const critical = rawAggregates.special?.['critical'] || 0;
      return {
        up: rawAggregates.upvotes,
        down: rawAggregates.downvotes,
        critical,
      };
    }

    // Fall back to local vote count (single user perspective)
    return {
      up: rawVote?.vote === 'up' ? 1 : 0,
      down: rawVote?.vote === 'down' ? 1 : 0,
      critical: rawVote?.vote === 'critical' ? 1 : 0,
    };
  }, [rawAggregates, rawVote]);

  // Helper to update aggregates from server response
  const updateAggregatesFromResponse = useCallback(
    (response: VoteResponse) => {
      const aggregates: VoteAggregates = {
        upvotes: response.upvotes,
        downvotes: response.downvotes,
        special: response.special,
      };
      setAggregates(targetType, targetId, aggregates);
    },
    [targetType, targetId, setAggregates]
  );

  // Fetch aggregates based on target type
  const fetchAggregatesFromServer = useCallback(async () => {
    switch (targetType) {
      case 'message':
        return api.getMessageVotes(targetId);
      case 'tool_use':
        return api.getToolUseVotes(targetId);
      case 'memory':
        return api.getMemoryVotes(targetId);
      case 'reasoning':
        return api.getReasoningVotes(targetId);
    }
  }, [targetType, targetId]);

  // API call based on target type
  const submitVoteToServer = useCallback(async (vote: 'up' | 'down' | 'critical', quickFeedback?: string) => {
    switch (targetType) {
      case 'message':
        return api.voteOnMessage(targetId, vote as 'up' | 'down', quickFeedback);
      case 'tool_use':
        return api.voteOnToolUse(targetId, vote as 'up' | 'down', quickFeedback);
      case 'memory':
        return api.voteOnMemory(targetId, vote);
      case 'reasoning':
        return api.voteOnReasoning(targetId, vote as 'up' | 'down');
    }
  }, [targetType, targetId]);

  const removeVoteFromServer = useCallback(async () => {
    switch (targetType) {
      case 'message':
        return api.removeMessageVote(targetId);
      case 'tool_use':
        return api.removeToolUseVote(targetId);
      case 'memory':
        return api.removeMemoryVote(targetId);
      case 'reasoning':
        return api.removeReasoningVote(targetId);
    }
  }, [targetType, targetId]);

  // Vote handler - toggle behavior with API call
  const handleVote = useCallback(async (vote: VoteType) => {
    const current = useFeedbackStore.getState().getVote(targetType, targetId);
    setIsLoading(true);
    setError(null);

    try {
      // If clicking the same vote, remove it (toggle off)
      if (current?.vote === vote) {
        const response = await removeVoteFromServer();
        removeVote(targetType, targetId);
        updateAggregatesFromResponse(response);
      } else {
        // Otherwise, set the new vote
        const response = await submitVoteToServer(vote);
        addVote(targetType, targetId, vote);
        updateAggregatesFromResponse(response);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit vote');
      console.error('Vote error:', err);
    } finally {
      setIsLoading(false);
    }
  }, [targetType, targetId, addVote, removeVote, submitVoteToServer, removeVoteFromServer, updateAggregatesFromResponse]);

  // Quick feedback handler (for tool_use specific feedback)
  const handleQuickFeedback = useCallback(async (feedback: string) => {
    const current = useFeedbackStore.getState().getVote(targetType, targetId);
    setIsLoading(true);
    setError(null);

    try {
      // Add or update vote with quick feedback
      const response = await submitVoteToServer(current?.vote || 'down', feedback);
      addVote(targetType, targetId, current?.vote || 'down', feedback);
      updateAggregatesFromResponse(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit feedback');
      console.error('Quick feedback error:', err);
    } finally {
      setIsLoading(false);
    }
  }, [targetType, targetId, addVote, submitVoteToServer, updateAggregatesFromResponse]);

  // Unvote handler
  const handleUnvote = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await removeVoteFromServer();
      removeVote(targetType, targetId);
      updateAggregatesFromResponse(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove vote');
      console.error('Unvote error:', err);
    } finally {
      setIsLoading(false);
    }
  }, [targetType, targetId, removeVote, removeVoteFromServer, updateAggregatesFromResponse]);

  // Fetch aggregates on mount
  useEffect(() => {
    if (!targetId) {
      setLoadingAggregates(false);
      return;
    }

    let isMounted = true;

    const loadAggregates = async () => {
      try {
        const response = await fetchAggregatesFromServer();
        if (isMounted) {
          updateAggregatesFromResponse(response);
        }
      } catch (err) {
        // Silently fail for 404 - no votes exist yet
        if (isMounted && err instanceof Error && !err.message.includes('404')) {
          console.warn('Failed to fetch vote aggregates:', err);
        }
      } finally {
        if (isMounted) {
          setLoadingAggregates(false);
        }
      }
    };

    loadAggregates();

    return () => {
      isMounted = false;
    };
  }, [targetId, fetchAggregatesFromServer, updateAggregatesFromResponse]);

  return {
    // Current state
    currentVote: currentVote?.vote || null,
    currentQuickFeedback: currentVote?.quickFeedback,

    // Loading and error state
    isLoading,
    loadingAggregates,
    error,

    // Handlers
    vote: handleVote,
    unvote: handleUnvote,
    setQuickFeedback: handleQuickFeedback,

    // Aggregate counts
    counts,
  };
}
