import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { cls } from '../../utils/cls';
import type { MessageId } from '../../types/streaming';

export interface BranchNavigatorProps {
  messageId?: MessageId;
  currentIndex: number;
  totalBranches: number;
  onNavigate: (direction: 'prev' | 'next') => void;
  className?: string;
}

const BranchNavigator: React.FC<BranchNavigatorProps> = ({
  currentIndex,
  totalBranches,
  onNavigate,
  className = '',
  // messageId is optional and unused - kept for API compatibility
}) => {
  // Don't render if there's only one branch or none
  if (totalBranches <= 1) return null;

  const displayIndex = currentIndex + 1; // Convert from 0-based to 1-based

  return (
    <div className={cls('flex items-center gap-1 text-xs text-muted', className)}>
      <button
        onClick={() => onNavigate('prev')}
        disabled={currentIndex === 0}
        className={cls(
          'p-1 rounded transition-colors',
          currentIndex === 0
            ? 'opacity-30 cursor-not-allowed'
            : 'hover:bg-surface-hover hover:text-default cursor-pointer'
        )}
        aria-label="Previous branch"
      >
        <ChevronLeft className="w-3 h-3" />
      </button>

      <span className="min-w-[3rem] text-center font-mono">
        {displayIndex}/{totalBranches}
      </span>

      <button
        onClick={() => onNavigate('next')}
        disabled={currentIndex === totalBranches - 1}
        className={cls(
          'p-1 rounded transition-colors',
          currentIndex === totalBranches - 1
            ? 'opacity-30 cursor-not-allowed'
            : 'hover:bg-surface-hover hover:text-default cursor-pointer'
        )}
        aria-label="Next branch"
      >
        <ChevronRight className="w-3 h-3" />
      </button>
    </div>
  );
};

export default BranchNavigator;
