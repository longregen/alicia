import React, { useState, useEffect } from 'react';
import { useLocation } from 'wouter';
import { cls } from '../../../utils/cls';
import type { MemoryCategory } from '../../../stores/memoryStore';
import { useMemoryStore } from '../../../stores/memoryStore';
import { useSidebarStore } from '../../../stores/sidebarStore';
import { MemoryEditor } from './MemoryEditor';
import StarRating, { importanceToStar, starToImportance } from '../../atoms/StarRating';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '../../atoms/Dialog';
import Button from '../../atoms/Button';
import type { MemoryDeletionReason } from '../../../hooks/useMemories';

export interface MemoryDetailProps {
  memoryId: string;
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
  history: {
    bg: 'bg-muted/10',
    text: 'text-muted',
    border: 'border-muted',
  },
};

const formatDate = (timestamp: number): string => {
  return new Date(timestamp).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

const formatRelativeDate = (timestamp: number): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) return 'just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}d ago`;
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
};

/**
 * MemoryDetail component for viewing a single memory.
 */
export const MemoryDetail: React.FC<MemoryDetailProps> = ({
  memoryId,
  className = '',
}) => {
  const [, navigate] = useLocation();
  const [editorOpen, setEditorOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Deletion dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDeletionReason, setSelectedDeletionReason] = useState<MemoryDeletionReason | null>(null);

  // Sidebar state for mobile hamburger
  const openSidebar = useSidebarStore((state) => state.setOpen);

  // Get memory from store
  const memory = useMemoryStore((state) => state.memories[memoryId]);
  const updateMemory = useMemoryStore((state) => state.updateMemory);
  const deleteMemory = useMemoryStore((state) => state.deleteMemory);
  const pinMemory = useMemoryStore((state) => state.pinMemory);
  const archiveMemory = useMemoryStore((state) => state.archiveMemory);

  // If memory doesn't exist in store, try to fetch it
  const setMemory = useMemoryStore((state) => state.setMemory);

  useEffect(() => {
    if (!memory) {
      // Try to fetch memory from API
      const fetchMemory = async () => {
        setIsLoading(true);
        try {
          const response = await fetch(`/api/v1/memories/${memoryId}`);
          if (!response.ok) {
            if (response.status === 404) {
              setError('Memory not found');
            } else {
              throw new Error(`Failed to fetch memory: ${response.status}`);
            }
            return;
          }
          const apiMemory = await response.json();
          // Convert and store
          const category: MemoryCategory = apiMemory.tags?.[0] || 'fact';
          setMemory({
            id: apiMemory.id,
            content: apiMemory.content,
            category,
            tags: apiMemory.tags || [],
            importance: apiMemory.importance || 0.5,
            createdAt: apiMemory.created_at * 1000,
            updatedAt: apiMemory.updated_at * 1000,
            pinned: apiMemory.pinned || false,
            archived: apiMemory.archived || false,
            usageCount: 0,
          });
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Failed to load memory');
        } finally {
          setIsLoading(false);
        }
      };
      fetchMemory();
    }
  }, [memoryId, memory, setMemory]);

  const handleBack = () => {
    navigate('/memory');
  };

  const handleEdit = () => {
    setEditorOpen(true);
  };

  const handleSave = async (content: string, category: MemoryCategory) => {
    setIsLoading(true);
    try {
      const response = await fetch(`/api/v1/memories/${memoryId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content }),
      });

      if (!response.ok) {
        throw new Error(`Failed to update memory: ${response.status}`);
      }

      updateMemory(memoryId, { content, category });
      setEditorOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update memory');
    } finally {
      setIsLoading(false);
    }
  };

  const handlePin = async () => {
    if (!memory) return;
    setIsLoading(true);
    try {
      const response = await fetch(`/api/v1/memories/${memoryId}/pin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pinned: !memory.pinned }),
      });

      if (!response.ok) {
        throw new Error(`Failed to pin memory: ${response.status}`);
      }

      pinMemory(memoryId, !memory.pinned);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to pin memory');
    } finally {
      setIsLoading(false);
    }
  };

  const handleArchive = async () => {
    if (!memory) return;
    if (!confirm('Archive this memory?')) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/v1/memories/${memoryId}/archive`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(`Failed to archive memory: ${response.status}`);
      }

      archiveMemory(memoryId);
      navigate('/memory');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to archive memory');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteClick = () => {
    setSelectedDeletionReason(null);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    setIsLoading(true);
    try {
      const options: RequestInit = {
        method: 'DELETE',
      };

      if (selectedDeletionReason) {
        options.headers = { 'Content-Type': 'application/json' };
        options.body = JSON.stringify({ reason: selectedDeletionReason });
      }

      const response = await fetch(`/api/v1/memories/${memoryId}`, options);

      if (!response.ok) {
        throw new Error(`Failed to delete memory: ${response.status}`);
      }

      deleteMemory(memoryId);
      setDeleteDialogOpen(false);
      navigate('/memory');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete memory');
    } finally {
      setIsLoading(false);
    }
  };

  const handleRatingChange = async (stars: number) => {
    if (!memory) return;
    setIsLoading(true);
    try {
      const importance = starToImportance(stars);
      const response = await fetch(`/api/v1/memories/${memoryId}/importance`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ importance }),
      });

      if (!response.ok) {
        throw new Error(`Failed to update importance: ${response.status}`);
      }

      updateMemory(memoryId, { importance });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update importance');
    } finally {
      setIsLoading(false);
    }
  };

  // Loading state
  if (isLoading && !memory) {
    return (
      <div className={cls('flex items-center justify-center p-8', className)}>
        <div className="flex items-center gap-2 text-muted">
          <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          <span className="text-sm">Loading memory...</span>
        </div>
      </div>
    );
  }

  // Error state
  if (error && !memory) {
    return (
      <div className={cls('flex flex-col items-center justify-center p-8', className)}>
        <svg
          className="w-16 h-16 text-error mb-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <p className="text-error text-lg font-medium mb-2">{error}</p>
        <button
          onClick={handleBack}
          className="text-accent hover:text-accent-hover transition-colors"
        >
          Back to Memory List
        </button>
      </div>
    );
  }

  // No memory found
  if (!memory) {
    return null;
  }

  const categoryStyle = categoryColors[memory.category];

  return (
    <div className={cls('h-full flex flex-col', className)}>
      {/* Header */}
      <header className="flex items-center justify-between p-4 border-b border-border shrink-0">
        <div className="flex items-center gap-3">
          <button
            onClick={() => openSidebar(true)}
            className="lg:hidden p-2 -ml-2 hover:bg-elevated rounded-md transition-colors"
            aria-label="Open sidebar"
          >
            <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
          <button
            onClick={handleBack}
            className="p-2 rounded hover:bg-surface-hover transition-colors text-muted hover:text-default"
            title="Back to list"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h2 className="font-medium text-default">Memory Details</h2>
          {memory.pinned && (
            <span className="text-accent" title="Pinned">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6zM15.657 5.404a.75.75 0 10-1.06-1.06l-1.061 1.06a.75.75 0 001.06 1.06l1.06-1.06zM6.464 14.596a.75.75 0 10-1.06-1.06l-1.06 1.06a.75.75 0 001.06 1.06l1.06-1.06zM18 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 0118 10zM5 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 015 10zM14.596 15.657a.75.75 0 001.06-1.06l-1.06-1.061a.75.75 0 10-1.06 1.06l1.06 1.06zM5.404 6.464a.75.75 0 001.06-1.06l-1.06-1.06a.75.75 0 10-1.061 1.06l1.06 1.06z" />
              </svg>
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handlePin}
            disabled={isLoading}
            className={cls(
              'p-2 rounded transition-colors disabled:opacity-50',
              memory.pinned
                ? 'text-accent hover:bg-surface-hover'
                : 'text-muted hover:text-accent hover:bg-surface-hover'
            )}
            title={memory.pinned ? 'Unpin' : 'Pin'}
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path d="M10 2a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 2zM10 15a.75.75 0 01.75.75v1.5a.75.75 0 01-1.5 0v-1.5A.75.75 0 0110 15zM10 7a3 3 0 100 6 3 3 0 000-6zM15.657 5.404a.75.75 0 10-1.06-1.06l-1.061 1.06a.75.75 0 001.06 1.06l1.06-1.06zM6.464 14.596a.75.75 0 10-1.06-1.06l-1.06 1.06a.75.75 0 001.06 1.06l1.06-1.06zM18 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 0118 10zM5 10a.75.75 0 01-.75.75h-1.5a.75.75 0 010-1.5h1.5A.75.75 0 015 10zM14.596 15.657a.75.75 0 001.06-1.06l-1.06-1.061a.75.75 0 10-1.06 1.06l1.06 1.06zM5.404 6.464a.75.75 0 001.06-1.06l-1.06-1.06a.75.75 0 10-1.061 1.06l1.06 1.06z" />
            </svg>
          </button>
          <button
            onClick={handleEdit}
            disabled={isLoading}
            className="p-2 rounded text-muted hover:text-default hover:bg-surface-hover transition-colors disabled:opacity-50"
            title="Edit"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
              />
            </svg>
          </button>
          <button
            onClick={handleArchive}
            disabled={isLoading}
            className="p-2 rounded text-muted hover:text-warning hover:bg-surface-hover transition-colors disabled:opacity-50"
            title="Archive"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
              />
            </svg>
          </button>
          <button
            onClick={handleDeleteClick}
            disabled={isLoading}
            className="p-2 rounded text-muted hover:text-error hover:bg-surface-hover transition-colors disabled:opacity-50"
            title="Delete"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
        </div>
      </header>

      {/* Error banner */}
      {error && (
        <div className="mx-4 mt-4 p-3 rounded border border-error bg-error-subtle text-error text-sm">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto space-y-6">
          {/* Category and metadata row */}
          <div className="flex items-center gap-4 flex-wrap">
            <span
              className={cls(
                'px-3 py-1.5 rounded text-sm font-medium border',
                categoryStyle.bg,
                categoryStyle.text,
                categoryStyle.border
              )}
            >
              {memory.category}
            </span>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted">Importance:</span>
              <StarRating
                rating={importanceToStar(memory.importance)}
                onRate={handleRatingChange}
                isLoading={isLoading}
                showValue
              />
            </div>
            <span className="flex items-center gap-1 text-sm text-muted">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
              </svg>
              Used {memory.usageCount} {memory.usageCount === 1 ? 'time' : 'times'}
            </span>
          </div>

          {/* Main content */}
          <div className="bg-surface border border-border rounded-lg p-6">
            <p className="text-default text-base leading-relaxed whitespace-pre-wrap">
              {memory.content}
            </p>
          </div>

          {/* Tags */}
          {memory.tags && memory.tags.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-muted mb-2">Tags</h3>
              <div className="flex flex-wrap gap-2">
                {memory.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="px-2.5 py-1 rounded text-sm bg-sunken text-muted border border-muted"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Timestamps */}
          <div className="border-t border-border pt-4 space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted">Created</span>
              <span className="text-default" title={formatDate(memory.createdAt)}>
                {formatRelativeDate(memory.createdAt)} ({formatDate(memory.createdAt)})
              </span>
            </div>
            {memory.updatedAt > memory.createdAt && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted">Last updated</span>
                <span className="text-default" title={formatDate(memory.updatedAt)}>
                  {formatRelativeDate(memory.updatedAt)} ({formatDate(memory.updatedAt)})
                </span>
              </div>
            )}
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted">Memory ID</span>
              <code className="text-xs text-muted bg-sunken px-2 py-1 rounded font-mono">
                {memory.id}
              </code>
            </div>
          </div>
        </div>
      </div>

      {/* Editor modal */}
      <MemoryEditor
        memory={memory}
        isOpen={editorOpen}
        onSave={handleSave}
        onCancel={() => setEditorOpen(false)}
        isLoading={isLoading}
      />

      {/* Delete confirmation dialog with reason selection */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Memory</DialogTitle>
            <DialogDescription>
              <span className="block mt-2 p-3 bg-sunken rounded text-sm text-default">
                "{memory.content.slice(0, 100)}{memory.content.length > 100 ? '...' : ''}"
              </span>
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            <p className="text-sm text-muted mb-3">Why are you deleting this memory? (optional)</p>
            <div className="grid grid-cols-2 gap-2">
              {[
                { value: 'wrong' as const, label: 'Wrong', icon: '❌' },
                { value: 'useless' as const, label: 'Useless', icon: '🗑️' },
                { value: 'old' as const, label: 'Outdated', icon: '📅' },
                { value: 'duplicate' as const, label: 'Duplicate', icon: '📋' },
                { value: 'other' as const, label: 'Other', icon: '💭' },
              ].map((reason) => (
                <button
                  key={reason.value}
                  onClick={() => setSelectedDeletionReason(
                    selectedDeletionReason === reason.value ? null : reason.value
                  )}
                  className={cls(
                    'flex items-center gap-2 px-3 py-2 rounded-md border transition-colors text-sm',
                    selectedDeletionReason === reason.value
                      ? 'border-accent bg-accent/10 text-accent'
                      : 'border-border hover:border-muted hover:bg-surface-hover text-default'
                  )}
                >
                  <span>{reason.icon}</span>
                  <span>{reason.label}</span>
                </button>
              ))}
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={isLoading}
            >
              {isLoading ? 'Deleting...' : 'Delete Memory'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
