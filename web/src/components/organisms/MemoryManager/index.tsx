import React, { useState, useMemo } from 'react';
import { cls } from '../../../utils/cls';
import type { Memory, MemoryCategory } from '../../../stores/memoryStore';
import { useMemories, type MemoryDeletionReason } from '../../../hooks/useMemories';
import { MemorySearch } from './MemorySearch';
import { MemoryList } from './MemoryList';
import { MemoryEditor } from './MemoryEditor';

export interface MemoryManagerProps {
  className?: string;
}

type ViewTab = 'active' | 'deleted';

/**
 * MemoryManager organism component for managing global memories.
 *
 * Features:
 * - Display all memories with category, importance, creation date
 * - Search/filter by content or category
 * - Create new memories (content + category)
 * - Edit, delete memories with inline actions
 * - Deleted tab for restore/permanent delete
 * - Uses memoryStore with API integration via useMemories hook
 */
export const MemoryManager: React.FC<MemoryManagerProps> = ({ className = '' }) => {
  const {
    memories,
    deletedMemories,
    isLoading,
    isFetching,
    error,
    create,
    update,
    remove,
    setRating,
    setCategory,
    permanentDelete,
  } = useMemories();

  // Local state
  const [viewTab, setViewTab] = useState<ViewTab>('active');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<MemoryCategory | 'all'>('all');
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingMemory, setEditingMemory] = useState<Memory | null>(null);

  // Filter memories based on search and category
  const filteredMemories = useMemo(() => {
    const sourceMemories = viewTab === 'active' ? memories : deletedMemories;
    let result = sourceMemories;

    // Apply category filter
    if (selectedCategory !== 'all') {
      result = result.filter((m) => m.category === selectedCategory);
    }

    // Apply search filter
    if (searchQuery.trim()) {
      const lowerQuery = searchQuery.toLowerCase();
      result = result.filter((m) => m.content.toLowerCase().includes(lowerQuery));
    }

    // Sort by updated date
    return result.sort((a, b) => b.updatedAt - a.updatedAt);
  }, [memories, deletedMemories, viewTab, searchQuery, selectedCategory]);

  // Handlers
  const handleCreateNew = () => {
    setEditingMemory(null);
    setEditorOpen(true);
  };

  const handleEdit = (memory: Memory) => {
    setEditingMemory(memory);
    setEditorOpen(true);
  };

  const handleDelete = async (memory: Memory, reason?: MemoryDeletionReason) => {
    try {
      await remove(memory.id, reason);
    } catch (err) {
      console.error('Failed to delete memory:', err);
    }
  };

  const handleRatingChange = async (memory: Memory, stars: number) => {
    try {
      await setRating(memory.id, stars);
    } catch (err) {
      console.error('Failed to update rating:', err);
    }
  };

  const handleCategoryChange = async (memory: Memory, category: MemoryCategory) => {
    try {
      await setCategory(memory.id, category);
    } catch (err) {
      console.error('Failed to update category:', err);
    }
  };

  const handleRestore = async (_memory: Memory) => {
    // Restore endpoint no longer exists in backend
    console.warn('Memory restore is not supported in the current backend');
  };

  const handlePermanentDelete = async (memory: Memory) => {
    if (confirm(`Permanently delete this memory?\n\nThis cannot be undone.`)) {
      try {
        await permanentDelete(memory.id);
      } catch (err) {
        console.error('Failed to permanently delete memory:', err);
      }
    }
  };

  const handleSave = async (content: string, category: MemoryCategory) => {
    try {
      if (editingMemory) {
        await update(editingMemory.id, content, category);
      } else {
        await create(content, category);
      }
      setEditorOpen(false);
      setEditingMemory(null);
    } catch (err) {
      console.error('Failed to save memory:', err);
    }
  };

  const handleCancel = () => {
    setEditorOpen(false);
    setEditingMemory(null);
  };

  const activeCount = memories.length;
  const deletedCount = deletedMemories.length;

  return (
    <div className={cls('layout-stack h-full bg-background min-h-0', className)}>
      {/* Header with tabs inline */}
      <header className="h-12 border-b border-border flex items-center justify-between px-3 shrink-0 gap-3">
        <div className="flex items-center gap-2">
          <svg
            className="w-4 h-4 text-accent"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
            />
          </svg>
          <h2 className="font-medium text-default text-sm">Memories</h2>
        </div>

        {/* Tabs - inline in header */}
        <div className="flex gap-0.5 p-0.5 bg-sunken rounded">
          <button
            onClick={() => setViewTab('active')}
            className={cls(
              'px-2 py-1 rounded text-xs font-medium transition-colors',
              viewTab === 'active'
                ? 'bg-surface text-default shadow-sm'
                : 'text-muted hover:text-default'
            )}
          >
            Active
            <span className={cls(
              'ml-1 px-1 py-0.5 rounded text-[10px]',
              viewTab === 'active' ? 'bg-accent-subtle text-accent' : 'text-muted'
            )}>
              {activeCount}
            </span>
          </button>
          <button
            onClick={() => setViewTab('deleted')}
            className={cls(
              'px-2 py-1 rounded text-xs font-medium transition-colors',
              viewTab === 'deleted'
                ? 'bg-surface text-default shadow-sm'
                : 'text-muted hover:text-default'
            )}
          >
            Deleted
            {deletedCount > 0 && (
              <span className={cls(
                'ml-1 px-1 py-0.5 rounded text-[10px]',
                viewTab === 'deleted' ? 'bg-error-subtle text-error' : 'text-muted'
              )}>
                {deletedCount}
              </span>
            )}
          </button>
        </div>

        <button
          onClick={handleCreateNew}
          className={cls(
            'px-2 py-1 rounded bg-accent text-on-emphasis text-xs font-medium flex items-center gap-1',
            'hover:bg-accent-hover transition-colors'
          )}
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add
        </button>
      </header>

      {/* Search and filters */}
      <div className="px-3 py-2 border-b border-border shrink-0">
        <MemorySearch
          searchQuery={searchQuery}
          selectedCategory={selectedCategory}
          onSearchChange={setSearchQuery}
          onCategoryChange={setSelectedCategory}
        />
      </div>

      {/* Error message */}
      {error && (
        <div className="mx-3 mt-2 p-2 rounded border border-error bg-error-subtle text-error text-xs">
          {error}
        </div>
      )}

      {/* Memory list */}
      <div className="flex-1 overflow-y-auto px-3 py-2 min-h-0">
        {isFetching ? (
          <div className="flex items-center justify-center p-8">
            <div className="layout-center-gap text-muted">
              <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              <span className="text-sm">Loading memories...</span>
            </div>
          </div>
        ) : (
          <MemoryList
            memories={filteredMemories}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onRatingChange={handleRatingChange}
            onCategoryChange={handleCategoryChange}
            isLoading={isLoading}
            showDeletedActions={viewTab === 'deleted'}
            onRestore={handleRestore}
            onPermanentDelete={handlePermanentDelete}
          />
        )}
      </div>

      {/* Editor modal */}
      <MemoryEditor
        memory={editingMemory}
        isOpen={editorOpen}
        onSave={handleSave}
        onCancel={handleCancel}
        isLoading={isLoading}
      />
    </div>
  );
};

// Named exports for individual components
export { MemorySearch } from './MemorySearch';
export { MemoryList } from './MemoryList';
export { MemoryEditor } from './MemoryEditor';
export { MemoryDetail } from './MemoryDetail';
