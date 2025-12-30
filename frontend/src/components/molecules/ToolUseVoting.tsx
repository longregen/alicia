import React, { useState } from 'react';
import FeedbackControls from '../atoms/FeedbackControls';
import GhostButton from '../atoms/GhostButton';
import { useFeedback } from '../../hooks/useFeedback';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
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
    <div className={cls(CSS.flex, CSS.flexCol, CSS.gap2, className)}>
      {/* Main voting controls */}
      <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2)}>
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
          <GhostButton
            size="sm"
            onClick={toggleFeedbackOptions}
            ariaLabel="Show quick feedback options"
          >
            {currentQuickFeedback ? '✓ Feedback' : '+ Feedback'}
          </GhostButton>
        )}
      </div>

      {/* Quick feedback options */}
      {showQuickFeedback && showFeedbackOptions && (
        <div className={cls(
          CSS.flex,
          CSS.flexCol,
          CSS.gap2,
          CSS.p3,
          CSS.bgSurfaceBg,
          CSS.roundedLg,
          CSS.border,
          'border-gray-300 dark:border-gray-600'
        )}>
          <div className={cls(CSS.textXs, CSS.textMuted, CSS.fontMedium)}>
            What's your feedback?
          </div>
          <div className={cls(CSS.flex, 'flex-wrap', CSS.gap2)}>
            {QUICK_FEEDBACK_OPTIONS.map((option) => (
              <button
                key={option.id}
                onClick={() => handleQuickFeedback(option.id)}
                disabled={isLoading}
                className={cls(
                  CSS.flex,
                  CSS.itemsCenter,
                  CSS.gap1,
                  'px-2 py-1',
                  CSS.textXs,
                  CSS.rounded,
                  CSS.border,
                  CSS.transitionAll,
                  CSS.duration200,
                  currentQuickFeedback === option.id
                    ? 'bg-primary-blue-glow text-primary-blue border-primary-blue'
                    : 'bg-surface-bg text-muted-text border-gray-300 dark:border-gray-600 hover:border-primary-blue hover:text-primary-text',
                  isLoading ? CSS.cursorNotAllowed : CSS.cursorPointer
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
