import React, { useState } from 'react';
import FeedbackControls from '../atoms/FeedbackControls';
import Button from '../atoms/Button';
import { useFeedback } from '../../hooks/useFeedback';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';

/**
 * ToolUseVoting molecule component for voting on tool usage results.
 * Includes quick feedback options specific to tool use (helpful, incorrect, slow, etc.).
 */

export interface ToolUseVotingProps extends BaseComponentProps {
  /** Unique ID of the tool use instance */
  toolUseId: string;
  /** Compact mode for inline display */
  compact?: boolean;
  /** Show quick feedback options */
  showQuickFeedback?: boolean;
}

const QUICK_FEEDBACK_OPTIONS = [
  { id: 'helpful', label: 'Helpful', emoji: '👍' },
  { id: 'incorrect', label: 'Incorrect result', emoji: '❌' },
  { id: 'slow', label: 'Too slow', emoji: '🐌' },
  { id: 'unnecessary', label: 'Unnecessary', emoji: '🤔' },
] as const;

const ToolUseVoting: React.FC<ToolUseVotingProps> = ({
  toolUseId,
  compact = false,
  showQuickFeedback = true,
  className = '',
}) => {
  const [showFeedbackOptions, setShowFeedbackOptions] = useState(false);
  const {
    currentVote,
    vote,
    counts,
    isLoading,
    setQuickFeedback,
    currentQuickFeedback,
  } = useFeedback('tool_use', toolUseId);

  const handleQuickFeedback = async (feedbackId: string) => {
    await setQuickFeedback(feedbackId);
    setShowFeedbackOptions(false);
  };

  const toggleFeedbackOptions = () => {
    setShowFeedbackOptions(!showFeedbackOptions);
  };

  return (
    <div className={cls('flex flex-col gap-2', className)}>
      {/* Main voting controls */}
      <div className="flex items-center gap-2">
        <FeedbackControls
          currentVote={currentVote as 'up' | 'down' | null}
          onVote={vote}
          upvotes={counts.up}
          downvotes={counts.down}
          isLoading={isLoading}
          compact={compact}
        />

        {/* Quick feedback toggle button */}
        {showQuickFeedback && !compact && (
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleFeedbackOptions}
            aria-label="Show quick feedback options"
          >
            {currentQuickFeedback ? '✓ Feedback' : '+ Feedback'}
          </Button>
        )}
      </div>

      {/* Quick feedback options */}
      {showQuickFeedback && showFeedbackOptions && (
        <div className="flex flex-col gap-2 p-3 bg-surface rounded-lg border border-muted">
          <div className="text-xs text-muted font-medium">
            What's your feedback?
          </div>
          <div className="flex flex-wrap gap-2">
            {QUICK_FEEDBACK_OPTIONS.map((option) => (
              <button
                key={option.id}
                onClick={() => handleQuickFeedback(option.id)}
                disabled={isLoading}
                className={cls(
                  'flex items-center gap-1 px-2 py-1 text-xs rounded border transition-all duration-200',
                  currentQuickFeedback === option.id
                    ? 'bg-accent-subtle text-accent border-accent'
                    : 'bg-surface text-muted border hover:border-accent hover:text-default',
                  isLoading ? 'cursor-not-allowed' : 'cursor-pointer'
                )}
              >
                <span>{option.emoji}</span>
                <span>{option.label}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default ToolUseVoting;
