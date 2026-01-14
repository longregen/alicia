import { useState, useEffect, useCallback } from 'react';
import { Sidebar } from './components/Sidebar';
import ChatWindowBridge from './components/organisms/ChatWindowBridge';
import WelcomeScreen from './components/organisms/WelcomeScreen';
import { Settings, type SettingsTab } from './components/Settings';
import { MemoryManager } from './components/organisms/MemoryManager';
import ServerInfoPanel from './components/organisms/ServerPanel/ServerInfoPanel';
import { useConversations } from './hooks/useConversations';
import { useMessages } from './hooks/useMessages';
import { useDatabase } from './hooks/useDatabase';
import { MessageProvider } from './contexts/MessageContext';
import { ConfigProvider } from './contexts/ConfigContext';
import { WebSocketProvider } from './contexts/WebSocketContext';
import { storage } from './utils/storage';
import Toast from './components/atoms/Toast';

// View types for main content area navigation
export type AppView = 'chat' | 'memory' | 'server' | 'settings';

interface AppContentProps {
  selectedConversationId: string | null;
  setSelectedConversationId: (id: string | null) => void;
  activeView: AppView;
  setActiveView: (view: AppView) => void;
  settingsTab: SettingsTab;
  setSettingsTab: (tab: SettingsTab) => void;
}

