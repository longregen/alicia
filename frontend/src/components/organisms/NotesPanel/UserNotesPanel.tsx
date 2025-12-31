import React, { useState } from 'react';
import { cls } from '../../../utils/cls';
import { useNotes } from '../../../hooks/useNotes';
import type { NoteCategory } from '../../../stores/notesStore';

/**
 * UserNotesPanel organism component.
 *
 * Displays and manages user notes for a specific target (message, tool_use, reasoning).
 * Provides functionality to:
 * - View all notes for a target
 * - Add new notes with category selection
 * - Edit existing notes
 * - Delete notes
 */

export interface UserNotesPanelProps {
  /** Type of target element */
  targetType: 'message' | 'tool_use' | 'reasoning';
  /** ID of the target element */
  targetId: string;
  /** Additional CSS classes */
  className?: string;
  /** Whether to show in compact mode */
  compact?: boolean;
}

const CATEGORY_LABELS: Record<NoteCategory, string> = {
  improvement: 'Improvement',
  correction: 'Correction',
  context: 'Context',
  general: 'General',
};

const CATEGORY_COLORS: Record<NoteCategory, string> = {
  improvement: 'bg-accent-subtle text-accent border-accent',
  correction: 'bg-error-subtle text-error border-error',
  context: 'bg-warning-subtle text-warning border-warning',
  general: 'bg-surface text-default border',
};

