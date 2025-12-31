import React, { useState } from 'react';
import StatusBadge from '../atoms/StatusBadge';
import ToolUseVoting from './ToolUseVoting';
import Button from '../atoms/Button';
import { cls } from '../../utils/cls';
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
  const getStatusType = (status?: string): 'running' | 'completed' | 'error' | 'idle' => {
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
  const shouldTruncate = hasResult && (tool.result?.length ?? 0) > previewLength;
  const previewResult = shouldTruncate
    ? tool.result?.slice(0, previewLength) + '...'
    : tool.result;

  return (
    <div
      className={cls(
        'layout-stack-gap p-3 rounded-lg border',
        'bg-tool-use border',
        className
      )}
    >
      {/* Header */}
      <div className="layout-between-gap">
        <div className="layout-center-gap flex-1 min-w-0">
          {/* Tool icon */}
          <div className="flex-shrink-0 text-lg">🔧</div>

          {/* Tool name and description */}
          <div className="flex flex-col gap-1 flex-1 min-w-0">
            <div className="layout-center-gap">
              <span className="text-sm font-semibold text-tool-use">
                {tool.name}
              </span>
              {tool.status && (
                <StatusBadge
                  status={getStatusType(tool.status)}
                  className="rounded-full"
                />
              )}
            </div>
            {tool.description && (
              <p className="text-xs text-muted truncate">
                {tool.description}
              </p>
            )}
          </div>
        </div>

        {/* Expand/collapse button */}
        {hasResult && (
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleExpanded}
            aria-label={isExpanded ? 'Collapse result' : 'Expand result'}
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
          </Button>
        )}
      </div>

      {/* Result section */}
      {hasResult && (
        <div className="layout-stack-gap">
          {/* Result content */}
          <div className="p-3 rounded-lg bg-tool-result border">
            <pre className="text-xs text-tool-result whitespace-pre-wrap font-mono">
              {isExpanded || !shouldTruncate ? tool.result : previewResult}
            </pre>
          </div>

          {/* Show more/less button */}
          {shouldTruncate && (
            <Button
              variant="ghost"
              size="sm"
              onClick={toggleExpanded}
              aria-label={isExpanded ? 'Show less' : 'Show more'}
            >
              {isExpanded ? 'Show less' : 'Show more'}
            </Button>
          )}
        </div>
      )}

      {/* Voting controls */}
      {showVoting && tool.id && (
        <div className="pt-2 border-t border-muted">
          <ToolUseVoting toolUseId={tool.id} compact showQuickFeedback />
        </div>
      )}
    </div>
  );
};

export default ToolUseCard;
