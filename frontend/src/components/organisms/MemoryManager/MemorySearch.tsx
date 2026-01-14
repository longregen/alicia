import React, { useState } from 'react';
import { cls } from '../../../utils/cls';
import type { MemoryCategory } from '../../../stores/memoryStore';

export interface MemorySearchProps {
  /** Current search query */
  searchQuery: string;
  /** Selected category filter */
  selectedCategory: MemoryCategory | 'all';
  /** Callback when search query changes */
  onSearchChange: (query: string) => void;
  /** Callback when category filter changes */
  onCategoryChange: (category: MemoryCategory | 'all') => void;
  className?: string;
}

const categories: Array<{ value: MemoryCategory | 'all'; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'preference', label: 'Preferences' },
  { value: 'fact', label: 'Facts' },
  { value: 'context', label: 'Context' },
  { value: 'instruction', label: 'Instructions' },
];

/**
 * MemorySearch component for filtering memories.
 *
 * Features:
 * - Search input for filtering by content
 * - Category dropdown for filtering by type
 */
export const MemorySearch: React.FC<MemorySearchProps> = ({
  searchQuery,
  selectedCategory,
  onSearchChange,
  onCategoryChange,
  className = '',
}) => {
  const [showCategories, setShowCategories] = useState(false);

  return (
    <div className={cls('flex flex-col gap-3', className)}>
      {/* Search bar */}
      <input
        type="text"
        value={searchQuery}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder="Search memories..."
        className={cls(
          'flex px-3 py-2 rounded border bg-surface text-default text-sm',
          'focus:outline-none focus:border-accent transition-colors'
        )}
      />

      {/* Category filter */}
      <div className="layout-center-gap">
        <span className="text-sm text-muted">Filter:</span>
        <div className="relative">
          <button
            onClick={() => setShowCategories(!showCategories)}
            className={cls(
              'px-3 py-1 rounded border bg-surface text-default text-sm',
              'hover:bg-sunken transition-colors focus:outline-none',
              'layout-center-gap min-w-[140px]'
            )}
          >
            <span>
              {categories.find((c) => c.value === selectedCategory)?.label || 'All'}
            </span>
            <svg
              className={cls('w-4 h-4 text-muted transition-transform', showCategories ? 'rotate-180' : '')}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {showCategories && (
            <>
              {/* Backdrop */}
              <div
                className="fixed inset-0 z-10"
                onClick={() => setShowCategories(false)}
              />

              {/* Dropdown */}
              <div className="absolute z-20 mt-1 w-full rounded border bg-surface shadow-lg">
                {categories.map((category) => (
                  <button
                    key={category.value}
                    onClick={() => {
                      onCategoryChange(category.value);
                      setShowCategories(false);
                    }}
                    className={cls(
                      'w-full px-3 py-2 text-sm text-default text-left',
                      'hover:bg-sunken transition-colors',
                      selectedCategory === category.value ? 'bg-accent-subtle text-accent' : '',
                      'first:rounded-t last:rounded-b'
                    )}
                  >
                    {category.label}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Clear filters */}
        {(searchQuery || selectedCategory !== 'all') && (
          <button
            onClick={() => {
              onSearchChange('');
              onCategoryChange('all');
            }}
            className="text-sm text-accent hover:underline transition-colors"
          >
            Clear filters
          </button>
        )}
      </div>
    </div>
  );
};
