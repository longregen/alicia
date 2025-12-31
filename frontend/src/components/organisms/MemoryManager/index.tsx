import React, { useState, useMemo } from 'react';
import { cls } from '../../../utils/cls';
import type { Memory, MemoryCategory } from '../../../stores/memoryStore';
import { useMemories } from '../../../hooks/useMemories';
import { MemorySearch } from './MemorySearch';
import { MemoryList } from './MemoryList';
import { MemoryEditor } from './MemoryEditor';

export interface MemoryManagerProps {
  className?: string;
}

/**
 * MemoryManager organism component for managing global memories.
 *
 * Features:
 * - Display all memories with category, pinned status, creation date
 * - Search/filter by content or category
 * - Create new memories (content + category)
 * - Edit, pin, archive, delete memories
 * - Uses memoryStore with API integration via useMemories hook
 *
 * @example
 * ```tsx
 * <MemoryManager />
 * ```
 */
export const MemoryManager: React.FC<MemoryManagerProps> = ({ className = '' }) => {
  const {
    memories,
    isLoading,
    isFetching,
    error,
    create,
    update,
    remove,
    pin,
    archive,
    search,
    filterByCategory,
  } = useMemories();

  // Local state
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<MemoryCategory | 'all'>('all');
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingMemory, setEditingMemory] = useState<Memory | null>(null);

  // Filter memories based on search and category
  const filteredMemories = useMemo(() => {
    let result = memories;

    // Apply category filter
    if (selectedCategory !== 'all') {
      result = filterByCategory(selectedCategory);
    }

    // Apply search filter
    if (searchQuery.trim()) {
      result = search(searchQuery);
      // Further filter by category if needed
      if (selectedCategory !== 'all') {
        result = result.filter((m) => m.category === selectedCategory);
      }
    }

    // Sort: pinned first, then by updated date
    return result.sort((a, b) => {
      if (a.pinned && !b.pinned) return -1;
      if (!a.pinned && b.pinned) return 1;
      return b.updatedAt - a.updatedAt;
    });
  }, [memories, searchQuery, selectedCategory, search, filterByCategory]);

  // Handlers
  const handleCreateNew = () => {
    setEditingMemory(null);
    setEditorOpen(true);
  };

  const handleEdit = (memory: Memory) => {
    setEditingMemory(memory);
    setEditorOpen(true);
  };

  const handleDelete = async (memory: Memory) => {
    if (confirm(`Are you sure you want to delete this memory?\n\n"${memory.content}"`)) {
      try {
        await remove(memory.id);
      } catch (err) {
        console.error('Failed to delete memory:', err);
      }
    }
  };

  const handlePin = (memory: Memory) => {
    pin(memory.id, !memory.pinned);
  };

  const handleArchive = async (memory: Memory) => {
    if (confirm(`Archive this memory?\n\n"${memory.content}"`)) {
      try {
        archive(memory.id);
      } catch (err) {
        console.error('Failed to archive memory:', err);
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

  return (
    <div className={cls('flex flex-col gap-4 h-full', className)}>
      {/* Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-lg font-semibold text-default">Memory Manager</h1>
        <p className="text-sm text-muted">
          Manage global memories that persist across conversations
        </p>
      </div>

      {/* Search and filters */}
      <MemorySearch
        searchQuery={searchQuery}
        selectedCategory={selectedCategory}
        onSearchChange={setSearchQuery}
        onCategoryChange={setSelectedCategory}
        onCreateNew={handleCreateNew}
      />

      {/* Error message */}
      {error && (
        <div className="p-3 rounded border border-error bg-error-subtle text-error text-sm">
          {error}
        </div>
      )}

      {/* Loading indicator */}
      {isFetching && (
        <div className="flex items-center justify-center p-4">
          <div className="flex items-center gap-2 text-muted">
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
      )}

      {/* Memory list */}
      {!isFetching && (
        <div className="flex-1 overflow-y-auto">
          <MemoryList
            memories={filteredMemories}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onPin={handlePin}
            onArchive={handleArchive}
            isLoading={isLoading}
          />
        </div>
      )}

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
