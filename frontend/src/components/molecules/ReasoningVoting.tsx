import React from 'react';
import FeedbackControls from '../atoms/FeedbackControls';
import { useFeedback } from '../../hooks/useFeedback';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type { BaseComponentProps } from '../../types/components';

/**
 * ReasoningVoting molecule component for voting on reasoning steps.
 * Simple voting interface with helpful/unhelpful context.
 */

export interface ReasoningVotingProps extends BaseComponentProps {
  /** Unique ID of the reasoning step */
  reasoningId: string;
  /** Compact mode for inline display */
  compact?: boolean;
  /** Show label text */
  showLabel?: boolean;
}

const ReasoningVoting: React.FC<ReasoningVotingProps> = ({
  reasoningId,
  compact = false,
  showLabel = true,
  className = '',
}) => {
  const {
    currentVote,
    vote,
    counts,
    isLoading,
  } = useFeedback('reasoning', reasoningId);

  return (
    <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2, className)}>
      {/* Label */}
      {showLabel && (
        <span className={cls(CSS.textXs, CSS.textMuted)}>
          Was this reasoning helpful?
        </span>
      )}

      {/* Voting controls */}
      <FeedbackControls
        currentVote={currentVote as 'up' | 'down' | null}
        onVote={vote}
        upvotes={counts.up}
        downvotes={counts.down}
        isLoading={isLoading}
        compact={compact}
      />
    </div>
  );
};

export default ReasoningVoting;
