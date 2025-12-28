import React, { useState } from 'react';
import { cls } from '../../../utils/cls';
import { CSS } from '../../../utils/constants';
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
  /** Callback to create new memory */
  onCreateNew: () => void;
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
 * MemorySearch component for filtering and creating memories.
 *
 * Features:
 * - Search input for filtering by content
 * - Category dropdown for filtering by type
 * - Create new memory button
 */
export const MemorySearch: React.FC<MemorySearchProps> = ({
  searchQuery,
  selectedCategory,
  onSearchChange,
  onCategoryChange,
  onCreateNew,
  className = '',
}) => {
  const [showCategories, setShowCategories] = useState(false);

  return (
    <div className={cls(CSS.flex, CSS.flexCol, CSS.gap3, className)}>
      {/* Search bar and create button */}
      <div className={cls(CSS.flex, CSS.gap2)}>
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search memories..."
          className={cls(
            CSS.flex,
            'flex-1',
            CSS.px3,
            CSS.py2,
            CSS.rounded,
            CSS.border,
            CSS.borderSurface300,
            CSS.bgSurfaceBg,
            CSS.textPrimary,
            CSS.textSm,
            CSS.focusOutlineNone,
            'focus:border-primary-blue',
            CSS.transitionColors
          )}
        />
        <button
          onClick={onCreateNew}
          className={cls(
            CSS.px4,
            CSS.py2,
            CSS.rounded,
            CSS.bgPrimaryBlue,
            CSS.textWhite,
            CSS.textSm,
            CSS.fontMedium,
            CSS.hoverBgPrimaryBlue,
            CSS.transitionColors,
            CSS.focusOutlineNone,
            'focus:ring-2 focus:ring-primary-blue focus:ring-offset-2'
          )}
        >
          Create Memory
        </button>
      </div>

      {/* Category filter */}
      <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap2)}>
        <span className={cls(CSS.textSm, CSS.textMuted)}>Filter:</span>
        <div className="relative">
          <button
            onClick={() => setShowCategories(!showCategories)}
            className={cls(
              CSS.px3,
              CSS.py1,
              CSS.rounded,
              CSS.border,
              CSS.borderSurface300,
              CSS.bgSurfaceBg,
              CSS.textPrimary,
              CSS.textSm,
              CSS.hoverBgSurface100,
              CSS.transitionColors,
              CSS.focusOutlineNone,
              CSS.flex,
              CSS.itemsCenter,
              CSS.gap2,
              'min-w-[140px]'
            )}
          >
            <span>
              {categories.find((c) => c.value === selectedCategory)?.label || 'All'}
            </span>
            <svg
              className={cls('w-4 h-4', CSS.textMuted, CSS.transitionTransform, showCategories ? 'rotate-180' : '')}
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
              <div
                className={cls(
                  'absolute z-20 mt-1 w-full',
                  CSS.rounded,
                  CSS.border,
                  CSS.borderSurface300,
                  CSS.bgSurfaceBg,
                  'shadow-lg'
                )}
              >
                {categories.map((category) => (
                  <button
                    key={category.value}
                    onClick={() => {
                      onCategoryChange(category.value);
                      setShowCategories(false);
                    }}
                    className={cls(
                      CSS.wFull,
                      CSS.px3,
                      CSS.py2,
                      CSS.textSm,
                      CSS.textPrimary,
                      'text-left',
                      CSS.hoverBgSurface100,
                      CSS.transitionColors,
                      selectedCategory === category.value ? CSS.bgPrimaryBlue : '',
                      selectedCategory === category.value ? CSS.textWhite : '',
                      'first:rounded-t',
                      'last:rounded-b'
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
            className={cls(
              CSS.textSm,
              CSS.textPrimaryBlue,
              'hover:underline',
              CSS.transitionColors
            )}
          >
            Clear filters
          </button>
        )}
      </div>
    </div>
  );
};