const UserNotesPanel: React.FC<UserNotesPanelProps> = ({
  targetType,
  targetId,
  className = '',
  compact = false,
}) => {
  const { notes, addNote, updateNote, deleteNote, isLoading, error } = useNotes(targetType, targetId);

  const [isAddingNote, setIsAddingNote] = useState(false);
  const [newNoteContent, setNewNoteContent] = useState('');
  const [newNoteCategory, setNewNoteCategory] = useState<NoteCategory>('general');
  const [editingNoteId, setEditingNoteId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState('');

  const handleAddNoteClick = () => {
    setIsAddingNote(true);
  };

  const handleCancelAdd = () => {
    setIsAddingNote(false);
    setNewNoteContent('');
    setNewNoteCategory('general');
  };

  const handleSubmitAdd = async () => {
    if (newNoteContent.trim()) {
      await addNote(newNoteContent, newNoteCategory);
      handleCancelAdd();
    }
  };

  const handleStartEdit = (noteId: string, currentContent: string) => {
    setEditingNoteId(noteId);
    setEditContent(currentContent);
  };

  const handleCancelEdit = () => {
    setEditingNoteId(null);
    setEditContent('');
  };

  const handleSubmitEdit = async (noteId: string) => {
    if (editContent.trim()) {
      await updateNote(noteId, editContent);
      handleCancelEdit();
    }
  };

  const handleDelete = async (noteId: string) => {
    if (window.confirm('Are you sure you want to delete this note?')) {
      await deleteNote(noteId);
    }
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className={cls(
      'flex',
      'flex-col',
      'gap-3',
      'bg-surface',
      'border',
      'rounded-lg',
      compact ? 'p-3' : 'p-4',
      className
    )}>
      {/* Header */}
      <div className={cls('flex', 'items-center', 'justify-between')}>
        <h3 className={cls('font-semibold', compact ? 'text-sm' : 'text-base', 'text-default')}>
          Notes {notes.length > 0 && <span className="text-muted">({notes.length})</span>}
        </h3>
        {!isAddingNote && (
          <button
            onClick={handleAddNoteClick}
            disabled={isLoading}
            className={cls(
              'flex',
              'items-center',
              'gap-1',
              'px-2 py-1',
              'text-sm',
              'rounded',
              'bg-accent',
              'text-on-emphasis',
              'hover:bg-accent-hover',
              'transition-colors',
              'duration-200',
              isLoading ? 'opacity-50 cursor-not-allowed' : ''
            )}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Note
          </button>
        )}
      </div>

      {/* Error display */}
      {error && (
        <div className={cls(
          'p-2',
          'rounded',
          'bg-error-subtle',
          'text-error',
          'text-sm'
        )}>
          {error}
        </div>
      )}

      {/* Add note form */}
      {isAddingNote && (
        <div className={cls('flex', 'flex-col', 'gap-2', 'p-3', 'bg-sunken', 'rounded-lg')}>
          <div className={cls('flex', 'gap-2')}>
            {(Object.keys(CATEGORY_LABELS) as NoteCategory[]).map((category) => (
              <button
                key={category}
                onClick={() => setNewNoteCategory(category)}
                className={cls(
                  'px-2 py-1',
                  'text-xs',
                  'rounded',
                  'border',
                  'transition-colors',
                  'duration-200',
                  newNoteCategory === category
                    ? CATEGORY_COLORS[category]
                    : 'bg-transparent border text-muted hover:bg-surface'
                )}
              >
                {CATEGORY_LABELS[category]}
              </button>
            ))}
          </div>
          <textarea
            value={newNoteContent}
            onChange={(e) => setNewNoteContent(e.target.value)}
            placeholder="Write your note here..."
            rows={3}
            className={cls(
              'w-full',
              'p-2',
              'rounded',
              'border',
              'bg-surface',
              'text-default',
              'text-sm',
              'focus:outline-none focus:ring-2 focus:ring-accent'
            )}
          />
          <div className={cls('flex', 'gap-2', 'justify-between')}>
            <div className={cls('flex', 'gap-2')}>
              <button
                onClick={handleSubmitAdd}
                disabled={!newNoteContent.trim() || isLoading}
                className={cls(
                  'px-3 py-1',
                  'text-sm',
                  'rounded',
                  'bg-accent',
                  'text-on-emphasis',
                  'hover:bg-accent-hover',
                  'transition-colors',
                  'duration-200',
                  'disabled:opacity-50 disabled:cursor-not-allowed'
                )}
              >
                Save
              </button>
              <button
                onClick={handleCancelAdd}
                disabled={isLoading}
                className={cls(
                  'px-3 py-1',
                  'text-sm',
                  'rounded',
                  'bg-surface',
                  'text-default',
                  'hover:bg-sunken',
                  'transition-colors',
                  'duration-200',
                  'border'
                )}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Notes list */}
      {notes.length === 0 && !isAddingNote ? (
        <div className={cls('text-center', 'text-muted', 'text-sm', 'py-3')}>
          No notes yet. Add one to get started.
        </div>
      ) : (
        <div className={cls('flex', 'flex-col', 'gap-2')}>
          {notes.map((note) => (
            <div
              key={note.id}
              className={cls(
                'flex',
                'flex-col',
                'gap-2',
                'p-3',
                'border',
                'rounded-lg',
                'bg-surface'
              )}
            >
              {/* Note header */}
              <div className={cls('flex', 'items-center', 'justify-between')}>
                <span className={cls('px-2 py-0.5', 'text-xs', 'rounded', 'border', CATEGORY_COLORS[note.category])}>
                  {CATEGORY_LABELS[note.category]}
                </span>
                <span className={cls('text-xs', 'text-muted')}>
                  {formatTimestamp(note.createdAt)}
                </span>
              </div>

              {/* Note content */}
              {editingNoteId === note.id ? (
                <div className={cls('flex', 'flex-col', 'gap-2')}>
                  <textarea
                    value={editContent}
                    onChange={(e) => setEditContent(e.target.value)}
                    rows={3}
                    className={cls(
                      'w-full',
                      'p-2',
                      'rounded',
                      'border',
                      'bg-surface',
                      'text-default',
                      'text-sm',
                      'focus:outline-none focus:ring-2 focus:ring-accent'
                    )}
                  />
                  <div className={cls('flex', 'gap-2')}>
                    <button
                      onClick={() => handleSubmitEdit(note.id)}
                      disabled={!editContent.trim() || isLoading}
                      className={cls(
                        'px-2 py-1',
                        'text-xs',
                        'rounded',
                        'bg-accent',
                        'text-on-emphasis',
                        'hover:bg-accent-hover',
                        'transition-colors',
                        'duration-200',
                        'disabled:opacity-50 disabled:cursor-not-allowed'
                      )}
                    >
                      Save
                    </button>
                    <button
                      onClick={handleCancelEdit}
                      className={cls(
                        'px-2 py-1',
                        'text-xs',
                        'rounded',
                        'bg-surface',
                        'text-default',
                        'hover:bg-sunken',
                        'transition-colors',
                        'duration-200',
                        'border'
                      )}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <p className={cls('text-sm', 'text-default', 'whitespace-pre-wrap')}>
                    {note.content}
                  </p>

                  {/* Note actions */}
                  <div className={cls('flex', 'gap-2', 'items-center')}>
                    <button
                      onClick={() => handleStartEdit(note.id, note.content)}
                      disabled={isLoading}
                      className={cls(
                        'text-xs',
                        'text-muted',
                        'hover:text-accent',
                        'transition-colors',
                        'duration-200',
                        isLoading ? 'opacity-50 cursor-not-allowed' : ''
                      )}
                    >
                      Edit
                    </button>
                    <span className="text-muted">•</span>
                    <button
                      onClick={() => handleDelete(note.id)}
                      disabled={isLoading}
                      className={cls(
                        'text-xs',
                        'text-muted',
                        'hover:text-error',
                        'transition-colors',
                        'duration-200',
                        isLoading ? 'opacity-50 cursor-not-allowed' : ''
                      )}
                    >
                      Delete
                    </button>
                    {note.updatedAt !== note.createdAt && (
                      <>
                        <span className="text-muted">•</span>
                        <span className={cls('text-xs', 'text-muted')}>
                          edited {formatTimestamp(note.updatedAt)}
                        </span>
                      </>
                    )}
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Loading indicator */}
      {isLoading && (
        <div className={cls('flex', 'items-center', 'justify-center', 'gap-2', 'py-2')}>
          <div className="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin" />
          <span className={cls('text-sm', 'text-muted')}>Processing...</span>
        </div>
      )}
    </div>
  );
};

export default UserNotesPanel;
