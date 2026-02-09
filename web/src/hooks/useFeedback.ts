import { useMemo } from 'react';

export type VoteType = 'up' | 'down' | 'critical';
export type VotableType = 'message' | 'tool_use' | 'memory' | 'reasoning' | 'memory_usage' | 'memory_extraction';

// Stub: voting/feedback APIs removed from backend
export function useFeedback(_targetType: VotableType, _targetId: string) {
  const counts = useMemo(() => ({ up: 0, down: 0, critical: 0 }), []);

  const handleVote = async (_vote: VoteType) => {
    console.warn('Feedback/voting is not supported in the current backend');
  };

  const handleUnvote = async () => {
    console.warn('Feedback/voting is not supported in the current backend');
  };

  const handleQuickFeedback = async (_feedback: string) => {
    console.warn('Feedback/voting is not supported in the current backend');
  };

  return {
    currentVote: null,
    currentQuickFeedback: undefined,
    isLoading: false,
    loadingAggregates: false,
    error: null,
    vote: handleVote,
    unvote: handleUnvote,
    setQuickFeedback: handleQuickFeedback,
    counts,
  };
}
