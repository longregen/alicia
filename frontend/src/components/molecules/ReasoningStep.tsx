import React, { useState } from 'react';
import ReasoningVoting from './ReasoningVoting';
import GhostButton from '../atoms/GhostButton';
import { cls } from '../../utils/cls';
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
        'flex flex-col gap-2 p-3 rounded-lg',
        'bg-reasoning border-l-4 border-accent',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-2">
        <button
          onClick={toggleExpanded}
          className={cls(
            'flex items-center gap-2 text-xs font-medium',
            'text-reasoning hover:text-accent',
            'transition-colors duration-200 cursor-pointer'
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
      <div className="text-sm text-default whitespace-pre-wrap">
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
        <div className="pt-2 border-t border-muted">
          <ReasoningVoting reasoningId={id} compact />
        </div>
      )}
    </div>
  );
};

export default ReasoningStep;
