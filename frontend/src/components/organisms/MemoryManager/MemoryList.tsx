import React from 'react';
import { cls } from '../../../utils/cls';
import { CSS } from '../../../utils/constants';
import type { Memory, MemoryCategory } from '../../../stores/memoryStore';

export interface MemoryListProps {
  /** List of memories to display */
  memories: Memory[];
  /** Callback when edit is clicked */
  onEdit: (memory: Memory) => void;
  /** Callback when delete is clicked */
  onDelete: (memory: Memory) => void;
  /** Callback when pin is toggled */
  onPin: (memory: Memory) => void;
  /** Callback when archive is clicked */
  onArchive: (memory: Memory) => void;
  /** Whether any operation is in progress */
  isLoading?: boolean;
  className?: string;
}

const categoryColors: Record<MemoryCategory, { bg: string; text: string; border: string }> = {
  preference: {
    bg: 'bg-purple-50 dark:bg-purple-900/20',
    text: 'text-purple-700 dark:text-purple-300',
    border: 'border-purple-200 dark:border-purple-700',
  },
  fact: {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    text: 'text-blue-700 dark:text-blue-300',
    border: 'border-blue-200 dark:border-blue-700',
  },
  context: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    text: 'text-green-700 dark:text-green-300',
    border: 'border-green-200 dark:border-green-700',
  },
  instruction: {
    bg: 'bg-orange-50 dark:bg-orange-900/20',
    text: 'text-orange-700 dark:text-orange-300',
    border: 'border-orange-200 dark:border-orange-700',
  },
};

