import { Conversation } from '../types/models';

interface SidebarProps {
  conversations: Conversation[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onNew: () => void;
  onDelete: (id: string) => void;
  onSettings: () => void;
  loading: boolean;
}

export function Sidebar({
  conversations,
  selectedId,
  onSelect,
  onNew,
  onDelete,
  onSettings,
  loading,
}: SidebarProps) {
  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h2>Alicia</h2>
        <button onClick={onNew} className="new-chat-btn" data-testid="new-chat-btn">
          New Chat
        </button>
      </div>
      <div className="conversations-list">
        {loading && <div className="loading">Loading...</div>}
        {!loading && conversations.length === 0 && (
          <div className="empty-state">No conversations yet</div>
        )}
        {conversations.map(conv => (
          <div
            key={conv.id}
            className={`conversation-item ${selectedId === conv.id ? 'selected' : ''}`}
            onClick={() => onSelect(conv.id)}
            data-conversation-id={conv.id}
            data-testid="conversation-item"
          >
            <div className="conversation-title">
              {conv.title || 'New Conversation'}
            </div>
            <button
              className="delete-btn"
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
      <div className="sidebar-footer">
        <button onClick={onSettings} className="settings-btn" title="Settings" data-testid="settings-btn">
          ⚙️ Settings
        </button>
      </div>
    </div>
  );
}
