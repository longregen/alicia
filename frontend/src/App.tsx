import { useState, useEffect, useCallback, useRef } from 'react';
import { Switch, Route, useRoute, useLocation, Redirect } from 'wouter';
import { Sidebar } from './components/Sidebar';
import ChatWindowBridge from './components/organisms/ChatWindowBridge';
import WelcomeScreen from './components/organisms/WelcomeScreen';
import { Settings, type SettingsTab } from './components/Settings';
import { MemoryManager, MemoryDetail } from './components/organisms/MemoryManager';
import ServerInfoPanel from './components/organisms/ServerPanel/ServerInfoPanel';
import { useConversations } from './hooks/useConversations';
import { useMessages } from './hooks/useMessages';
import { useDatabase } from './hooks/useDatabase';
import { MessageProvider } from './contexts/MessageContext';
import { ConfigProvider } from './contexts/ConfigContext';
import { WebSocketProvider } from './contexts/WebSocketContext';
import { useConversationStore } from './stores/conversationStore';
import Toast from './components/atoms/Toast';
import { Z_INDEX } from './constants/zIndex';

// View types for main content area navigation
export type AppView = 'chat' | 'memory' | 'server' | 'settings';

// Valid settings tabs for validation
const VALID_SETTINGS_TABS: SettingsTab[] = ['mcp', 'optimization', 'preferences'];

function isValidSettingsTab(tab: string): tab is SettingsTab {
  return VALID_SETTINGS_TABS.includes(tab as SettingsTab);
}

// Helper to derive activeView from the current route
function useActiveView(): AppView {
  const [location] = useLocation();
  if (location.startsWith('/memory')) return 'memory';
  if (location.startsWith('/server')) return 'server';
  if (location.startsWith('/settings')) return 'settings';
  return 'chat';
}

// Settings page wrapper that validates tab parameter
function SettingsPage({ conversationId }: { conversationId: string | null }) {
  const [match, params] = useRoute('/settings/:tab');

  if (match && params?.tab) {
    const tab = isValidSettingsTab(params.tab) ? params.tab : 'mcp';
    // Redirect to valid tab if invalid
    if (!isValidSettingsTab(params.tab)) {
      return <Redirect to="/settings/mcp" />;
    }
    return <Settings conversationId={conversationId} defaultTab={tab} />;
  }

  return <Settings conversationId={conversationId} defaultTab="mcp" />;
}

// 404 Page component
function NotFoundPage() {
  const [, navigate] = useLocation();

  return (
    <div className="h-full flex items-center justify-center bg-background">
      <div className="text-center p-8">
        <h1 className="text-6xl font-bold text-muted-foreground mb-4">404</h1>
        <p className="text-xl text-muted-foreground mb-6">Page not found</p>
        <button
          onClick={() => navigate('/')}
          className="btn btn-primary"
        >
          Go Home
        </button>
      </div>
    </div>
  );
}

