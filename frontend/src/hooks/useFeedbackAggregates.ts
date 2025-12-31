import { useMemo } from 'react';
import { useFeedbackStore } from '../stores/feedbackStore';
import { useConversationStore } from './useConversationStore';
import type { MessageId } from '../types/streaming';

export type SentimentIndicator = 'positive' | 'mixed' | 'negative' | 'neutral';

export interface FeedbackAggregates {
  // Counts by target type
  toolUseFeedback: {
    upvotes: number;
    downvotes: number;
    critical: number;
  };
  memoryFeedback: {
    upvotes: number;
    downvotes: number;
    critical: number;
  };
  reasoningFeedback: {
    upvotes: number;
    downvotes: number;
    critical: number;
  };

  // Overall sentiment
  sentiment: SentimentIndicator;

  // Total counts
  totalVotes: number;
  totalPositive: number;
  totalNegative: number;
}

/**
 * Computes summary indicators (green/yellow/red) from feedback for a message.
 * Aggregates votes across all tools, memories, and reasoning steps in the message.
 *
 * @param messageId - The message ID to compute aggregates for
 * @returns Aggregate feedback counts and sentiment indicator
 */
export function useFeedbackAggregates(messageId: MessageId): FeedbackAggregates {
  // Get message data
  const getMessageToolCalls = useConversationStore((state) => state.getMessageToolCalls);
  const getMessageMemoryTraces = useConversationStore((state) => state.getMessageMemoryTraces);

  // Get feedback store
  const getVoteCounts = useFeedbackStore((state) => state.getVoteCounts);

  // Get all related entities
  const toolCalls = getMessageToolCalls(messageId);
  const memoryTraces = getMessageMemoryTraces(messageId);

  // Compute aggregates
  const aggregates = useMemo(() => {
    // Aggregate tool use feedback
    const toolUseFeedback = toolCalls.reduce(
      (acc, toolCall) => {
        const counts = getVoteCounts('tool_use', toolCall.id);
        return {
          upvotes: acc.upvotes + counts.up,
          downvotes: acc.downvotes + counts.down,
          critical: acc.critical + counts.critical,
        };
      },
      { upvotes: 0, downvotes: 0, critical: 0 }
    );

    // Aggregate memory feedback
    const memoryFeedback = memoryTraces.reduce(
      (acc, memory) => {
        const counts = getVoteCounts('memory', memory.id);
        return {
          upvotes: acc.upvotes + counts.up,
          downvotes: acc.downvotes + counts.down,
          critical: acc.critical + counts.critical,
        };
      },
      { upvotes: 0, downvotes: 0, critical: 0 }
    );

    // Note: reasoning steps are not yet in conversationStore
    // Placeholder for future implementation
    const reasoningFeedback = { upvotes: 0, downvotes: 0, critical: 0 };

    // Calculate totals
    const totalPositive =
      toolUseFeedback.upvotes +
      memoryFeedback.upvotes +
      reasoningFeedback.upvotes;

    const totalNegative =
      toolUseFeedback.downvotes +
      memoryFeedback.downvotes +
      reasoningFeedback.downvotes +
      toolUseFeedback.critical +
      memoryFeedback.critical +
      reasoningFeedback.critical;

    const totalVotes = totalPositive + totalNegative;

    // Calculate sentiment indicator
    let sentiment: SentimentIndicator = 'neutral';

    if (totalVotes === 0) {
      sentiment = 'neutral';
    } else {
      const positiveRatio = totalPositive / totalVotes;

      if (positiveRatio >= 0.8) {
        sentiment = 'positive'; // Green: mostly positive
      } else if (positiveRatio >= 0.4) {
        sentiment = 'mixed'; // Yellow: mixed feedback
      } else {
        sentiment = 'negative'; // Red: mostly negative
      }
    }

    return {
      toolUseFeedback,
      memoryFeedback,
      reasoningFeedback,
      sentiment,
      totalVotes,
      totalPositive,
      totalNegative,
    };
  }, [toolCalls, memoryTraces, getVoteCounts]);

  return aggregates;
}
