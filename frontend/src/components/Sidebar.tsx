import { useState } from 'react';
import { Conversation } from '../types/models';

interface SidebarProps {
  conversations: Conversation[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onNew: () => void;
  onDelete: (id: string) => void;
  onRenameConversation: (id: string, newTitle: string) => void;
  onSettings: () => void;
  loading: boolean;
}

export function Sidebar({
  conversations,
  selectedId,
  onSelect,
  onNew,
  onDelete,
  onRenameConversation,
  onSettings,
  loading,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');

  const handleStartEdit = (conv: Conversation) => {
    setEditingId(conv.id);
    setEditTitle(conv.title || 'New Conversation');
  };

  const handleSaveEdit = (id: string) => {
    if (editTitle.trim()) {
      onRenameConversation(id, editTitle.trim());
    }
    setEditingId(null);
    setEditTitle('');
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditTitle('');
  };

  const handleKeyDown = (e: React.KeyboardEvent, id: string) => {
    if (e.key === 'Enter') {
      handleSaveEdit(id);
    } else if (e.key === 'Escape') {
      handleCancelEdit();
    }
  };

  return (
    <div className="w-[300px] bg-surface text-default flex flex-col border-r border-default h-full">
      <div className="p-5 border-b border-default">
        <h2 className="mb-3 text-2xl font-semibold">Alicia</h2>
        <button
          onClick={onNew}
          className="btn btn-secondary w-full"
          data-testid="new-chat-btn"
        >
          New Chat
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-2.5">
        {loading && <div className="text-center text-muted p-5">Loading...</div>}
        {!loading && conversations.length === 0 && (
          <div className="text-center text-muted p-5">No conversations yet</div>
        )}
        {conversations.map(conv => (
          <div
            key={conv.id}
            className={`p-3 mb-2 bg-elevated rounded-md cursor-pointer flex justify-between items-center hover:bg-sunken transition-colors ${
              selectedId === conv.id ? 'bg-sunken border-l-2 border-accent' : ''
            }`}
            onClick={() => {
              if (editingId !== conv.id) {
                onSelect(conv.id);
              }
            }}
            data-conversation-id={conv.id}
            data-testid="conversation-item"
          >
            {editingId === conv.id ? (
              <input
                type="text"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                onBlur={() => handleSaveEdit(conv.id)}
                onKeyDown={(e) => handleKeyDown(e, conv.id)}
                onClick={(e) => e.stopPropagation()}
                className="flex-1 bg-app border border-muted rounded px-2 py-1 text-sm text-default focus:outline-none focus:border-accent"
                autoFocus
              />
            ) : (
              <div
                className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap"
                onClick={(e) => {
                  e.stopPropagation();
                  handleStartEdit(conv);
                }}
              >
                {conv.title || 'New Conversation'}
              </div>
            )}
            <button
              className="bg-transparent border-0 text-muted hover:text-error text-2xl cursor-pointer px-2 ml-2 transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(conv.id);
              }}
              data-testid="delete-conversation-btn"
            >
              ×
            </button>
          </div>
        ))}
      </div>
      <div className="p-2.5 px-5 border-t border-default">
        <button
          onClick={onSettings}
          className="btn btn-secondary w-full"
          title="Settings"
          data-testid="settings-btn"
        >
          ⚙️ Settings
        </button>
      </div>
    </div>
  );
}