function AppContent({
  selectedConversationId,
  setSelectedConversationId,
  activeView,
  setActiveView,
  settingsTab,
  setSettingsTab,
}: AppContentProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'success' | 'error' | 'warning' } | null>(null);

  const {
    conversations,
    loading: conversationsLoading,
    error: conversationsError,
    createConversation,
    deleteConversation,
    updateConversation,
  } = useConversations();

  // Note: useMessages requires MessageContext - this component must be inside MessageProvider
  const {
    messages,
    loading: messagesLoading,
    error: messagesError,
    sending,
    sendMessage,
    syncError,
  } = useMessages(selectedConversationId);

  const handleNewConversation = useCallback(async () => {
    // Generate a default title - backend requires a non-empty title
    const defaultTitle = `New Chat`;
    const newConv = await createConversation(defaultTitle);
    if (newConv) {
      setSelectedConversationId(newConv.id);
      setSidebarOpen(false);
    }
  }, [createConversation, setSelectedConversationId]);

  // Persist conversation selection
  useEffect(() => {
    storage.setSelectedConversationId(selectedConversationId);
  }, [selectedConversationId]);

  // Handle missing conversations - clear stale IDs from localStorage
  useEffect(() => {
    if (selectedConversationId && !conversationsLoading) {
      if (conversations.length === 0) {
        // No conversations exist - clear stale ID
        setSelectedConversationId(null);
      } else {
        const conversationExists = conversations.some(conv => conv.id === selectedConversationId);
        if (!conversationExists) {
          setToast({ message: 'Conversation not found. Starting a new chat.', variant: 'warning' });
          setSelectedConversationId(null);
        }
      }
    }
  }, [selectedConversationId, conversations, conversationsLoading, setSelectedConversationId]);

  const handleDeleteConversation = async (id: string) => {
    if (id === selectedConversationId) {
      setSelectedConversationId(null);
    }
    await deleteConversation(id);
  };

  const handleRenameConversation = async (id: string, newTitle: string) => {
    await updateConversation(id, { title: newTitle });
  };

  const handleArchiveConversation = async (id: string) => {
    await updateConversation(id, { status: 'archived' });
  };

  const handleUnarchiveConversation = async (id: string) => {
    await updateConversation(id, { status: 'active' });
  };

  const handleSendMessage = async (content: string) => {
    await sendMessage(content);
  };

  const handleSelectConversation = (id: string) => {
    setSelectedConversationId(id);
    setActiveView('chat');
    setSidebarOpen(false);
  };

  const handleOpenSettings = (tab: SettingsTab = 'mcp') => {
    setSettingsTab(tab);
    setActiveView('settings');
    setSidebarOpen(false);
  };

  const handlePanelChange = (panel: 'memory' | 'server' | 'settings') => {
    setSidebarOpen(false);
    if (panel === 'memory') {
      setActiveView('memory');
    } else if (panel === 'server') {
      setActiveView('server');
    } else {
      setActiveView('settings');
    }
  };

  return (
    <div className="app flex h-screen bg-app">
      {/* Error banners - dismiss on page reload or when error clears */}
      {conversationsError && (
        <div className="fixed top-0 left-0 right-0 bg-error text-white p-3 text-center z-[1000]">
          {conversationsError}
        </div>
      )}
      {messagesError && (
        <div className="fixed top-0 left-0 right-0 bg-error text-white p-3 text-center z-[1000]">
          {messagesError}
        </div>
      )}

      {/* Toast notifications */}
      {toast && (
        <div className="fixed top-5 right-5 z-[1000]">
          <Toast
            message={toast.message}
            variant={toast.variant}
            duration={3000}
            onDismiss={() => setToast(null)}
            visible={true}
          />
        </div>
      )}

      {/* Mobile hamburger menu */}
      <button
        onClick={() => setSidebarOpen(!sidebarOpen)}
        className="fixed top-4 left-4 z-[60] lg:hidden p-2 bg-surface rounded-md shadow-md hover:bg-elevated transition-colors"
        aria-label="Toggle sidebar"
      >
        <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          {sidebarOpen ? (
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          ) : (
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          )}
        </svg>
      </button>

      {/* Overlay for mobile/tablet */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-[40] lg:hidden animate-fade-in"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div
        className={`
          fixed lg:static inset-y-0 left-0 z-[50]
          transform transition-transform duration-300 ease-in-out
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
        `}
      >
        <Sidebar
          conversations={conversations}
          selectedId={selectedConversationId}
          activeView={activeView}
          onSelect={handleSelectConversation}
          onNew={handleNewConversation}
          onDelete={handleDeleteConversation}
          onRenameConversation={handleRenameConversation}
          onArchive={handleArchiveConversation}
          onUnarchive={handleUnarchiveConversation}
          onSettings={handleOpenSettings}
          onPanelChange={handlePanelChange}
          loading={conversationsLoading}
        />
      </div>

      {/* Main content area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {activeView === 'memory' ? (
          <div className="h-full bg-background">
            <div className="p-6 md:px-8 border-b border-border bg-card">
              <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Memory</h1>
            </div>
            <div className="flex-1 overflow-y-auto p-4 md:p-8 h-[calc(100%-81px)]">
              <MemoryManager />
            </div>
          </div>
        ) : activeView === 'server' ? (
          <div className="h-full bg-background">
            <div className="p-6 md:px-8 border-b border-border bg-card">
              <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Server Info</h1>
            </div>
            <div className="flex-1 overflow-y-auto p-4 md:p-8 h-[calc(100%-81px)]">
              <ServerInfoPanel />
            </div>
          </div>
        ) : activeView === 'settings' ? (
          <Settings
            conversationId={selectedConversationId}
            defaultTab={settingsTab}
          />
        ) : selectedConversationId ? (
          <ChatWindowBridge
            messages={messages}
            loading={messagesLoading}
            sending={sending}
            onSendMessage={handleSendMessage}
            conversationId={selectedConversationId}
            syncError={syncError}
          />
        ) : (
          <WelcomeScreen
            conversations={conversations}
            onNewConversation={handleNewConversation}
            onSelectConversation={handleSelectConversation}
            loading={conversationsLoading}
          />
        )}
      </div>
    </div>
  );
}

function App() {
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(() => {
    return storage.getSelectedConversationId();
  });
  const [activeView, setActiveView] = useState<AppView>('chat');
  const [settingsTab, setSettingsTab] = useState<SettingsTab>('mcp');

  // Initialize database
  const { isReady, error: dbError } = useDatabase();

  if (dbError) {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-app text-default p-8">
        <h1 className="text-2xl font-bold mb-4">Database Error</h1>
        <p className="text-error mb-2">{dbError.message}</p>
        <p className="text-muted">Please refresh the page to try again.</p>
      </div>
    );
  }

  if (!isReady) {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-app text-default p-8">
        <div className="w-12 h-12 border-4 border-accent border-t-transparent rounded-full animate-spin mb-4"></div>
        <p className="text-muted">Initializing database...</p>
      </div>
    );
  }

  return (
    <ConfigProvider>
      <WebSocketProvider>
        <MessageProvider>
          <AppContent
            selectedConversationId={selectedConversationId}
            setSelectedConversationId={setSelectedConversationId}
            activeView={activeView}
            setActiveView={setActiveView}
            settingsTab={settingsTab}
            setSettingsTab={setSettingsTab}
          />
        </MessageProvider>
      </WebSocketProvider>
    </ConfigProvider>
  );
}

export default App;
