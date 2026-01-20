import React, { useState, useMemo } from 'react';
import StatusBadge from '../atoms/StatusBadge';
import ToolUseVoting from './ToolUseVoting';
import Button from '../atoms/Button';
import ToolVisualizationRouter from './ToolVisualizations/ToolVisualizationRouter';
import { toolIcons, toolDisplayNames } from './ToolVisualizations';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, ToolData } from '../../types/components';

/**
 * ToolUseCard molecule component for displaying tool usage with integrated voting.
 * Supports expand/collapse for detailed results and includes voting controls.
 * Uses specialized visualizations for native tools (web_*, garden_*).
 */

// Native tools that have specialized visualizations
const NATIVE_TOOLS = [
  'web_read',
  'web_fetch_raw',
  'web_fetch_structured',
  'web_search',
  'web_extract_links',
  'web_extract_metadata',
  'web_screenshot',
  'garden_describe_table',
  'garden_execute_sql',
  'garden_schema_explore',
];

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

  // Check if this is a native tool with specialized visualization
  const isNativeTool = useMemo(() => NATIVE_TOOLS.includes(tool.name), [tool.name]);

  // Get tool icon
  const toolIcon = useMemo(() => toolIcons[tool.name] || '🔧', [tool.name]);

  // Get display name
  const displayName = useMemo(() => toolDisplayNames[tool.name] || tool.name, [tool.name]);

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

  // Parse result for native tools
  const parsedResult = useMemo(() => {
    if (!hasResult || !isNativeTool) return null;
    try {
      return JSON.parse(tool.result!);
    } catch {
      return tool.result;
    }
  }, [hasResult, isNativeTool, tool.result]);

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
          <div className="flex-shrink-0 text-lg">{toolIcon}</div>

          {/* Tool name and description */}
          <div className="flex flex-col gap-1 flex-1 min-w-0">
            <div className="layout-center-gap">
              <span className="text-sm font-semibold text-tool-use">
                {displayName}
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
          {/* Native tool visualization or generic result */}
          {isExpanded ? (
            isNativeTool && parsedResult ? (
              <ToolVisualizationRouter
                toolName={tool.name}
                result={parsedResult}
              />
            ) : (
              <div className="p-3 rounded-lg bg-tool-result border">
                <pre className="text-xs text-tool-result whitespace-pre-wrap font-mono">
                  {tool.result}
                </pre>
              </div>
            )
          ) : (
            <div className="p-3 rounded-lg bg-tool-result border">
              <pre className="text-xs text-tool-result whitespace-pre-wrap font-mono">
                {previewResult}
              </pre>
            </div>
          )}

          {/* Show more/less button */}
          {(shouldTruncate || isNativeTool) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={toggleExpanded}
              aria-label={isExpanded ? 'Show less' : 'Show more'}
            >
              {isExpanded ? 'Show less' : isNativeTool ? 'Show visualization' : 'Show more'}
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
