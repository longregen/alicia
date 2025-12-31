import React from 'react';
import FeedbackControls from '../atoms/FeedbackControls';
import ScoreBadge from '../atoms/ScoreBadge';
import { useFeedback } from '../../hooks/useFeedback';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';

/**
 * MemoryVoting molecule component for voting on memory relevance.
 * Displays relevance score alongside voting controls.
 */

export interface MemoryVotingProps extends BaseComponentProps {
  /** Unique ID of the memory */
  memoryId: string;
  /** Relevance score (0-1) */
  relevance?: number;
  /** Compact mode for inline display */
  compact?: boolean;
  /** Show relevance score */
  showRelevance?: boolean;
}

const MemoryVoting: React.FC<MemoryVotingProps> = ({
  memoryId,
  relevance,
  compact = false,
  showRelevance = true,
  className = '',
}) => {
  const {
    currentVote,
    vote,
    counts,
    isLoading,
  } = useFeedback('memory', memoryId);

  return (
    <div className={cls('flex items-center gap-3', className)}>
      {/* Relevance score */}
      {showRelevance && relevance !== undefined && (
        <div className="flex items-center gap-1">
          <span className="text-xs text-muted">Relevance:</span>
          <ScoreBadge
            score={relevance}
            max={1}
            showPercent
            size="sm"
            thresholds={{
              error: 30,
              warning: 60,
              success: 80,
            }}
          />
        </div>
      )}

      {/* Voting controls */}
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted">Relevant?</span>
        <FeedbackControls
          currentVote={currentVote as 'up' | 'down' | null}
          onVote={vote}
          upvotes={counts.up}
          downvotes={counts.down}
          isLoading={isLoading}
          compact={compact}
        />
      </div>
    </div>
  );
};

export default MemoryVoting;