function AppContent() {
  const [, navigate] = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; variant: 'success' | 'error' | 'warning' } | null>(null);

  // Prevent body scroll when mobile sidebar is open
  useEffect(() => {
    if (sidebarOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [sidebarOpen]);

  // Get conversation ID from route if on chat page
  const [chatMatch, chatParams] = useRoute('/chat/:conversationId');
  const selectedConversationId = chatMatch ? chatParams.conversationId : null;

  const activeView = useActiveView();

  const {
    conversations,
    loading: conversationsLoading,
    hasFetched: conversationsHasFetched,
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
    refetch: refetchMessages,
  } = useMessages(selectedConversationId);

  // Listen for refresh requests from branch switching
  const refreshRequestCounter = useConversationStore((state) => state.refreshRequestCounter);
  const prevRefreshCounter = useRef(refreshRequestCounter);

  useEffect(() => {
    // Only refetch if counter changed (skip initial mount)
    if (prevRefreshCounter.current !== refreshRequestCounter && prevRefreshCounter.current !== 0) {
      refetchMessages();
    }
    prevRefreshCounter.current = refreshRequestCounter;
  }, [refreshRequestCounter, refetchMessages]);

  const handleNewConversation = useCallback(async () => {
    // Generate a default title - backend requires a non-empty title
    const defaultTitle = `New Chat`;
    const newConv = await createConversation(defaultTitle);
    if (newConv) {
      navigate(`/chat/${newConv.id}`);
      setSidebarOpen(false);
    }
  }, [createConversation, navigate]);

  // Handle missing conversations - redirect to home if conversation doesn't exist
  // Only check after the initial fetch has completed to avoid race conditions on page refresh
  useEffect(() => {
    if (selectedConversationId && conversationsHasFetched && !conversationsLoading) {
      if (conversations.length === 0) {
        // No conversations exist - redirect to home
        navigate('/');
      } else {
        const conversationExists = conversations.some(conv => conv.id === selectedConversationId);
        if (!conversationExists) {
          setToast({ message: 'Conversation not found. Starting a new chat.', variant: 'warning' });
          navigate('/');
        }
      }
    }
  }, [selectedConversationId, conversations, conversationsHasFetched, conversationsLoading, navigate]);

  const handleDeleteConversation = async (id: string) => {
    if (id === selectedConversationId) {
      navigate('/');
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

  // Handlers for current conversation (used by ChatWindow menu)
  const handleArchiveCurrentConversation = useCallback(async () => {
    if (selectedConversationId) {
      await updateConversation(selectedConversationId, { status: 'archived' });
      navigate('/');
    }
  }, [selectedConversationId, updateConversation, navigate]);

  const handleDeleteCurrentConversation = useCallback(async () => {
    if (selectedConversationId) {
      await deleteConversation(selectedConversationId);
      navigate('/');
    }
  }, [selectedConversationId, deleteConversation, navigate]);

  const handleSelectConversation = (id: string) => {
    navigate(`/chat/${id}`);
    setSidebarOpen(false);
  };

  const handleOpenSettings = (tab: SettingsTab = 'mcp') => {
    navigate(`/settings/${tab}`);
    setSidebarOpen(false);
  };

  const handlePanelChange = (panel: 'memory' | 'server' | 'settings') => {
    setSidebarOpen(false);
    if (panel === 'memory') {
      navigate('/memory');
    } else if (panel === 'server') {
      navigate('/server');
    } else {
      navigate('/settings');
    }
  };

  return (
    <div className="app flex h-screen bg-app">
      {/* Error banners - dismiss on page reload or when error clears */}
      {conversationsError && (
        <div
          className="fixed top-0 left-0 right-0 bg-error text-white p-3 text-center"
          style={{ zIndex: Z_INDEX.ERROR_BANNER }}
        >
          {conversationsError}
        </div>
      )}
      {messagesError && (
        <div
          className="fixed top-0 left-0 right-0 bg-error text-white p-3 text-center"
          style={{ zIndex: Z_INDEX.ERROR_BANNER }}
        >
          {messagesError}
        </div>
      )}

      {/* Toast notifications */}
      {toast && (
        <div className="fixed top-5 right-5" style={{ zIndex: Z_INDEX.TOAST }}>
          <Toast
            message={toast.message}
            variant={toast.variant}
            duration={3000}
            onDismiss={() => setToast(null)}
            visible={true}
          />
        </div>
      )}

      {/* Mobile hamburger menu - only show when sidebar is closed */}
      {!sidebarOpen && (
        <button
          onClick={() => setSidebarOpen(true)}
          className="fixed top-4 left-4 lg:hidden p-2 bg-surface rounded-md shadow-md hover:bg-elevated transition-colors"
          style={{ zIndex: Z_INDEX.HAMBURGER }}
          aria-label="Open sidebar"
        >
          <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
      )}

      {/* Overlay for mobile/tablet */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 lg:hidden animate-fade-in"
          style={{ zIndex: Z_INDEX.OVERLAY }}
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div
        className={`
          fixed lg:static inset-y-0 left-0
          transform transition-transform duration-300 ease-in-out
          ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
        `}
        style={{ zIndex: Z_INDEX.SIDEBAR }}
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
          onClose={() => setSidebarOpen(false)}
        />
      </div>

      {/* Main content area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <Switch>
          <Route path="/memory/:memoryId">
            {(params) => (
              <div className="h-full bg-background">
                <MemoryDetail memoryId={params.memoryId} />
              </div>
            )}
          </Route>
          <Route path="/memory">
            <div className="h-full bg-background">
              <div className="p-6 md:px-8 border-b border-border bg-card">
                <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Memory</h1>
              </div>
              <div className="flex-1 overflow-y-auto p-4 md:p-8 h-[calc(100%-81px)]">
                <MemoryManager />
              </div>
            </div>
          </Route>
          <Route path="/server">
            <div className="h-full bg-background">
              <div className="p-6 md:px-8 border-b border-border bg-card">
                <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Server Info</h1>
              </div>
              <div className="flex-1 overflow-y-auto p-4 md:p-8 h-[calc(100%-81px)]">
                <ServerInfoPanel />
              </div>
            </div>
          </Route>
          {/* Redirect /settings to /settings/mcp */}
          <Route path="/settings">
            <Redirect to="/settings/mcp" />
          </Route>
          {/* Settings with tab parameter - validated by SettingsPage */}
          <Route path="/settings/:tab">
            <SettingsPage conversationId={selectedConversationId} />
          </Route>
          <Route path="/chat/:conversationId">
            {selectedConversationId ? (
              <ChatWindowBridge
                messages={messages}
                loading={messagesLoading}
                sending={sending}
                onSendMessage={handleSendMessage}
                onArchive={handleArchiveCurrentConversation}
                onDelete={handleDeleteCurrentConversation}
                conversationId={selectedConversationId}
                syncError={syncError}
              />
            ) : (
              <Redirect to="/" />
            )}
          </Route>
          <Route path="/">
            <WelcomeScreen
              conversations={conversations}
              onNewConversation={handleNewConversation}
              onSelectConversation={handleSelectConversation}
              loading={conversationsLoading}
            />
          </Route>
          {/* 404 catch-all route */}
          <Route>
            <NotFoundPage />
          </Route>
        </Switch>
      </div>
    </div>
  );
}

function App() {
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
          <AppContent />
        </MessageProvider>
      </WebSocketProvider>
    </ConfigProvider>
  );
}

export default App;
