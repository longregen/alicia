import { useState, useCallback, useEffect, useRef } from 'react';
import { useNotes } from '../hooks/useNotes';
import { useSidebarStore } from '../stores/sidebarStore';

function NoteEditor({
  noteId,
  initialTitle,
  initialContent,
  onSave,
  onDelete,
}: {
  noteId: string;
  initialTitle: string;
  initialContent: string;
  onSave: (id: string, data: { title?: string; content?: string }) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}) {
  const [title, setTitle] = useState(initialTitle);
  const [content, setContent] = useState(initialContent);
  const saveTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const titleRef = useRef(title);
  const contentRef = useRef(content);

  const debouncedSave = useCallback(
    (data: { title?: string; content?: string }) => {
      if (saveTimeout.current) {
        clearTimeout(saveTimeout.current);
      }
      saveTimeout.current = setTimeout(() => {
        onSave(noteId, data);
      }, 500);
    },
    [noteId, onSave]
  );

  useEffect(() => {
    return () => {
      if (saveTimeout.current) {
        clearTimeout(saveTimeout.current);
        onSave(noteId, { title: titleRef.current, content: contentRef.current });
      }
    };
  }, [noteId, onSave]);

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newTitle = e.target.value;
    setTitle(newTitle);
    titleRef.current = newTitle;
    debouncedSave({ title: newTitle });
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setContent(newContent);
    contentRef.current = newContent;
    debouncedSave({ content: newContent });
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 border-b border-border">
        <input
          type="text"
          value={title}
          onChange={handleTitleChange}
          className="text-xl font-semibold bg-transparent border-none outline-none text-foreground w-full"
          placeholder="Note title..."
        />
        <button
          onClick={() => onDelete(noteId)}
          className="ml-4 p-2 text-muted-foreground hover:text-error rounded-md hover:bg-elevated transition-colors"
          title="Delete note"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>
      <textarea
        value={content}
        onChange={handleContentChange}
        className="flex-1 p-4 bg-transparent border-none outline-none text-foreground resize-none font-mono text-sm"
        placeholder="Write your note..."
      />
    </div>
  );
}

export function NotesPage() {
  const { notes, selectedNote, selectedNoteId, loading, createNote, updateNote, deleteNote, selectNote } = useNotes();
  const setSidebarOpen = useSidebarStore((state) => state.setOpen);

  return (
    <div className="flex h-full bg-background">
      <div className="w-72 border-r border-border flex flex-col">
        <div className="p-4 border-b border-border flex items-center gap-3">
          <button
            onClick={() => setSidebarOpen(true)}
            className="lg:hidden p-2 -ml-2 hover:bg-elevated rounded-md transition-colors"
            aria-label="Open sidebar"
          >
            <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
          <h2 className="text-lg font-semibold text-foreground flex-1">Notes</h2>
          <button
            onClick={createNote}
            className="p-2 hover:bg-elevated rounded-md transition-colors text-muted-foreground hover:text-foreground"
            title="New note"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          {loading && notes.length === 0 ? (
            <div className="p-4 text-center text-muted-foreground">Loading...</div>
          ) : notes.length === 0 ? (
            <div className="p-4 text-center text-muted-foreground">
              <p>No notes yet</p>
              <button
                onClick={createNote}
                className="mt-2 text-sm text-accent hover:underline"
              >
                Create your first note
              </button>
            </div>
          ) : (
            notes.map((note) => (
              <button
                key={note.id}
                onClick={() => selectNote(note.id)}
                className={`w-full text-left p-3 border-b border-border transition-colors ${
                  selectedNoteId === note.id
                    ? 'bg-sidebar-accent'
                    : 'hover:bg-elevated'
                }`}
              >
                <div className="font-medium text-foreground text-sm truncate">
                  {note.title || 'Untitled'}
                </div>
                <div className="text-xs text-muted-foreground mt-1 truncate">
                  {note.content ? note.content.slice(0, 60) : 'Empty note'}
                </div>
                <div className="text-xs text-muted-foreground mt-1">
                  {new Date(note.updated_at).toLocaleDateString()}
                </div>
              </button>
            ))
          )}
        </div>
      </div>

      <div className="flex-1 flex flex-col">
        {selectedNote ? (
          <NoteEditor
            key={selectedNote.id}
            noteId={selectedNote.id}
            initialTitle={selectedNote.title}
            initialContent={selectedNote.content}
            onSave={updateNote}
            onDelete={deleteNote}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <svg className="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p>Select a note or create a new one</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
