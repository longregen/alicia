import React, { useState } from 'react';
import { cls } from '../../../utils/cls';
import type { Memory, MemoryCategory } from '../../../stores/memoryStore';
import StarRating, { importanceToStar } from '../../atoms/StarRating';
import { Popover, PopoverTrigger, PopoverContent } from '../../atoms/Popover';
import type { MemoryDeletionReason } from '../../../hooks/useMemories';

type SortField = 'content' | 'category' | 'importance' | 'usageCount' | 'createdAt';
type SortDirection = 'asc' | 'desc';

export interface MemoryListProps {
  /** List of memories to display */
  memories: Memory[];
  /** Callback when edit is clicked */
  onEdit: (memory: Memory) => void;
  /** Callback when delete is clicked with optional reason */
  onDelete: (memory: Memory, reason?: MemoryDeletionReason) => void;
  /** Callback when rating changes */
  onRatingChange?: (memory: Memory, stars: number) => void;
  /** Callback when category changes */
  onCategoryChange?: (memory: Memory, category: MemoryCategory) => void;
  /** Callback when a memory row is clicked */
  onSelect?: (memory: Memory) => void;
  /** Whether any operation is in progress */
  isLoading?: boolean;
  /** Show restore/permanent delete for deleted view */
  showDeletedActions?: boolean;
  /** Callback when restore is clicked (for deleted view) */
  onRestore?: (memory: Memory) => void;
  /** Callback when permanent delete is clicked (for deleted view) */
  onPermanentDelete?: (memory: Memory) => void;
  className?: string;
}

const deletionReasons: { value: MemoryDeletionReason; label: string }[] = [
  { value: 'wrong', label: 'Wrong' },
  { value: 'useless', label: 'Useless' },
  { value: 'old', label: 'Outdated' },
  { value: 'duplicate', label: 'Duplicate' },
  { value: 'other', label: 'Other' },
];

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
  history: {
    bg: 'bg-muted/10',
    text: 'text-muted',
    border: 'border-muted',
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
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
};

/**
 * MemoryList component for displaying a list of memories in a table.
 *
 * Features:
 * - Table-based layout
 * - Shows content, category, importance, usage, creation date
 * - Action buttons: pin, edit, archive, delete
 * - Empty state
 * - Responsive design
 */
const categories: MemoryCategory[] = ['preference', 'fact', 'context', 'instruction', 'history'];

