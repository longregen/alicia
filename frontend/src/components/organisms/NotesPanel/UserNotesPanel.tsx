import React, { useState } from 'react';
import { cls } from '../../../utils/cls';
import { CSS } from '../../../utils/constants';
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
  improvement: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border-blue-300 dark:border-blue-700',
  correction: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 border-red-300 dark:border-red-700',
  context: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 border-purple-300 dark:border-purple-700',
  general: 'bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-700',
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
      CSS.flex,
      CSS.flexCol,
      CSS.gap3,
      'bg-surface-bg',
      'border',
      'border-gray-300',
      'dark:border-gray-600',
      CSS.roundedLg,
      compact ? CSS.p3 : CSS.p4,
      className
    )}>
      {/* Header */}
      <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyBetween)}>
        <h3 className={cls(CSS.fontSemibold, compact ? CSS.textSm : CSS.textBase, CSS.textPrimary)}>
          Notes {notes.length > 0 && <span className={CSS.textMuted}>({notes.length})</span>}
        </h3>
        {!isAddingNote && (
          <button
            onClick={handleAddNoteClick}
            disabled={isLoading}
            className={cls(
              CSS.flex,
              CSS.itemsCenter,
              CSS.gap1,
              'px-2 py-1',
              CSS.textSm,
              CSS.rounded,
              'bg-primary-blue',
              'text-white-text',
              'hover:bg-primary-blue-hover',
              CSS.transitionColors,
              CSS.duration200,
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
          CSS.p2,
          CSS.rounded,
          'bg-red-100 dark:bg-red-900/30',
          'text-red-700 dark:text-red-300',
          CSS.textSm
        )}>
          {error}
        </div>
      )}

      {/* Add note form */}
      {isAddingNote && (
        <div className={cls(CSS.flex, CSS.flexCol, CSS.gap2, CSS.p3, 'bg-gray-50 dark:bg-gray-800', CSS.roundedLg)}>
          <div className={cls(CSS.flex, CSS.gap2)}>
            {(Object.keys(CATEGORY_LABELS) as NoteCategory[]).map((category) => (
              <button
                key={category}
                onClick={() => setNewNoteCategory(category)}
                className={cls(
                  'px-2 py-1',
                  CSS.textXs,
                  CSS.rounded,
                  'border',
                  CSS.transitionColors,
                  CSS.duration200,
                  newNoteCategory === category
                    ? CATEGORY_COLORS[category]
                    : 'bg-transparent border-gray-300 dark:border-gray-600 text-muted-text hover:bg-gray-100 dark:hover:bg-gray-700'
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
              CSS.wFull,
              CSS.p2,
              CSS.rounded,
              'border',
              'border-gray-300 dark:border-gray-600',
              'bg-white dark:bg-gray-900',
              CSS.textPrimary,
              CSS.textSm,
              'focus:outline-none focus:ring-2 focus:ring-primary-blue'
            )}
          />
          <div className={cls(CSS.flex, CSS.gap2, CSS.justifyBetween)}>
            <div className={cls(CSS.flex, CSS.gap2)}>
              <button
                onClick={handleSubmitAdd}
                disabled={!newNoteContent.trim() || isLoading}
                className={cls(
                  'px-3 py-1',
                  CSS.textSm,
                  CSS.rounded,
                  'bg-primary-blue',
                  'text-white-text',
                  'hover:bg-primary-blue-hover',
                  CSS.transitionColors,
                  CSS.duration200,
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
                  CSS.textSm,
                  CSS.rounded,
                  'bg-gray-200 dark:bg-gray-700',
                  CSS.textPrimary,
                  'hover:bg-gray-300 dark:hover:bg-gray-600',
                  CSS.transitionColors,
                  CSS.duration200
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
        <div className={cls(CSS.textCenter, CSS.textMuted, CSS.textSm, CSS.py3)}>
          No notes yet. Add one to get started.
        </div>
      ) : (
        <div className={cls(CSS.flex, CSS.flexCol, CSS.gap2)}>
          {notes.map((note) => (
            <div
              key={note.id}
              className={cls(
                CSS.flex,
                CSS.flexCol,
                CSS.gap2,
                CSS.p3,
                'border',
                'border-gray-200 dark:border-gray-700',
                CSS.roundedLg,
                'bg-white dark:bg-gray-900'
              )}
            >
              {/* Note header */}
              <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyBetween)}>
                <span className={cls('px-2 py-0.5', CSS.textXs, CSS.rounded, 'border', CATEGORY_COLORS[note.category])}>
                  {CATEGORY_LABELS[note.category]}
                </span>
                <span className={cls(CSS.textXs, CSS.textMuted)}>
                  {formatTimestamp(note.createdAt)}
                </span>
              </div>

              {/* Note content */}
              {editingNoteId === note.id ? (
                <div className={cls(CSS.flex, CSS.flexCol, CSS.gap2)}>
                  <textarea
                    value={editContent}
                    onChange={(e) => setEditContent(e.target.value)}
                    rows={3}
                    className={cls(
                      CSS.wFull,
                      CSS.p2,
                      CSS.rounded,
                      'border',
                      'border-gray-300 dark:border-gray-600',
                      'bg-white dark:bg-gray-900',
                      CSS.textPrimary,
                      CSS.textSm,
                      'focus:outline-none focus:ring-2 focus:ring-primary-blue'
                    )}
                  />
                  <div className={cls(CSS.flex, CSS.gap2)}>
                    <button
                      onClick={() => handleSubmitEdit(note.id)}
                      disabled={!editContent.trim() || isLoading}
                      className={cls(
                        'px-2 py-1',
                        CSS.textXs,
                        CSS.rounded,
                        'bg-primary-blue',
                        'text-white-text',
                        'hover:bg-primary-blue-hover',
                        CSS.transitionColors,
                        CSS.duration200,
                        'disabled:opacity-50 disabled:cursor-not-allowed'
                      )}
                    >
                      Save
                    </button>
                    <button
                      onClick={handleCancelEdit}
                      className={cls(
                        'px-2 py-1',
                        CSS.textXs,
                        CSS.rounded,
                        'bg-gray-200 dark:bg-gray-700',
                        CSS.textPrimary,
                        'hover:bg-gray-300 dark:hover:bg-gray-600',
                        CSS.transitionColors,
                        CSS.duration200
                      )}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <p className={cls(CSS.textSm, CSS.textPrimary, 'whitespace-pre-wrap')}>
                    {note.content}
                  </p>

                  {/* Note actions */}
                  <div className={cls(CSS.flex, CSS.gap2, CSS.itemsCenter)}>
                    <button
                      onClick={() => handleStartEdit(note.id, note.content)}
                      disabled={isLoading}
                      className={cls(
                        CSS.textXs,
                        CSS.textMuted,
                        'hover:text-primary-blue',
                        CSS.transitionColors,
                        CSS.duration200,
                        isLoading ? 'opacity-50 cursor-not-allowed' : ''
                      )}
                    >
                      Edit
                    </button>
                    <span className={CSS.textMuted}>•</span>
                    <button
                      onClick={() => handleDelete(note.id)}
                      disabled={isLoading}
                      className={cls(
                        CSS.textXs,
                        CSS.textMuted,
                        'hover:text-error',
                        CSS.transitionColors,
                        CSS.duration200,
                        isLoading ? 'opacity-50 cursor-not-allowed' : ''
                      )}
                    >
                      Delete
                    </button>
                    {note.updatedAt !== note.createdAt && (
                      <>
                        <span className={CSS.textMuted}>•</span>
                        <span className={cls(CSS.textXs, CSS.textMuted)}>
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
        <div className={cls(CSS.flex, CSS.itemsCenter, CSS.justifyCenter, CSS.gap2, CSS.py2)}>
          <div className="w-4 h-4 border-2 border-primary-blue-glow border-t-transparent rounded-full animate-spin" />
          <span className={cls(CSS.textSm, CSS.textMuted)}>Processing...</span>
        </div>
      )}
    </div>
  );
};

export default UserNotesPanel;
