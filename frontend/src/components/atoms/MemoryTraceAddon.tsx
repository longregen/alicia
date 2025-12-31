import React, { useState } from 'react';
import { cls } from '../../utils/cls';
import type { BaseComponentProps } from '../../types/components';
import FeedbackControls from './FeedbackControls';
import { useFeedback } from '../../hooks/useFeedback';
import type { VoteType } from '../../stores/feedbackStore';

// Memory trace interface
export interface MemoryTrace {
  id: string;
  messageId: string;
  content: string;
  relevance: number; // 0-1 score
}

// Component props interface
export interface MemoryTraceAddonProps extends BaseComponentProps {
  /** Array of memory traces to display */
  traces: MemoryTrace[];
}

const MemoryTraceAddon: React.FC<MemoryTraceAddonProps> = ({
  traces,
  className = ''
}) => {
  const [expandedTraceId, setExpandedTraceId] = useState<string | null>(null);
  const [hoveredTraceId, setHoveredTraceId] = useState<string | null>(null);

  // Get feedback state for expanded trace
  const expandedTrace = expandedTraceId
    ? traces.find(t => t.id === expandedTraceId)
    : null;
  const feedback = useFeedback('memory', expandedTrace?.id || '');

  // Map VoteType to UI vote type (critical maps to down for UI display)
  const currentUiVote: 'up' | 'down' | null =
    feedback.currentVote === 'critical' ? 'down' : feedback.currentVote;

  // Handler for voting on the expanded memory
  const handleVote = (vote: VoteType) => {
    if (expandedTrace) {
      feedback.vote(vote);
    }
  };

  // Sort traces by relevance (highest first)
  const sortedTraces = [...traces].sort((a, b) => b.relevance - a.relevance);

  // Convert relevance score to percentage
  const getRelevancePercentage = (relevance: number): number => {
    return Math.round(relevance * 100);
  };

  // Get color class based on relevance threshold
  const getRelevanceColor = (relevance: number): string => {
    if (relevance >= 0.8) return 'text-accent';
    if (relevance >= 0.6) return 'text-accent';
    if (relevance >= 0.4) return 'text-accent';
    return 'text-muted';
  };

  // Get background color class based on relevance threshold
  const getRelevanceBgColor = (relevance: number): string => {
    if (relevance >= 0.8) return 'bg-accent-subtle';
    if (relevance >= 0.6) return 'bg-accent-subtle';
    if (relevance >= 0.4) return 'bg-accent-subtle';
    return 'bg-surface';
  };

  // Truncate content for preview
  const getTruncatedContent = (content: string, maxLength: number = 100): string => {
    if (content.length <= maxLength) return content;
    return content.substring(0, maxLength) + '...';
  };

  const renderMemoryBadge = (trace: MemoryTrace) => {
    const percentage = getRelevancePercentage(trace.relevance);
    const isHovered = hoveredTraceId === trace.id;
    const isExpanded = expandedTraceId === trace.id;

    return (
      <div key={trace.id} className="relative">
        <button
          className={cls(
            'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
            'transition-all duration-200 cursor-pointer',
            'hover:scale-105 hover:shadow-md',
            getRelevanceBgColor(trace.relevance),
            getRelevanceColor(trace.relevance),
            isExpanded ? 'ring-2 ring-accent scale-105' : ''
          )}
          onMouseEnter={() => setHoveredTraceId(trace.id)}
          onMouseLeave={() => setHoveredTraceId(null)}
          onClick={() => setExpandedTraceId(expandedTraceId === trace.id ? null : trace.id)}
          title={`Memory trace (${percentage}% relevant)`}
        >
          <span className="text-sm">🧠</span>
          <span className="font-semibold">{percentage}%</span>
        </button>

        {/* Hover tooltip */}
        {isHovered && !isExpanded && (
          <div className={cls(
            'absolute z-20 transition-all duration-200',
            'bg-overlay backdrop-blur-sm text-on-emphasis text-xs rounded-md p-3 min-w-[16rem] max-w-[20rem]',
            'pointer-events-none shadow-xl border border-border-emphasis',
            'top-full left-1/2 transform -translate-x-1/2 mt-2'
          )}>
            <div className="font-semibold text-sm mb-1.5 text-accent">
              Memory Trace ({percentage}% relevant)
            </div>
            <div className="text-subtle leading-relaxed">
              {getTruncatedContent(trace.content, 120)}
            </div>
            <div className="text-[11px] text-muted mt-2">
              Click to view full memory
            </div>

            {/* Tooltip arrow */}
            <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-b-overlay" />
          </div>
        )}
      </div>
    );
  };

  const renderExpandedMemory = () => {
    if (!expandedTraceId) return null;

    const expandedTrace = sortedTraces.find(trace => trace.id === expandedTraceId);
    if (!expandedTrace) return null;

    const percentage = getRelevancePercentage(expandedTrace.relevance);

    return (
      <div className={cls(
        'transition-all duration-300 ease-in-out overflow-hidden',
        'bg-accent-subtle rounded-lg p-4 mt-3',
        'border border-border',
        'max-h-96 opacity-100'
      )}>
        <div className={cls(
          'flex',
          'items-center',
          'justify-between',
          'mb-2'
        )}>
          <div className={cls('flex', 'items-center', 'gap-2')}>
            <span className="text-lg">🧠</span>
            <span className="text-sm font-semibold text-accent">
              Memory Trace
            </span>
          </div>
          <div className={cls(
            'text-xs px-2 py-1 rounded-full font-semibold',
            getRelevanceBgColor(expandedTrace.relevance),
            getRelevanceColor(expandedTrace.relevance)
          )}>
            {percentage}% relevant
          </div>
        </div>

        <div className={cls(
          'text-sm text-default leading-relaxed',
          'max-h-64 overflow-y-auto',
          'pr-2 custom-scrollbar'
        )}>
          {expandedTrace.content}
        </div>

        {/* Footer with feedback controls and metadata */}
        <div className={cls(
          'flex',
          'items-center',
          'justify-between',
          'mt-3 pt-3 border-t border-border-muted'
        )}>
          <div className="text-[11px] text-muted">
            Memory ID: {expandedTrace.id}
          </div>
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

  if (sortedTraces.length === 0) {
    return null;
  }

  return (
    <div className={cls('w-full', className)}>
      {/* Memory badges row */}
      <div className={cls('flex', 'items-center', 'gap-2', 'flex-wrap')}>
        {sortedTraces.map(renderMemoryBadge)}
      </div>

      {/* Expanded memory details */}
      {renderExpandedMemory()}
    </div>
  );
};

export default MemoryTraceAddon;