export const MemoryList: React.FC<MemoryListProps> = ({
  memories,
  onEdit,
  onDelete,
  onRatingChange,
  onCategoryChange,
  onSelect,
  isLoading = false,
  showDeletedActions = false,
  onRestore,
  onPermanentDelete,
  className = '',
}) => {
  const [deletePopoverOpen, setDeletePopoverOpen] = useState<string | null>(null);
  const [categoryPopoverOpen, setCategoryPopoverOpen] = useState<string | null>(null);
  const [sortField, setSortField] = useState<SortField>('importance');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const sortedMemories = [...memories].sort((a, b) => {
    let comparison = 0;
    switch (sortField) {
      case 'content':
        comparison = a.content.localeCompare(b.content);
        break;
      case 'category':
        comparison = a.category.localeCompare(b.category);
        break;
      case 'importance':
        comparison = a.importance - b.importance;
        break;
      case 'usageCount':
        comparison = a.usageCount - b.usageCount;
        break;
      case 'createdAt':
        comparison = a.createdAt - b.createdAt;
        break;
    }
    return sortDirection === 'asc' ? comparison : -comparison;
  });

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) {
      return (
        <svg className="w-3 h-3 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
        </svg>
      );
    }
    return sortDirection === 'asc' ? (
      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
      </svg>
    ) : (
      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
      </svg>
    );
  };

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
    <div className={cls('overflow-x-auto', className)}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left">
            <th className="pb-3 pr-4 font-medium text-muted">
              <button
                onClick={() => handleSort('content')}
                className="flex items-center gap-1 hover:text-default transition-colors"
              >
                Content <SortIcon field="content" />
              </button>
            </th>
            <th className="pb-3 pr-4 font-medium text-muted w-28">
              <button
                onClick={() => handleSort('category')}
                className="flex items-center gap-1 hover:text-default transition-colors"
              >
                Category <SortIcon field="category" />
              </button>
            </th>
            <th className="pb-3 pr-4 font-medium text-muted w-20 text-center">
              <button
                onClick={() => handleSort('importance')}
                className="flex items-center gap-1 hover:text-default transition-colors mx-auto"
              >
                Importance <SortIcon field="importance" />
              </button>
            </th>
            <th className="pb-3 pr-4 font-medium text-muted w-16 text-center">
              <button
                onClick={() => handleSort('usageCount')}
                className="flex items-center gap-1 hover:text-default transition-colors mx-auto"
              >
                Used <SortIcon field="usageCount" />
              </button>
            </th>
            <th className="pb-3 pr-4 font-medium text-muted w-24">
              <button
                onClick={() => handleSort('createdAt')}
                className="flex items-center gap-1 hover:text-default transition-colors"
              >
                Created <SortIcon field="createdAt" />
              </button>
            </th>
            <th className="pb-3 font-medium text-muted w-36 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {sortedMemories.map((memory) => {
            const categoryStyle = categoryColors[memory.category];

            return (
              <tr
                key={memory.id}
                className={cls(
                  'border-b border-border/50 hover:bg-surface-hover transition-colors',
                  onSelect ? 'cursor-pointer' : ''
                )}
                onClick={() => onSelect?.(memory)}
              >
                {/* Content */}
                <td className="py-3 pr-4">
                  <div className="max-w-md">
                    <p className="text-default truncate" title={memory.content}>
                      {memory.content}
                    </p>
                    {memory.tags && memory.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {memory.tags.slice(0, 3).map((tag, index) => (
                          <span
                            key={index}
                            className="px-1.5 py-0.5 rounded text-[10px] bg-sunken text-muted"
                          >
                            {tag}
                          </span>
                        ))}
                        {memory.tags.length > 3 && (
                          <span className="text-[10px] text-muted">+{memory.tags.length - 3}</span>
                        )}
                      </div>
                    )}
                  </div>
                </td>

                {/* Category - with dropdown to change */}
                <td className="py-3 pr-4" onClick={(e) => e.stopPropagation()}>
                  {onCategoryChange ? (
                    <Popover
                      open={categoryPopoverOpen === memory.id}
                      onOpenChange={(open) => setCategoryPopoverOpen(open ? memory.id : null)}
                    >
                      <PopoverTrigger asChild>
                        <button
                          className={cls(
                            'px-2 py-1 rounded text-xs font-medium border cursor-pointer',
                            'flex items-center gap-1 transition-all',
                            'hover:brightness-95 hover:shadow-sm',
                            categoryPopoverOpen === memory.id ? 'ring-2 ring-accent/30' : '',
                            categoryStyle.bg,
                            categoryStyle.text,
                            categoryStyle.border
                          )}
                          disabled={isLoading}
                        >
                          {memory.category}
                          <svg className={cls('w-3 h-3 transition-transform', categoryPopoverOpen === memory.id ? 'rotate-180' : '')} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                          </svg>
                        </button>
                      </PopoverTrigger>
                      <PopoverContent
                        align="start"
                        side="bottom"
                        className="w-44 p-1.5 shadow-lg"
                      >
                        <div className="flex flex-col gap-0.5">
                          {categories.map((cat) => {
                            const style = categoryColors[cat];
                            const isSelected = memory.category === cat;
                            return (
                              <button
                                key={cat}
                                onClick={() => {
                                  onCategoryChange(memory, cat);
                                  setCategoryPopoverOpen(null);
                                }}
                                className={cls(
                                  'px-3 py-2.5 rounded text-xs font-medium text-left flex items-center gap-2',
                                  'transition-all cursor-pointer border',
                                  isSelected
                                    ? cls(style.bg, style.text, style.border)
                                    : 'border-transparent hover:bg-accent-subtle hover:border-accent/30 text-default'
                                )}
                              >
                                <span className={cls('w-2.5 h-2.5 rounded-full', style.bg, 'border', style.border)} />
                                <span className="flex-1">{cat}</span>
                                {isSelected && (
                                  <svg className={cls('w-4 h-4', style.text)} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                  </svg>
                                )}
                              </button>
                            );
                          })}
                        </div>
                      </PopoverContent>
                    </Popover>
                  ) : (
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
                  )}
                </td>

                {/* Importance - Star Rating */}
                <td className="py-3 pr-4">
                  <div
                    onClick={(e) => e.stopPropagation()}
                    className="flex justify-center"
                  >
                    <StarRating
                      rating={importanceToStar(memory.importance)}
                      onRate={(stars) => onRatingChange?.(memory, stars)}
                      isLoading={isLoading}
                      compact
                      readOnly={!onRatingChange}
                    />
                  </div>
                </td>

                {/* Usage count */}
                <td className="py-3 pr-4 text-center text-muted">
                  {memory.usageCount}
                </td>

                {/* Created date */}
                <td className="py-3 pr-4 text-muted">
                  {formatDate(memory.createdAt)}
                </td>

                {/* Actions */}
                <td className="py-3">
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onEdit(memory);
                      }}
                      disabled={isLoading}
                      className={cls(
                        'p-1.5 rounded text-muted hover:text-default hover:bg-sunken',
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

                    {/* Delete with inline popover */}
                    {showDeletedActions ? (
                      // For deleted view: show restore and permanent delete
                      <>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onRestore?.(memory);
                          }}
                          disabled={isLoading}
                          className={cls(
                            'p-1.5 rounded text-muted hover:text-success hover:bg-sunken',
                            'transition-colors disabled:opacity-50'
                          )}
                          title="Restore"
                          aria-label="Restore memory"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                          </svg>
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onPermanentDelete?.(memory);
                          }}
                          disabled={isLoading}
                          className={cls(
                            'p-1.5 rounded text-muted hover:text-error hover:bg-sunken',
                            'transition-colors disabled:opacity-50'
                          )}
                          title="Delete permanently"
                          aria-label="Delete memory permanently"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </>
                    ) : (
                      // For active view: inline delete popover
                      <Popover
                        open={deletePopoverOpen === memory.id}
                        onOpenChange={(open) => setDeletePopoverOpen(open ? memory.id : null)}
                      >
                        <PopoverTrigger asChild>
                          <button
                            disabled={isLoading}
                            className={cls(
                              'p-1.5 rounded text-muted hover:text-error hover:bg-error-subtle',
                              'transition-all disabled:opacity-50',
                              deletePopoverOpen === memory.id ? 'text-error bg-error-subtle ring-2 ring-error/30' : ''
                            )}
                            title="Delete"
                            aria-label="Delete memory"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
                        </PopoverTrigger>
                        <PopoverContent
                          align="end"
                          side="bottom"
                          className="w-52 p-2 shadow-lg"
                        >
                          <p className="text-xs font-medium text-default mb-2 px-1">Why delete?</p>
                          <div className="flex flex-col gap-1 mb-3">
                            {deletionReasons.map((reason) => (
                              <button
                                key={reason.value}
                                onClick={() => {
                                  onDelete(memory, reason.value);
                                  setDeletePopoverOpen(null);
                                }}
                                className={cls(
                                  'px-3 py-2.5 text-xs rounded text-left cursor-pointer',
                                  'bg-surface border border-border',
                                  'hover:border-error hover:bg-error-subtle hover:text-error hover:shadow-sm',
                                  'transition-all active:scale-[0.98]'
                                )}
                              >
                                {reason.label}
                              </button>
                            ))}
                          </div>
                          <div className="border-t border-border pt-2">
                            <button
                              onClick={() => {
                                onDelete(memory);
                                setDeletePopoverOpen(null);
                              }}
                              className={cls(
                                'w-full px-3 py-2.5 text-xs rounded cursor-pointer',
                                'bg-error text-white font-medium',
                                'hover:bg-error/80 hover:shadow-md transition-all active:scale-[0.98]'
                              )}
                            >
                              Delete without reason
                            </button>
                          </div>
                        </PopoverContent>
                      </Popover>
                    )}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};
