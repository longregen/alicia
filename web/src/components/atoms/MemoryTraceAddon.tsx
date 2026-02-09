import React from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';
import FeedbackControls from './FeedbackControls';
import { useFeedback } from '../../hooks/useFeedback';
import type { VoteType } from '../../hooks/useFeedback';
import type { MemoryTrace } from '../../types/chat';
import { HoverPopover } from './HoverPopover';

// Component props interface
export interface MemoryTraceAddonProps extends BaseComponentProps {
  /** Array of memory traces to display */
  traces: MemoryTrace[];
}

// Separate component for popover content to use hooks
interface MemoryPopoverContentProps {
  trace: MemoryTrace;
  percentage: number;
  relevanceBgColor: string;
  relevanceColor: string;
}

const MemoryPopoverContent: React.FC<MemoryPopoverContentProps> = ({
  trace,
  percentage,
  relevanceBgColor,
  relevanceColor,
}) => {
  const feedback = useFeedback('memory_usage', trace.id);

  // Map VoteType to UI vote type (critical maps to down for UI display)
  const currentUiVote: 'up' | 'down' | null =
    feedback.currentVote === 'critical' ? 'down' : feedback.currentVote;

  const handleVote = (vote: VoteType) => {
    feedback.vote(vote);
  };

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
      <div
        className={cls(
          'text-sm text-default leading-relaxed',
          'max-h-48 overflow-y-auto',
          'pr-2 custom-scrollbar'
        )}
      >
        {trace.content}
      </div>

      {/* Footer with feedback controls and metadata */}
      <div
        className={cls(
          'flex items-center justify-between',
          'pt-3 border-t border-border-muted'
        )}
      >
        <div className="text-[11px] text-muted">Memory ID: {trace.id}</div>
        <FeedbackControls
          currentVote={currentUiVote}
          onVote={handleVote}
          upvotes={feedback.counts.up}
          downvotes={feedback.counts.down + feedback.counts.critical}
          isLoading={feedback.isLoading}
          compact
        />
      </div>
    </div>
  );
};

const MemoryTraceAddon: React.FC<MemoryTraceAddonProps> = ({ traces, className = '' }) => {
  // Sort traces by relevance (highest first)
  const sortedTraces = [...traces].sort((a, b) => b.relevance - a.relevance);

  // Convert relevance score to percentage
  const getRelevancePercentage = (relevance: number): number => {
    return Math.round(relevance * 100);
  };

  // Get color class based on relevance threshold
  const getRelevanceColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'text-success';
    if (relevance >= 0.4) return 'text-accent';
    return 'text-muted';
  };

  // Get background color class based on relevance threshold
  const getRelevanceBgColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'bg-success/10';
    if (relevance >= 0.4) return 'bg-accent-subtle';
    return 'bg-surface';
  };

  const renderMemoryBadge = (trace: MemoryTrace) => {
    const percentage = getRelevancePercentage(trace.relevance);
    const relevanceColor = getRelevanceColor(trace.relevance);
    const relevanceBgColor = getRelevanceBgColor(trace.relevance);

    return (
      <HoverPopover
        key={trace.id}
        content={
          <MemoryPopoverContent
            trace={trace}
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
            'hover:scale-105 hover:shadow-md hover:ring-2 hover:ring-accent/50',
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

  if (sortedTraces.length === 0) {
    return null;
  }

  return (
    <div className={cls('w-full', className)}>
      {/* Memory badges row */}
      <div className={cls('flex', 'items-center', 'gap-2', 'flex-wrap')}>
        {sortedTraces.map(renderMemoryBadge)}
      </div>
    </div>
  );
};

export { MemoryTraceAddon };
