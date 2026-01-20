import React, { useState, useEffect } from 'react';
import AudioAddon from './AudioAddon';
import FeedbackControls from './FeedbackControls';
import BranchNavigator from './BranchNavigator';
import { HoverPopover } from './HoverPopover';
import { AUDIO_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { flexCenterGap, uiPatterns } from '../../utils/uiPatterns';
import { useFeedback } from '../../hooks/useFeedback';
import type { BaseComponentProps, MessageAddon, AudioState, MemoryAddonData } from '../../types/components';

// Tool details interface
export interface ToolDetail {
  id: string;
  name: string;
  description: string;
  result?: string;
  status?: 'pending' | 'running' | 'completed' | 'error';
}

// Helper to format web search - compact inline format
const formatWebSearchCompact = (toolName: string, args: string, result: string): React.ReactNode => {
  let query = '';
  let limit = 5;
  let results: Array<{ title?: string; url?: string }> = [];

  // Parse arguments - strip "Arguments: " prefix if present
  try {
    const argsJson = args.replace(/^Arguments:\s*/i, '');
    const parsedArgs = JSON.parse(argsJson);
    query = parsedArgs.query || '';
    limit = parsedArgs.limit || 5;
  } catch { /* ignore */ }

  // Parse results
  try {
    const parsedResult = JSON.parse(result);
    results = parsedResult.results || [];
  } catch { /* ignore */ }

  const resultCount = results.length;

  return (
    <div className="space-y-2">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">{toolName}:</span> <span className="text-accent">"{query}"</span> <span className="text-muted-foreground/60">(limit {limit})</span>
      </div>
      <div className="text-xs">
        <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({resultCount})</span>
      </div>
      {results.length > 0 && (
        <div className="space-y-1.5 pl-1">
          {results.slice(0, 6).map((item, index) => (
            <div key={index}>
              <div className="text-xs font-medium text-default truncate">
                • {item.title || 'Untitled'}
              </div>
              {item.url && (
                <a
                  href={item.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block text-[10px] text-accent/70 truncate pl-2.5 hover:text-accent hover:underline"
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                >
                  {item.url}
                </a>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Helper to format web search results in a user-friendly way (fallback)
const formatWebSearchResult = (parsed: Record<string, unknown>): React.ReactNode => {
  const results = parsed.results as Array<{ title?: string; url?: string; snippet?: string }> | undefined;
  const resultCount = parsed.result_count as number | undefined;

  return (
    <div className="space-y-1.5">
      {results && results.length > 0 && (
        <>
          <div className="text-xs text-success flex items-center gap-1.5 mb-2">
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            Results ({resultCount || results.length}):
          </div>
          <div className="space-y-1.5 pl-1">
            {results.slice(0, 6).map((item, index) => (
              <div key={index}>
                <div className="text-xs font-medium text-default truncate">
                  • {item.title || 'Untitled'}
                </div>
                {item.url && (
                  <div className="text-[10px] text-accent/70 truncate pl-2.5">
                    {item.url}
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

// Helper to format memory - compact inline format
const formatMemoryCompact = (toolName: string, args: string, result: string): React.ReactNode => {
  let query = '';
  let limit = 5;
  let memories: Array<{ id?: string; content?: string; similarity?: number; relevance?: number }> = [];

  // Parse arguments - strip "Arguments: " prefix if present
  try {
    const argsJson = args.replace(/^Arguments:\s*/i, '');
    const parsedArgs = JSON.parse(argsJson);
    query = parsedArgs.query || parsedArgs.text || '';
    limit = parsedArgs.limit || parsedArgs.k || 5;
  } catch { /* ignore */ }

  // Parse results
  try {
    const parsedResult = JSON.parse(result);
    memories = parsedResult.memories || parsedResult || [];
  } catch { /* ignore */ }

  const resultCount = memories.length;

  return (
    <div className="space-y-2">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">{toolName}:</span> <span className="text-accent">"{query}"</span> <span className="text-muted-foreground/60">(limit {limit})</span>
      </div>
      <div className="text-xs">
        <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({resultCount})</span>
      </div>
      {memories.length > 0 && (
        <div className="space-y-1.5 pl-1">
          {memories.slice(0, 6).map((item, index) => (
            <div key={item.id || index}>
              {item.id ? (
                <a
                  href={`/memory/${item.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block text-xs text-default hover:text-accent hover:underline"
                  onClick={(e) => e.stopPropagation()}
                >
                  • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                </a>
              ) : (
                <div className="text-xs text-default">
                  • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                </div>
              )}
              {(item.similarity !== undefined || item.relevance !== undefined) && (
                <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                  {Math.round((item.similarity ?? item.relevance ?? 0) * 100)}% match
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Helper to format memory query results (fallback)
const formatMemoryQueryResult = (parsed: Record<string, unknown>): React.ReactNode => {
  const memories = parsed.memories as Array<{ id?: string; content?: string; similarity?: number }> | undefined;
  const count = parsed.count as number | undefined;

  return (
    <div className="space-y-1.5">
      {memories && memories.length > 0 && (
        <>
          <div className="text-xs">
            <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({count || memories.length})</span>
          </div>
          <div className="space-y-1.5 pl-1">
            {memories.slice(0, 6).map((item, index) => (
              <div key={item.id || index}>
                {item.id ? (
                  <a
                    href={`/memory/${item.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block text-xs text-default hover:text-accent hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                  </a>
                ) : (
                  <div className="text-xs text-default">
                    • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                  </div>
                )}
                {item.similarity !== undefined && (
                  <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                    {Math.round(item.similarity * 100)}% match
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

// Helper to parse and format tool arguments
const formatToolArguments = (description: string): React.ReactNode => {
  // Try to parse description as JSON (it might contain arguments)
  try {
    const parsed = JSON.parse(description);

    // Format known argument types nicely
    if (typeof parsed === 'object' && parsed !== null) {
      const entries = Object.entries(parsed);
      if (entries.length === 0) {
        return <span className="text-muted text-xs italic">No arguments</span>;
      }

      return (
        <div className="space-y-1">
          {entries.map(([key, value]) => (
            <div key={key} className="flex items-start gap-2 text-xs">
              <span className="text-muted font-medium min-w-[60px]">{key}:</span>
              <span className="text-default">
                {typeof value === 'string' ? `"${value}"` : JSON.stringify(value)}
              </span>
            </div>
          ))}
        </div>
      );
    }

    return <span className="text-default text-xs">{description}</span>;
  } catch {
    // Not JSON, return as plain text
    return <span className="text-default text-xs">{description}</span>;
  }
};

// Helper to parse and format tool results
const formatToolResult = (result: string, toolName: string): React.ReactNode => {
  // Try to parse as JSON for better formatting
  try {
    const parsed = JSON.parse(result);
    const lowerToolName = toolName.toLowerCase();

    // Handle web search results
    if (lowerToolName.includes('web_search') || lowerToolName.includes('search')) {
      if (parsed.results && Array.isArray(parsed.results)) {
        return formatWebSearchResult(parsed);
      }
    }

    // Handle memory query results
    if (lowerToolName.includes('memory')) {
      if (parsed.memories && Array.isArray(parsed.memories)) {
        return formatMemoryQueryResult(parsed);
      }
      // Handle array format for memory results
      if (Array.isArray(parsed)) {
        return (
          <div className="space-y-1.5 pl-1">
            {parsed.slice(0, 6).map((item: { id?: string; content?: string; relevance?: number; similarity?: number }, index: number) => (
              <div key={item.id || index}>
                {item.id ? (
                  <a
                    href={`/memory/${item.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block text-xs text-default hover:text-accent hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                  </a>
                ) : (
                  <div className="text-xs text-default">
                    • {item.content ? (item.content.length > 80 ? item.content.substring(0, 80) + '...' : item.content) : 'No content'}
                  </div>
                )}
                {(item.relevance !== undefined || item.similarity !== undefined) && (
                  <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                    {Math.round((item.relevance ?? item.similarity ?? 0) * 100)}% match
                  </div>
                )}
              </div>
            ))}
          </div>
        );
      }
    }

    // Handle simple success/error results
    if (typeof parsed === 'object' && parsed !== null) {
      if ('success' in parsed || 'error' in parsed || 'message' in parsed) {
        return (
          <div className="space-y-1">
            {parsed.success !== undefined && (
              <div className={cls(
                'flex items-center gap-1.5 text-xs',
                parsed.success ? 'text-success' : 'text-error'
              )}>
                <span>{parsed.success ? '✓' : '✗'}</span>
                <span>{parsed.success ? 'Success' : 'Failed'}</span>
              </div>
            )}
            {parsed.message && (
              <div className="text-xs text-default">{String(parsed.message)}</div>
            )}
            {parsed.error && (
              <div className="text-xs text-error">{String(parsed.error)}</div>
            )}
          </div>
        );
      }
    }

    // Generic JSON formatting with truncation for readability
    const jsonStr = JSON.stringify(parsed, null, 2);
    const truncated = jsonStr.length > 500;
    return (
      <pre className="text-xs text-default whitespace-pre-wrap font-mono bg-surface-bg p-2 rounded max-h-48 overflow-y-auto">
        {truncated ? jsonStr.substring(0, 500) + '\n...' : jsonStr}
      </pre>
    );
  } catch {
    // Not JSON, return as plain text
    return <span className="text-default text-xs">{result}</span>;
  }
};

// Popover content component for tool details
interface ToolPopoverContentProps {
  toolDetail: ToolDetail;
  showFeedback?: boolean;
}

const ToolPopoverContent: React.FC<ToolPopoverContentProps> = ({ toolDetail, showFeedback = false }) => {
  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('tool_use', toolDetail.id);

  const lowerName = toolDetail.name.toLowerCase();
  const isWebSearch = lowerName.includes('web_search') || (lowerName.includes('search') && !lowerName.includes('memory'));
  const isMemory = lowerName.includes('memory');
  const isCompleted = toolDetail.status === 'completed';

  // Render compact format for web_search
  if (isWebSearch && isCompleted && toolDetail.result && toolDetail.description) {
    return (
      <div className="space-y-3">
        <div className="max-h-64 overflow-y-auto pr-1 custom-scrollbar">
          {formatWebSearchCompact(toolDetail.name, toolDetail.description, toolDetail.result)}
        </div>
        {showFeedback && (
          <div className="pt-3 border-t border-border-muted">
            <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
            <FeedbackControls
              currentVote={currentVote as 'up' | 'down' | null}
              onVote={vote}
              upvotes={counts.up}
              downvotes={counts.down}
              isLoading={feedbackLoading}
              compact
            />
          </div>
        )}
      </div>
    );
  }

  // Render compact format for memory
  if (isMemory && isCompleted && toolDetail.result && toolDetail.description) {
    return (
      <div className="space-y-3">
        <div className="max-h-64 overflow-y-auto pr-1 custom-scrollbar">
          {formatMemoryCompact(toolDetail.name, toolDetail.description, toolDetail.result)}
        </div>
        {showFeedback && (
          <div className="pt-3 border-t border-border-muted">
            <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
            <FeedbackControls
              currentVote={currentVote as 'up' | 'down' | null}
              onVote={vote}
              upvotes={counts.up}
              downvotes={counts.down}
              isLoading={feedbackLoading}
              compact
            />
          </div>
        )}
      </div>
    );
  }

  // Generic tool content
  return (
    <div className="space-y-3">
      {/* Header with tool name */}
      <div>
        <span className="text-sm font-medium text-muted-foreground font-mono">{toolDetail.name}</span>
      </div>

      {/* Running indicator */}
      {toolDetail.status === 'running' && (
        <div className="text-xs text-accent flex items-center gap-1.5">
          <div className="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin" />
          Processing...
        </div>
      )}

      {/* Arguments section */}
      {toolDetail.description && (
        <div>
          <div className="flex items-center gap-1.5 text-xs text-muted font-medium mb-2">
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
            </svg>
            Arguments
          </div>
          <div className="rounded-md px-3 py-2 bg-black/20 text-muted-foreground">
            {formatToolArguments(toolDetail.description)}
          </div>
        </div>
      )}

      {/* Results section */}
      {toolDetail.result && (toolDetail.status === 'completed' || toolDetail.status === 'error') && (
        <div>
          <div className={cls(
            'flex items-center gap-1.5 text-xs font-medium mb-2',
            toolDetail.status === 'error' ? 'text-error' : 'text-success'
          )}>
            {toolDetail.status === 'error' ? (
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            ) : (
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            )}
            {toolDetail.status === 'error' ? 'Error' : 'Result'}
          </div>
          <div className="max-h-48 overflow-y-auto pr-1 custom-scrollbar">
            {formatToolResult(toolDetail.result, toolDetail.name)}
          </div>
        </div>
      )}

      {/* Pending state */}
      {toolDetail.status === 'pending' && (
        <div className="text-xs text-warning flex items-center gap-1.5">
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          Waiting to execute...
        </div>
      )}

      {/* Feedback controls for completed tools */}
      {showFeedback && toolDetail.status === 'completed' && (
        <div className="pt-3 border-t border-border-muted">
          <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
          <FeedbackControls
            currentVote={currentVote as 'up' | 'down' | null}
            onVote={vote}
            upvotes={counts.up}
            downvotes={counts.down}
            isLoading={feedbackLoading}
            compact
          />
        </div>
      )}
    </div>
  );
};

// Popover content component for memory
interface MemoryPopoverContentProps {
  memory: MemoryAddonData;
  percentage: number;
  relevanceBgColor: string;
  relevanceColor: string;
}

const MemoryPopoverContent: React.FC<MemoryPopoverContentProps> = ({
  memory,
  percentage,
  relevanceBgColor,
  relevanceColor,
}) => {
  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">🧠</span>
          <span className="text-sm font-semibold text-accent">Memory Trace</span>
        </div>
        <div
          className={cls(
            'text-xs px-2 py-1 rounded-full font-semibold',
            relevanceBgColor,
            relevanceColor
          )}
        >
          {percentage}% relevant
        </div>
      </div>

      {/* Content */}
      <div className="text-sm text-default leading-relaxed max-h-48 overflow-y-auto pr-2 custom-scrollbar">
        {memory.content}
      </div>

      {/* Footer */}
      <div className="text-[11px] text-muted pt-3 border-t border-border-muted">
        Memory ID: {memory.id}
      </div>
    </div>
  );
};

// Branch navigation data interface
export interface BranchData {
  currentIndex: number;
  totalBranches: number;
  onNavigate: (direction: 'prev' | 'next') => void;
}

// Component props interface
export interface ComplexAddonsProps extends BaseComponentProps {
  /** Array of addons to display */
  addons: MessageAddon[];
  /** Tool details for expandable tools */
  toolDetails?: ToolDetail[];
  /** Message timestamp */
  timestamp: Date;
  /** Whether to show feedback controls (default: false) */
  showFeedback?: boolean;
  /** Branch navigation data (shows navigator when totalBranches > 1) */
  branchData?: BranchData;
}

const ComplexAddons: React.FC<ComplexAddonsProps> = ({
  addons,
  toolDetails = [],
  timestamp,
  className = '',
  showFeedback = false,
  branchData,
}) => {
  const [audioState, setAudioState] = useState<AudioState>(AUDIO_STATES.IDLE);
  const [audioCurrentTime, setAudioCurrentTime] = useState(0);

  // Mock audio duration - in a real app this would come from the audio file
  const audioDuration = 45; // seconds

  // Simple audio simulation for demo
  useEffect(() => {
    if (audioState === AUDIO_STATES.PLAYING) {
      const interval = setInterval(() => {
        setAudioCurrentTime(prev => {
          if (prev >= audioDuration) {
            setAudioState(AUDIO_STATES.IDLE);
            return 0;
          }
          return prev + 0.1;
        });
      }, 100);
      return () => clearInterval(interval);
    }
  }, [audioState, audioDuration]);

  const getToolDetail = (addonId: string): ToolDetail | undefined => {
    return toolDetails.find(tool => tool.id === addonId);
  };

  const getAddonAnimation = (_addon: MessageAddon, toolDetail?: ToolDetail) => {
    if (toolDetail?.status === 'running') {
      return `${uiPatterns.pulseAnimation} scale-110`;
    }
    if (toolDetail?.status === 'pending') {
      return `${uiPatterns.pulseAnimation} opacity-70`;
    }
    if (toolDetail?.status === 'error') {
      return 'text-error';
    }
    return '';
  };

  // Helper functions for memory badges
  const getRelevancePercentage = (relevance: number): number => Math.round(relevance * 100);

  const getRelevanceColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'text-success';
    if (relevance >= 0.4) return 'text-accent';
    return 'text-muted';
  };

  const getRelevanceBgColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'bg-success/10';
    if (relevance >= 0.4) return 'bg-accent-subtle';
    return 'bg-surface';
  };

  const renderMemoryBadge = (memory: MemoryAddonData) => {
    const percentage = getRelevancePercentage(memory.relevance);
    const relevanceColor = getRelevanceColor(memory.relevance);
    const relevanceBgColor = getRelevanceBgColor(memory.relevance);

    return (
      <HoverPopover
        key={memory.id}
        content={
          <MemoryPopoverContent
            memory={memory}
            percentage={percentage}
            relevanceBgColor={relevanceBgColor}
            relevanceColor={relevanceColor}
          />
        }
        side="top"
        align="start"
        sideOffset={8}
        alignOffset={8}
        openDelay={150}
        closeDelay={300}
      >
        <button
          className={cls(
            'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
            'transition-all duration-200 cursor-pointer',
            'hover:scale-105 hover:ring-2 hover:ring-accent/50',
            relevanceBgColor,
            relevanceColor
          )}
          title={`Memory trace (${percentage}% relevant)`}
        >
          <span className="text-sm">🧠</span>
          <span className="font-semibold">{percentage}%</span>
        </button>
      </HoverPopover>
    );
  };

  const renderAddon = (addon: MessageAddon) => {
    // Feedback addon - render FeedbackControls inline
    if (addon.type === 'feedback' && addon.feedbackData) {
      return (
        <FeedbackControls
          key={addon.id}
          currentVote={addon.feedbackData.currentVote}
          onVote={addon.feedbackData.onVote}
          upvotes={addon.feedbackData.upvotes}
          downvotes={addon.feedbackData.downvotes}
          isLoading={addon.feedbackData.isLoading}
          compact
        />
      );
    }

    // Memory addon - render memory badges with hover popovers
    if (addon.type === 'memory' && addon.memoryData) {
      const sortedMemories = [...addon.memoryData].sort((a, b) => b.relevance - a.relevance);
      return (
        <div key={addon.id} className="flex items-center gap-1.5">
          {sortedMemories.map(renderMemoryBadge)}
        </div>
      );
    }

    // Audio addon
    if (addon.type === 'audio') {
      return (
        <AudioAddon
          key={addon.id}
          mode="compact"
          state={audioState}
          onPlay={() => setAudioState(AUDIO_STATES.PLAYING)}
          onPause={() => setAudioState(AUDIO_STATES.PAUSED)}
          onStop={() => {
            setAudioState(AUDIO_STATES.IDLE);
            setAudioCurrentTime(0);
          }}
          duration={audioDuration}
          currentTime={audioCurrentTime}
        />
      );
    }

    // Default rendering for tool/icon addon types - with hover popover
    const toolDetail = getToolDetail(addon.id);

    // Render tool badge with hover popover
    return (
      <HoverPopover
        key={addon.id}
        content={
          toolDetail ? (
            <ToolPopoverContent toolDetail={toolDetail} showFeedback={showFeedback} />
          ) : (
            <div className="text-sm text-muted">No details available</div>
          )
        }
        side="top"
        align="start"
        sideOffset={8}
        alignOffset={8}
        openDelay={150}
        closeDelay={300}
        width="w-96"
      >
        <button
          className={cls(
            'inline-flex items-center justify-center w-7 h-7 rounded-md',
            'text-sm cursor-pointer transition-all duration-200',
            'hover:bg-surface-hover hover:ring-2 hover:ring-accent/50 border border-transparent',
            toolDetail?.status === 'running' ? 'bg-accent/10 border-accent/30' : 'bg-surface',
            toolDetail?.status === 'completed' ? 'hover:border-success/30' : '',
            toolDetail?.status === 'error' ? 'bg-error/10 border-error/30' : '',
            getAddonAnimation(addon, toolDetail)
          )}
          title={`${toolDetail?.name || addon.tooltip} - Hover for details`}
        >
          {addon.emoji}
        </button>
      </HoverPopover>
    );
  };

  return (
    <div className={cls('space-y-2 w-full', className)}>
      {/* Main addon row */}
      <div className="flex items-center justify-between w-full gap-3">
        {/* Left: All addons inline */}
        <div className={cls(flexCenterGap(2), 'flex-wrap')}>
          {addons.map(renderAddon)}
        </div>

        {/* Right: Branch navigation + Timestamp */}
        <div className="flex items-center gap-3 flex-shrink-0">
          {/* Branch Navigator */}
          {branchData && branchData.totalBranches > 1 && (
            <BranchNavigator
              currentIndex={branchData.currentIndex}
              totalBranches={branchData.totalBranches}
              onNavigate={branchData.onNavigate}
            />
          )}

          {/* Timestamp with clock icon */}
          <div className={cls(
            'flex items-center gap-1.5',
            'text-[11px] text-muted-foreground',
            'font-medium tracking-wide'
          )}>
            <svg
              className="w-3 h-3 opacity-60"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <span>{timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ComplexAddons;
