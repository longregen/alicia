import React, { useState, useEffect, useCallback } from 'react';
import { cls } from '../../../utils/cls';
import type { Memory, MemoryCategory } from '../../../stores/memoryStore';

export interface MemoryEditorProps {
  /** Memory being edited (null for create mode) */
  memory: Memory | null;
  /** Whether the modal is open */
  isOpen: boolean;
  /** Callback when save is clicked */
  onSave: (content: string, category: MemoryCategory) => void;
  /** Callback when cancel is clicked */
  onCancel: () => void;
  /** Whether save operation is in progress */
  isLoading?: boolean;
  className?: string;
}

const categories: Array<{ value: MemoryCategory; label: string; description: string }> = [
  { value: 'preference', label: 'Preference', description: 'User preferences and settings' },
  { value: 'fact', label: 'Fact', description: 'Factual information about the user' },
  { value: 'context', label: 'Context', description: 'Contextual information for conversations' },
  { value: 'instruction', label: 'Instruction', description: 'Instructions for the assistant' },
  { value: 'history', label: 'History', description: 'Historical events or past interactions' },
];

/**
 * MemoryEditor component for creating/editing memories.
 *
 * Features:
 * - Modal dialog for focused editing
 * - Content textarea
 * - Category selection with descriptions
 * - Save/Cancel actions
 * - Validation
 */
export const MemoryEditor: React.FC<MemoryEditorProps> = ({
  memory,
  isOpen,
  onSave,
  onCancel,
  isLoading = false,
  className = '',
}) => {
  const [content, setContent] = useState('');
  const [category, setCategory] = useState<MemoryCategory>('fact');

  // Initialize form when memory changes
  useEffect(() => {
    if (memory) {
      setContent(memory.content);
      setCategory(memory.category);
    } else {
      setContent('');
      setCategory('fact');
    }
  }, [memory]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (content.trim()) {
      onSave(content.trim(), category);
    }
  };

  const handleCancel = useCallback(() => {
    setContent('');
    setCategory('fact');
    onCancel();
  }, [onCancel]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        handleCancel();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, handleCancel]);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-overlay z-40"
        onClick={handleCancel}
      />

      {/* Modal */}
      <div
        className={cls(
          'fixed inset-0 z-50 flex items-center justify-center p-4',
          className
        )}
        onClick={(e) => e.target === e.currentTarget && handleCancel()}
      >
        <div
          className="bg-surface shadow-2xl rounded-lg w-full max-w-2xl p-6 layout-stack-gap-4"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="layout-between">
            <h2 className="text-lg font-semibold text-default">
              {memory ? 'Edit Memory' : 'Create Memory'}
            </h2>
            <button
              onClick={handleCancel}
              className="text-muted hover:text-default transition-colors p-2 rounded hover:bg-sunken"
              aria-label="Close"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="layout-stack-gap-4">
            {/* Content */}
            <div className="layout-stack-gap">
              <label htmlFor="memory-content" className="text-sm font-medium text-default">
                Content
              </label>
              <textarea
                id="memory-content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Enter memory content..."
                rows={6}
                className={cls(
                  'w-full px-3 py-2 rounded border bg-surface text-default text-sm',
                  'focus:outline-none focus:border-accent transition-colors resize-none'
                )}
                autoFocus
                required
              />
            </div>

            {/* Category */}
            <div className="layout-stack-gap">
              <label htmlFor="memory-category" className="text-sm font-medium text-default">
                Category
              </label>
              <div className="grid grid-cols-2 gap-3">
                {categories.map((cat) => (
                  <button
                    key={cat.value}
                    type="button"
                    onClick={() => setCategory(cat.value)}
                    className={cls(
                      'p-3 rounded border text-left transition-colors',
                      'focus:outline-none focus:ring-2 focus:ring-accent',
                      category === cat.value
                        ? 'border-accent bg-accent-subtle'
                        : 'border-muted hover:bg-sunken'
                    )}
                  >
                    <div className="font-medium text-sm text-default">
                      {cat.label}
                    </div>
                    <div className="text-xs text-muted mt-1">
                      {cat.description}
                    </div>
                  </button>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-between gap-3 mt-2">
              <button
                type="button"
                onClick={handleCancel}
                disabled={isLoading}
                className={cls(
                  'px-4 py-2 rounded border text-default text-sm font-medium',
                  'hover:bg-sunken transition-colors',
                  'focus:outline-none focus:ring-2 focus:ring-muted',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!content.trim() || isLoading}
                className={cls(
                  'px-4 py-2 rounded bg-accent text-on-emphasis text-sm font-medium',
                  'hover:bg-accent-hover transition-colors',
                  'focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2',
                  'disabled:opacity-50 disabled:cursor-not-allowed',
                  'layout-center-gap'
                )}
              >
                {isLoading && (
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                )}
                {memory ? 'Update' : 'Create'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
};
