import { useState, useEffect, useRef } from 'react';
import { Brain, Server, Settings, ChevronLeft, ChevronRight, ChevronDown, ChevronUp, Edit2, Archive, ArchiveRestore, Trash2 } from 'lucide-react';
import { Conversation } from '../types/models';
import { useSidebarStore, COLLAPSED_WIDTH } from '../stores/sidebarStore';
import { formatRelativeTime } from '../lib/timeUtils';
import { cls } from '../utils/cls';
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from './atoms/Command';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from './atoms/DropdownMenu';
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from './atoms/Collapsible';
import { ConnectionStatusIndicator } from './ConnectionStatusIndicator';

interface SidebarProps {
  conversations: Conversation[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onNew: () => void;
  onDelete: (id: string) => void;
  onRenameConversation: (id: string, newTitle: string) => void;
  onArchive: (id: string) => void;
  onUnarchive: (id: string) => void;
  onSettings: () => void;
  onPanelChange: (panel: 'memory' | 'server' | 'settings') => void;
  loading: boolean;
}

export function Sidebar({
  conversations,
  selectedId,
  onSelect,
  onNew,
  onDelete,
  onRenameConversation,
  onArchive,
  onUnarchive,
  onSettings,
  onPanelChange,
  loading,
}: SidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [isResizing, setIsResizing] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [archivedOpen, setArchivedOpen] = useState(false);
  const sidebarRef = useRef<HTMLDivElement>(null);

  const { isCollapsed, width, toggleCollapsed, setWidth } = useSidebarStore();

  // Separate conversations by status
  const activeConversations = conversations.filter(c => c.status === 'active');
  const archivedConversations = conversations.filter(c => c.status === 'archived');

  const handleStartEdit = (conv: Conversation) => {
    if (isCollapsed) return; // Don't allow editing when collapsed
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

  // Keyboard shortcuts: Cmd/Ctrl+B to toggle sidebar, Cmd/Ctrl+K to search
  useEffect(() => {
    const handleKeyboardShortcut = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
        e.preventDefault();
        toggleCollapsed();
      } else if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setSearchOpen(true);
      }
    };

    window.addEventListener('keydown', handleKeyboardShortcut);
    return () => window.removeEventListener('keydown', handleKeyboardShortcut);
  }, [toggleCollapsed]);

  // Handle resize drag
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing || isCollapsed) return;

      const newWidth = e.clientX;
      setWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);

      return () => {
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };
    }
  }, [isResizing, isCollapsed, setWidth]);

  const actualWidth = isCollapsed ? COLLAPSED_WIDTH : width;

  const [openMenuId, setOpenMenuId] = useState<string | null>(null);

  const renderConversationItem = (conv: Conversation) => {
    const isArchived = conv.status === 'archived';

    return (
      <DropdownMenu key={conv.id} open={openMenuId === conv.id} onOpenChange={(open) => setOpenMenuId(open ? conv.id : null)}>
        <div
          className={cls(
            'conversation-item p-3 mb-2 rounded-md cursor-pointer transition-colors',
            'hover:bg-sidebar-accent',
            selectedId === conv.id && 'bg-sidebar-accent border-l-2 border-accent',
            isCollapsed && 'p-2 flex justify-center'
          )}
          onClick={() => {
            if (editingId !== conv.id) {
              onSelect(conv.id);
            }
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            setOpenMenuId(conv.id);
          }}
          data-conversation-id={conv.id}
          data-testid="conversation-item"
          title={isCollapsed ? conv.title || 'New Conversation' : undefined}
        >
            {isCollapsed ? (
              // Collapsed view: just show first letter
              <div className="w-8 h-8 rounded-full bg-primary/20 text-primary flex items-center justify-center font-semibold">
                {(conv.title || 'N')[0].toUpperCase()}
              </div>
            ) : (
              // Expanded view
              <div className="flex flex-col gap-1">
                <div className="flex justify-between items-start gap-2">
                  {editingId === conv.id ? (
                    <input
                      type="text"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onBlur={() => handleSaveEdit(conv.id)}
                      onKeyDown={(e) => handleKeyDown(e, conv.id)}
                      onClick={(e) => e.stopPropagation()}
                      className="flex-1 bg-input border border-border rounded px-2 py-1 text-sm text-foreground focus:outline-none focus:border-accent"
                      autoFocus
                    />
                  ) : (
                    <div
                      className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-medium"
                    >
                      {conv.title || 'New Conversation'}
                    </div>
                  )}
                </div>
                <div className="text-xs text-muted-foreground">
                  {formatRelativeTime(conv.updated_at)}
                </div>
              </div>
            )}
          </div>
        <DropdownMenuContent align="start" className="w-48">
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              handleStartEdit(conv);
            }}
          >
            <Edit2 className="w-4 h-4" />
            Rename
          </DropdownMenuItem>
          {isArchived ? (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                onUnarchive(conv.id);
              }}
            >
              <ArchiveRestore className="w-4 h-4" />
              Unarchive
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation();
                onArchive(conv.id);
              }}
            >
              <Archive className="w-4 h-4" />
              Archive
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(conv.id);
            }}
            data-testid="delete-conversation-menu-item"
          >
            <Trash2 className="w-4 h-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  };

  return (
    <>
      {/* Search Dialog */}
      <CommandDialog
        open={searchOpen}
        onOpenChange={setSearchOpen}
        title="Search Conversations"
        description="Search for a conversation by title"
      >
        <CommandInput placeholder="Search conversations..." />
        <CommandList>
          <CommandEmpty>No conversations found.</CommandEmpty>
          <CommandGroup heading="Conversations">
            {conversations.map((conv) => (
              <CommandItem
                key={conv.id}
                onSelect={() => {
                  onSelect(conv.id);
                  setSearchOpen(false);
                }}
                value={conv.title || 'New Conversation'}
              >
                <div className="flex flex-col gap-1 w-full">
                  <div className="font-medium">{conv.title || 'New Conversation'}</div>
                  <div className="text-xs text-muted-foreground">
                    {formatRelativeTime(conv.updated_at)}
                  </div>
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        </CommandList>
      </CommandDialog>

      <div
        ref={sidebarRef}
        className={cls(
          'bg-sidebar text-foreground flex flex-col border-r border-border h-full relative transition-all',
          isResizing && 'select-none'
        )}
        style={{ width: `${actualWidth}px` }}
      >
      {/* Header with toggle button */}
      <div className={cls('p-5 border-b border-border', isCollapsed && 'p-3')}>
        {!isCollapsed ? (
          <>
            <div className="layout-between mb-3">
              <h2 className="text-2xl font-semibold">Alicia</h2>
              <button
                onClick={toggleCollapsed}
                className="p-1 hover:bg-sidebar-accent rounded transition-colors"
                title="Collapse sidebar (⌘B)"
                aria-label="Collapse sidebar"
              >
                <ChevronLeft className="w-4 h-4" />
              </button>
            </div>
            <button
              onClick={onNew}
              className="btn btn-secondary w-full"
              data-testid="new-chat-btn"
            >
              New Chat
            </button>
          </>
        ) : (
          <button
            onClick={toggleCollapsed}
            className="w-full p-2 hover:bg-sidebar-accent rounded transition-colors"
            title="Expand sidebar (⌘B)"
            aria-label="Expand sidebar"
          >
            <ChevronRight className="w-5 h-5 mx-auto" />
          </button>
        )}
      </div>

      {/* Conversation list */}
      <div className="flex-1 overflow-y-auto p-2.5">
        {loading && !isCollapsed && (
          <div className="text-center text-muted-foreground p-5">Loading...</div>
        )}
        {!loading && conversations.length === 0 && !isCollapsed && (
          <div className="text-center text-muted-foreground p-5">No conversations yet</div>
        )}

        {/* Active conversations */}
        {!isCollapsed && activeConversations.length > 0 && (
          <div className="mb-4">
            <div className="text-xs font-semibold text-muted-foreground px-2 mb-2 uppercase tracking-wide">
              Active ({activeConversations.length})
            </div>
            {activeConversations.map(conv => renderConversationItem(conv))}
          </div>
        )}

        {/* Archived conversations (collapsible) */}
        {!isCollapsed && archivedConversations.length > 0 && (
          <div className="mb-4">
            <Collapsible open={archivedOpen} onOpenChange={setArchivedOpen}>
              <CollapsibleTrigger className="w-full layout-between text-xs font-semibold text-muted-foreground px-2 mb-2 uppercase tracking-wide hover:text-foreground transition-colors">
                <span>Archived ({archivedConversations.length})</span>
                {archivedOpen ? (
                  <ChevronUp className="w-3 h-3" />
                ) : (
                  <ChevronDown className="w-3 h-3" />
                )}
              </CollapsibleTrigger>
              <CollapsibleContent>
                {archivedConversations.map(conv => renderConversationItem(conv))}
              </CollapsibleContent>
            </Collapsible>
          </div>
        )}

        {/* Collapsed view shows all conversations */}
        {isCollapsed && conversations.map(conv => renderConversationItem(conv))}
      </div>

      {/* Bottom navigation */}
      <div className={cls('border-t border-border', isCollapsed ? 'p-2' : 'p-2.5')}>
        {/* Connection status indicator */}
        <ConnectionStatusIndicator isCollapsed={isCollapsed} />

        {isCollapsed ? (
          // Collapsed: Icon-only buttons
          <div className="flex flex-col gap-1 mt-2">
            <button
              onClick={() => onPanelChange('memory')}
              className="p-2 hover:bg-sidebar-accent rounded transition-colors"
              title="Memory"
              aria-label="Memory"
            >
              <Brain className="w-5 h-5 mx-auto" />
            </button>
            <button
              onClick={() => onPanelChange('server')}
              className="p-2 hover:bg-sidebar-accent rounded transition-colors"
              title="Server"
              aria-label="Server"
            >
              <Server className="w-5 h-5 mx-auto" />
            </button>
            <button
              onClick={onSettings}
              className="p-2 hover:bg-sidebar-accent rounded transition-colors"
              title="Settings"
              data-testid="settings-btn"
              aria-label="Settings"
            >
              <Settings className="w-5 h-5 mx-auto" />
            </button>
          </div>
        ) : (
          // Expanded: Full buttons
          <div className="flex flex-col gap-2 mt-2">
            <button
              onClick={() => onPanelChange('memory')}
              className="layout-center-gap p-2 hover:bg-sidebar-accent rounded transition-colors w-full text-left"
              title="Memory"
            >
              <Brain className="w-4 h-4" />
              <span>Memory</span>
            </button>
            <button
              onClick={() => onPanelChange('server')}
              className="layout-center-gap p-2 hover:bg-sidebar-accent rounded transition-colors w-full text-left"
              title="Server"
            >
              <Server className="w-4 h-4" />
              <span>Server</span>
            </button>
            <button
              onClick={onSettings}
              className="layout-center-gap p-2 hover:bg-sidebar-accent rounded transition-colors w-full text-left"
              title="Settings"
              data-testid="settings-btn"
            >
              <Settings className="w-4 h-4" />
              <span>Settings</span>
            </button>
          </div>
        )}
      </div>

      {/* Resize handle */}
      {!isCollapsed && (
        <div
          className={cls(
            'absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-accent/50 transition-colors',
            isResizing && 'bg-accent'
          )}
          onMouseDown={() => setIsResizing(true)}
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize sidebar"
        />
      )}
    </div>
    </>
  );
}