const formatDate = (timestamp: number): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  // Less than 1 minute
  if (diff < 60000) {
    return 'just now';
  }

  // Less than 1 hour
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000);
    return `${minutes}m ago`;
  }

  // Less than 1 day
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours}h ago`;
  }

  // Less than 1 week
  if (diff < 604800000) {
    const days = Math.floor(diff / 86400000);
    return `${days}d ago`;
  }

  // Otherwise show date
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
};

/**
 * MemoryList component for displaying a list of memories.
 *
 * Features:
 * - Card-based layout
 * - Shows content, category, pinned status, creation date
 * - Action buttons: edit, pin, archive, delete
 * - Empty state
 * - Responsive design
 */
export const MemoryList: React.FC<MemoryListProps> = ({
  memories,
  onEdit,
  onDelete,
  onPin,
  onArchive,
  isLoading = false,
  className = '',
}) => {
  if (memories.length === 0) {
    return (
      <div
        className={cls(
          CSS.flex,
          CSS.flexCol,
          CSS.itemsCenter,
          CSS.justifyCenter,
          CSS.p6,
          CSS.textCenter,
          className
        )}
      >
        <svg
          className={cls('w-16 h-16', CSS.textMuted, CSS.mb2)}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        </svg>
        <p className={cls(CSS.textMuted, CSS.textSm)}>No memories found</p>
        <p className={cls(CSS.textMuted, CSS.textXs, CSS.mt1)}>
          Create your first memory to get started
        </p>
      </div>
    );
  }

  return (
    <div className={cls(CSS.flexCol, CSS.gap3, className)}>
      {memories.map((memory) => {
        const categoryStyle = categoryColors[memory.category];

        return (
          <div
            key={memory.id}
            className={cls(
              CSS.p4,
              CSS.rounded,
              CSS.border,
              CSS.borderSurface300,
              CSS.bgSurfaceBg,
              CSS.flexCol,
              CSS.gap3,
              CSS.transitionAll,
              'hover:shadow-md',
              memory.pinned ? 'ring-2 ring-primary-blue ring-opacity-50' : ''
            )}
          >
            {/* Header */}
            <div className={cls(CSS.flex, CSS.itemsStart, CSS.justifyBetween, CSS.gap3)}>
              {/* Category and pinned badge */}
              <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2)}>
                <span
                  className={cls(
                    CSS.px2,
                    CSS.py1,
                    CSS.rounded,
                    CSS.textXs,
                    CSS.fontMedium,
                    CSS.border,
                    categoryStyle.bg,
                    categoryStyle.text,
                    categoryStyle.border
                  )}
                >
                  {memory.category}
                </span>
                {memory.pinned && (
                  <svg
                    className={cls('w-4 h-4', CSS.textPrimaryBlue)}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6zM15.657 5.404a.75.75 0 10-1.06-1.06l-1.061 1.06a.75.75 0 001.06 1.06l1.06-1.06zM6.464 14.596a.75.75 0 10-1.06-1.06l-1.06 1.06a.75.75 0 001.06 1.06l1.06-1.06zM18 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 0118 10zM5 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 015 10zM14.596 15.657a.75.75 0 001.06-1.06l-1.06-1.061a.75.75 0 10-1.06 1.06l1.06 1.06zM5.404 6.464a.75.75 0 001.06-1.06l-1.06-1.06a.75.75 0 10-1.061 1.06l1.06 1.06z" />
                  </svg>
                )}
              </div>

              {/* Actions */}
              <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap1)}>
                <button
                  onClick={() => onPin(memory)}
                  disabled={isLoading}
                  className={cls(
                    CSS.p2,
                    CSS.rounded,
                    CSS.textMuted,
                    'hover:text-primary-blue',
                    CSS.hoverBgSurface100,
                    CSS.transitionColors,
                    CSS.disabledOpacity50
                  )}
                  title={memory.pinned ? 'Unpin' : 'Pin'}
                  aria-label={memory.pinned ? 'Unpin memory' : 'Pin memory'}
                >
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6zM15.657 5.404a.75.75 0 10-1.06-1.06l-1.061 1.06a.75.75 0 001.06 1.06l1.06-1.06zM6.464 14.596a.75.75 0 10-1.06-1.06l-1.06 1.06a.75.75 0 001.06 1.06l1.06-1.06zM18 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 0118 10zM5 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 015 10zM14.596 15.657a.75.75 0 001.06-1.06l-1.06-1.061a.75.75 0 10-1.06 1.06l1.06 1.06zM5.404 6.464a.75.75 0 001.06-1.06l-1.06-1.06a.75.75 0 10-1.061 1.06l1.06 1.06z" />
                  </svg>
                </button>

                <button
                  onClick={() => onEdit(memory)}
                  disabled={isLoading}
                  className={cls(
                    CSS.p2,
                    CSS.rounded,
                    CSS.textMuted,
                    'hover:text-primary-text',
                    CSS.hoverBgSurface100,
                    CSS.transitionColors,
                    CSS.disabledOpacity50
                  )}
                  title="Edit"
                  aria-label="Edit memory"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>

                <button
                  onClick={() => onArchive(memory)}
                  disabled={isLoading}
                  className={cls(
                    CSS.p2,
                    CSS.rounded,
                    CSS.textMuted,
                    'hover:text-orange-600',
                    CSS.hoverBgSurface100,
                    CSS.transitionColors,
                    CSS.disabledOpacity50
                  )}
                  title="Archive"
                  aria-label="Archive memory"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
                    />
                  </svg>
                </button>

                <button
                  onClick={() => onDelete(memory)}
                  disabled={isLoading}
                  className={cls(
                    CSS.p2,
                    CSS.rounded,
                    CSS.textMuted,
                    'hover:text-red-600',
                    CSS.hoverBgSurface100,
                    CSS.transitionColors,
                    CSS.disabledOpacity50
                  )}
                  title="Delete"
                  aria-label="Delete memory"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </div>

            {/* Content */}
            <p className={cls(CSS.textSm, CSS.textPrimary, 'line-clamp-3')}>
              {memory.content}
            </p>

            {/* Footer */}
            <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyBetween)}>
              <span className={cls(CSS.textXs, CSS.textMuted)}>
                Created {formatDate(memory.createdAt)}
              </span>
              {memory.updatedAt > memory.createdAt && (
                <span className={cls(CSS.textXs, CSS.textMuted)}>
                  Updated {formatDate(memory.updatedAt)}
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
};
