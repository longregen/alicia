import React, { useState } from 'react';
import StatusBadge from '../atoms/StatusBadge';
import ToolUseVoting from './ToolUseVoting';
import GhostButton from '../atoms/GhostButton';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type { BaseComponentProps, ToolData } from '../../types/components';

/**
 * ToolUseCard molecule component for displaying tool usage with integrated voting.
 * Supports expand/collapse for detailed results and includes voting controls.
 */

export interface ToolUseCardProps extends BaseComponentProps {
  /** Tool data including name, description, status, and result */
  tool: ToolData;
  /** Initial expanded state */
  defaultExpanded?: boolean;
  /** Show voting controls */
  showVoting?: boolean;
}

const ToolUseCard: React.FC<ToolUseCardProps> = ({
  tool,
  defaultExpanded = false,
  showVoting = true,
  className = '',
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded);
  };

  // Map tool status to badge status
  const getStatusType = (status?: string) => {
    switch (status) {
      case 'running':
        return 'running';
      case 'completed':
        return 'completed';
      case 'error':
        return 'error';
      default:
        return 'idle';
    }
  };

  const hasResult = tool.result && tool.result.length > 0;
  const previewLength = 150;
  const shouldTruncate = hasResult && tool.result!.length > previewLength;
  const previewResult = shouldTruncate
    ? tool.result!.slice(0, previewLength) + '...'
    : tool.result;

  return (
    <div
      className={cls(
        CSS.flex,
        CSS.flexCol,
        CSS.gap2,
        CSS.p3,
        CSS.roundedLg,
        CSS.border,
        'bg-tool-use border-gray-300 dark:border-gray-600',
        className
      )}
    >
      {/* Header */}
      <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyBetween, CSS.gap2)}>
        <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2, 'flex-1', CSS.minW0)}>
          {/* Tool icon */}
          <div className="flex-shrink-0 text-lg">🔧</div>

          {/* Tool name and description */}
          <div className={cls(CSS.flex, CSS.flexCol, CSS.gap1, 'flex-1', CSS.minW0)}>
            <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2)}>
              <span className={cls(CSS.textSm, CSS.fontSemibold, CSS.textPrimary)}>
                {tool.name}
              </span>
              {tool.status && (
                <StatusBadge status={getStatusType(tool.status)} size="sm" />
              )}
            </div>
            {tool.description && (
              <p className={cls(CSS.textXs, CSS.textMuted, 'truncate')}>
                {tool.description}
              </p>
            )}
          </div>
        </div>

        {/* Expand/collapse button */}
        {hasResult && (
          <GhostButton
            size="sm"
            onClick={toggleExpanded}
            ariaLabel={isExpanded ? 'Collapse result' : 'Expand result'}
          >
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
          </GhostButton>
        )}
      </div>

      {/* Result section */}
      {hasResult && (
        <div className={cls(CSS.flex, CSS.flexCol, CSS.gap2)}>
          {/* Result content */}
          <div
            className={cls(
              CSS.p3,
              CSS.roundedLg,
              'bg-tool-result',
              CSS.border,
              'border-gray-300 dark:border-gray-600'
            )}
          >
            <pre className={cls(CSS.textXs, CSS.textPrimary, 'whitespace-pre-wrap', 'font-mono')}>
              {isExpanded || !shouldTruncate ? tool.result : previewResult}
            </pre>
          </div>

          {/* Show more/less button */}
          {shouldTruncate && (
            <GhostButton
              size="sm"
              onClick={toggleExpanded}
              ariaLabel={isExpanded ? 'Show less' : 'Show more'}
            >
              {isExpanded ? 'Show less' : 'Show more'}
            </GhostButton>
          )}
        </div>
      )}

      {/* Voting controls */}
      {showVoting && tool.id && (
        <div className={cls('pt-2', 'border-t', 'border-gray-300 dark:border-gray-600')}>
          <ToolUseVoting toolUseId={tool.id} compact showQuickFeedback />
        </div>
      )}
    </div>
  );
};

export default ToolUseCard;
