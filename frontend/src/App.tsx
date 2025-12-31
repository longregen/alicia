import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import ChatWindowBridge from './components/organisms/ChatWindowBridge';
import { Settings, type SettingsTab } from './components/Settings';
import { useConversations } from './hooks/useConversations';
import { useMessages } from './hooks/useMessages';
import { useDatabase } from './hooks/useDatabase';
import { MessageProvider } from './contexts/MessageContext';
import { ConfigProvider } from './contexts/ConfigContext';
import { storage } from './utils/storage';
import Toast from './components/atoms/Toast';

interface AppContentProps {
  selectedConversationId: string | null;
  setSelectedConversationId: (id: string | null) => void;
  settingsOpen: boolean;
  setSettingsOpen: (open: boolean) => void;
  settingsTab: SettingsTab;
  setSettingsTab: (tab: SettingsTab) => void;
}

function AppContent({
  selectedConversationId,
  setSelectedConversationId,
  settingsOpen,
  setSettingsOpen,
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

  // useMessages depends on MessageContext, so it must be called inside MessageProvider
  const {
    messages,
    loading: messagesLoading,
    error: messagesError,
    sending,
    sendMessage,
    syncError,
  } = useMessages(selectedConversationId);

  // Persist conversation selection
  useEffect(() => {
    storage.setSelectedConversationId(selectedConversationId);
  }, [selectedConversationId]);

  // Handle missing conversations
  useEffect(() => {
    if (selectedConversationId && conversations.length > 0) {
      const conversationExists = conversations.some(conv => conv.id === selectedConversationId);
      if (!conversationExists) {
        setToast({ message: 'Conversation not found. Starting a new chat.', variant: 'warning' });
        setSelectedConversationId(null);
      }
    }
  }, [selectedConversationId, conversations, setSelectedConversationId]);

  const handleNewConversation = async () => {
    // Generate a default title - backend requires a non-empty title
    const defaultTitle = `New Chat`;
    const newConv = await createConversation(defaultTitle);
    if (newConv) {
      setSelectedConversationId(newConv.id);
      setSidebarOpen(false);
    }
  };

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
    setSidebarOpen(false);
  };

  const handleOpenSettings = (tab: SettingsTab = 'mcp') => {
    setSettingsTab(tab);
    setSettingsOpen(true);
    setSidebarOpen(false);
  };

  const handlePanelChange = (panel: 'memory' | 'server' | 'settings') => {
    // Map sidebar panel names to settings tabs
    const tabMap: Record<string, SettingsTab> = {
      memory: 'memories',
      server: 'server',
      settings: 'mcp',
    };
    handleOpenSettings(tabMap[panel] || 'mcp');
  };

  return (
    <div className="app flex h-screen bg-app">
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
        {settingsOpen ? (
          <Settings
            isOpen={settingsOpen}
            onClose={() => setSettingsOpen(false)}
            conversationId={selectedConversationId}
            defaultTab={settingsTab}
          />
        ) : (
          <ChatWindowBridge
            messages={messages}
            loading={messagesLoading}
            sending={sending}
            onSendMessage={handleSendMessage}
            conversationId={selectedConversationId}
            syncError={syncError}
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
  const [settingsOpen, setSettingsOpen] = useState(false);
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
      <MessageProvider>
        <AppContent
          selectedConversationId={selectedConversationId}
          setSelectedConversationId={setSelectedConversationId}
          settingsOpen={settingsOpen}
          setSettingsOpen={setSettingsOpen}
          settingsTab={settingsTab}
          setSettingsTab={setSettingsTab}
        />
      </MessageProvider>
    </ConfigProvider>
  );
}

export default App;
