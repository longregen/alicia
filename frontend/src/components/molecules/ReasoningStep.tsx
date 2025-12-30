import React, { useState } from 'react';
import ReasoningVoting from './ReasoningVoting';
import GhostButton from '../atoms/GhostButton';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type { BaseComponentProps } from '../../types/components';

/**
 * ReasoningStep molecule component for displaying individual reasoning steps with voting.
 * Supports expand/collapse for longer reasoning content.
 */

export interface ReasoningStepProps extends BaseComponentProps {
  /** Unique ID for the reasoning step */
  id: string;
  /** Reasoning content */
  content: string;
  /** Step sequence number (optional) */
  sequence?: number;
  /** Initial expanded state */
  defaultExpanded?: boolean;
  /** Show voting controls */
  showVoting?: boolean;
}

const ReasoningStep: React.FC<ReasoningStepProps> = ({
  id,
  content,
  sequence,
  defaultExpanded = false,
  showVoting = true,
  className = '',
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded);
  };

  const previewLength = 100;
  const shouldTruncate = content.length > previewLength;
  const previewContent = shouldTruncate ? content.slice(0, previewLength) + '...' : content;

  return (
    <div
      className={cls(
        CSS.flex,
        CSS.flexCol,
        CSS.gap2,
        CSS.p3,
        CSS.roundedLg,
        'bg-reasoning border-l-4 border-blue-500',
        className
      )}
    >
      {/* Header */}
      <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyBetween, CSS.gap2)}>
        <button
          onClick={toggleExpanded}
          className={cls(
            CSS.flex,
            CSS.itemsCenter,
            CSS.gap2,
            CSS.textXs,
            'text-blue-600 dark:text-blue-400',
            CSS.fontMedium,
            'hover:text-blue-700 dark:hover:text-blue-300',
            CSS.transitionColors,
            CSS.duration200,
            CSS.cursorPointer
          )}
          aria-expanded={isExpanded}
          aria-label={isExpanded ? 'Collapse reasoning' : 'Expand reasoning'}
        >
          <span>
            {sequence !== undefined ? `Reasoning Step ${sequence}` : 'Reasoning'}
          </span>
          <svg
            className={cls(
              'w-4 h-4 transition-transform duration-200',
              isExpanded ? 'rotate-180' : 'rotate-0'
            )}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>
      </div>

      {/* Content */}
      <div className={cls(CSS.textSm, CSS.textPrimary, 'whitespace-pre-wrap')}>
        {isExpanded || !shouldTruncate ? content : previewContent}
      </div>

      {/* Show more button */}
      {shouldTruncate && !isExpanded && (
        <GhostButton size="sm" onClick={toggleExpanded} ariaLabel="Show more">
          Show more
        </GhostButton>
      )}

      {/* Voting controls */}
      {showVoting && isExpanded && (
        <div className={cls('pt-2', 'border-t', 'border-blue-200 dark:border-blue-800')}>
          <ReasoningVoting reasoningId={id} compact />
        </div>
      )}
    </div>
  );
};

export default ReasoningStep;
