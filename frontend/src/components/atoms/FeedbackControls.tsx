import React from 'react';
import { cls } from '../../utils/cls';
import type { VoteType } from '../../stores/feedbackStore';

/**
 * FeedbackControls atom component for voting on messages, tools, memories, and reasoning.
 *
 * Displays upvote/downvote buttons with counts and handles vote state.
 */

export interface FeedbackControlsProps {
  /** Current user vote state */
  currentVote: 'up' | 'down' | null;
  /** Callback when user votes */
  onVote: (vote: VoteType) => void;
  /** Number of upvotes */
  upvotes?: number;
  /** Number of downvotes */
  downvotes?: number;
  /** Whether the vote is being processed */
  isLoading?: boolean;
  /** Additional CSS classes */
  className?: string;
  /** Compact mode for inline display */
  compact?: boolean;
}

const FeedbackControls: React.FC<FeedbackControlsProps> = ({
  currentVote,
  onVote,
  upvotes = 0,
  downvotes = 0,
  isLoading = false,
  className = '',
  compact = false,
}) => {
  const handleUpvote = () => {
    if (!isLoading) {
      onVote('up');
    }
  };

  const handleDownvote = () => {
    if (!isLoading) {
      onVote('down');
    }
  };

  const buttonBaseClasses = cls(
    'flex items-center gap-1 rounded-md',
    'transition-all duration-200',
    isLoading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
    compact ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm'
  );

  return (
    <div className={cls('layout-center-gap', className)}>
      {/* Upvote button */}
      <button
        onClick={handleUpvote}
        disabled={isLoading}
        aria-label={currentVote === 'up' ? 'Remove upvote' : 'Upvote'}
        className={cls(
          buttonBaseClasses,
          currentVote === 'up'
            ? 'bg-success-subtle text-success border border-success'
            : 'bg-surface hover:bg-sunken text-muted hover:text-default border border-transparent hover:border'
        )}
      >
        <svg
          className={cls(compact ? 'w-3 h-3' : 'w-4 h-4')}
          fill={currentVote === 'up' ? 'currentColor' : 'none'}
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M14 10h4.764a2 2 0 011.789 2.894l-3.5 7A2 2 0 0115.263 21h-4.017c-.163 0-.326-.02-.485-.06L7 20m7-10V5a2 2 0 00-2-2h-.095c-.5 0-.905.405-.905.905 0 .714-.211 1.412-.608 2.006L7 11v9m7-10h-2M7 20H5a2 2 0 01-2-2v-6a2 2 0 012-2h2.5"
          />
        </svg>
        {upvotes > 0 && <span>{upvotes}</span>}
      </button>

      {/* Downvote button */}
      <button
        onClick={handleDownvote}
        disabled={isLoading}
        aria-label={currentVote === 'down' ? 'Remove downvote' : 'Downvote'}
        className={cls(
          buttonBaseClasses,
          currentVote === 'down'
            ? 'bg-error-subtle text-error border border-error'
            : 'bg-surface hover:bg-sunken text-muted hover:text-default border border-transparent hover:border'
        )}
      >
        <svg
          className={cls(compact ? 'w-3 h-3' : 'w-4 h-4')}
          fill={currentVote === 'down' ? 'currentColor' : 'none'}
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M10 14H5.236a2 2 0 01-1.789-2.894l3.5-7A2 2 0 018.736 3h4.018a2 2 0 01.485.06l3.76.94m-7 10v5a2 2 0 002 2h.096c.5 0 .905-.405.905-.904 0-.715.211-1.413.608-2.008L17 13V4m-7 10h2m5-10h2a2 2 0 012 2v6a2 2 0 01-2 2h-2.5"
          />
        </svg>
        {downvotes > 0 && <span>{downvotes}</span>}
      </button>

      {/* Loading indicator */}
      {isLoading && (
        <div className="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin" />
      )}
    </div>
  );
};

export default FeedbackControls;
