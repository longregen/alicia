import React from 'react';
import { cls } from '../../../utils/cls';
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
    bg: 'bg-accent-subtle',
    text: 'text-accent',
    border: 'border-accent',
  },
  fact: {
    bg: 'bg-success-subtle',
    text: 'text-success',
    border: 'border-success',
  },
  context: {
    bg: 'bg-warning-subtle',
    text: 'text-warning',
    border: 'border-warning',
  },
  instruction: {
    bg: 'bg-error-subtle',
    text: 'text-error',
    border: 'border-error',
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
      <div className={cls('flex flex-col items-center justify-center p-6 md:p-8 lg:p-12 text-center', className)}>
        <svg
          className="w-16 h-16 text-muted mb-2"
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
        <p className="text-muted text-sm">No memories found</p>
        <p className="text-muted text-xs mt-1">
          Create your first memory to get started
        </p>
      </div>
    );
  }

  return (
    <div className={cls('grid grid-cols-1 md:grid-cols-2 gap-3', className)}>
      {memories.map((memory) => {
        const categoryStyle = categoryColors[memory.category];

        return (
          <div
            key={memory.id}
            className={cls(
              'p-4 rounded border bg-surface flex flex-col gap-3 transition-all',
              'hover:shadow-md',
              memory.pinned ? 'ring-2 ring-accent ring-opacity-50' : ''
            )}
          >
            {/* Header */}
            <div className="flex items-start justify-between gap-3">
              {/* Category and pinned badge */}
              <div className="flex items-center gap-2">
                <span
                  className={cls(
                    'px-2 py-1 rounded text-xs font-medium border',
                    categoryStyle.bg,
                    categoryStyle.text,
                    categoryStyle.border
                  )}
                >
                  {memory.category}
                </span>
                {memory.pinned && (
                  <svg
                    className="w-4 h-4 text-accent"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6zM15.657 5.404a.75.75 0 10-1.06-1.06l-1.061 1.06a.75.75 0 001.06 1.06l1.06-1.06zM6.464 14.596a.75.75 0 10-1.06-1.06l-1.06 1.06a.75.75 0 001.06 1.06l1.06-1.06zM18 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 0118 10zM5 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 015 10zM14.596 15.657a.75.75 0 001.06-1.06l-1.06-1.061a.75.75 0 10-1.06 1.06l1.06 1.06zM5.404 6.464a.75.75 0 001.06-1.06l-1.06-1.06a.75.75 0 10-1.061 1.06l1.06 1.06z" />
                  </svg>
                )}
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1">
                <button
                  onClick={() => onPin(memory)}
                  disabled={isLoading}
                  className={cls(
                    'p-2 rounded text-muted hover:text-accent hover:bg-sunken',
                    'transition-colors disabled:opacity-50'
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
                    'p-2 rounded text-muted hover:text-default hover:bg-sunken',
                    'transition-colors disabled:opacity-50'
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
                    'p-2 rounded text-muted hover:text-warning hover:bg-sunken',
                    'transition-colors disabled:opacity-50'
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
                    'p-2 rounded text-muted hover:text-error hover:bg-sunken',
                    'transition-colors disabled:opacity-50'
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
            <p className="text-sm text-default line-clamp-3">
              {memory.content}
            </p>

            {/* Tags */}
            {memory.tags && memory.tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {memory.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="px-2 py-0.5 rounded text-xs bg-sunken text-muted border border-muted"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}

            {/* Metadata: Importance and Usage Count */}
            <div className="flex items-center gap-4 text-xs text-muted">
              <span className="flex items-center gap-1">
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                </svg>
                {Math.round(memory.importance * 100)}%
              </span>
              <span className="flex items-center gap-1">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                </svg>
                Used {memory.usageCount} {memory.usageCount === 1 ? 'time' : 'times'}
              </span>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted">
                Created {formatDate(memory.createdAt)}
              </span>
              {memory.updatedAt > memory.createdAt && (
                <span className="text-xs text-muted">
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
