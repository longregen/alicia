import { useState, useEffect, useCallback } from 'react';
import { Switch, Route, useRoute, useLocation, Redirect } from 'wouter';
import { Sidebar } from './components/Sidebar';
import ChatWindow from './components/organisms/ChatWindow';
import WelcomeScreen from './components/organisms/WelcomeScreen';
import { Settings, type SettingsTab } from './components/Settings';
import { MemoryManager, MemoryDetail } from './components/organisms/MemoryManager';
import { NotesPage } from './components/NotesPage';
import { useConversations } from './hooks/useConversations';
import { useChat } from './hooks/useChat';
import { useTheme } from './hooks/useTheme';
import { ConfigProvider } from './contexts/ConfigContext';
import { WebSocketProvider, useWebSocket } from './contexts/WebSocketContext';
import { createMessageId, createConversationId, createEmptyMessage, type MessageId } from './types/chat';
import { MessageType } from './types/protocol';
import Toast from './components/atoms/Toast';
import { Z_INDEX } from './constants/zIndex';
import type { AppView } from './types/app';
import { useSidebarStore } from './stores/sidebarStore';
import { useChatStore } from './stores/chatStore';
import { nanoid } from 'nanoid';

const VALID_SETTINGS_TABS: SettingsTab[] = ['mcp', 'preferences'];

function isValidSettingsTab(tab: string): tab is SettingsTab {
  return VALID_SETTINGS_TABS.includes(tab as SettingsTab);
}

function useActiveView(): AppView {
  const [location] = useLocation();
  if (location.startsWith('/notes')) return 'notes';
  if (location.startsWith('/memory')) return 'memory';
  if (location.startsWith('/settings')) return 'settings';
  return 'chat';
}

function SettingsPage() {
  const [match, params] = useRoute('/settings/:tab');

  if (match && params?.tab) {
    if (!isValidSettingsTab(params.tab)) {
      return <Redirect to="/settings/mcp" />;
    }
    return <Settings defaultTab={params.tab} />;
  }

  return <Settings defaultTab="mcp" />;
}

function NotFoundPage() {
  const [, navigate] = useLocation();

  return (
    <div className="h-full flex items-center justify-center bg-background">
      <div className="text-center p-8">
        <h1 className="text-6xl font-bold text-muted-foreground mb-4">404</h1>
        <p className="text-xl text-muted-foreground mb-6">Page not found</p>
        <button onClick={() => navigate('/')} className="btn btn-primary">
          Go Home
        </button>
      </div>
    </div>
  );
}

function AppContent() {
  const [, navigate] = useLocation();
  const sidebarOpen = useSidebarStore((state) => state.isOpen);
  const setSidebarOpen = useSidebarStore((state) => state.setOpen);
  useTheme();
  const [toast, setToast] = useState<{
    message: string;
    variant: 'success' | 'error' | 'warning';
  } | null>(null);

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

  const {
    error: messagesError,
    sendMessage,
    switchBranch,
  } = useChat(selectedConversationId);

  const { send: wsSend } = useWebSocket();

  const handleNewConversation = useCallback(async () => {
    const defaultTitle = `New Chat`;
    const newConv = await createConversation(defaultTitle);
    if (newConv) {
      navigate(`/chat/${newConv.id}`);
      setSidebarOpen(false);
    }
  }, [createConversation, navigate, setSidebarOpen]);

  useEffect(() => {
    if (selectedConversationId && conversationsHasFetched && !conversationsLoading) {
      if (conversations.length === 0) {
        navigate('/');
      } else {
        const conversationExists = conversations.some((conv) => conv.id === selectedConversationId);
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

  const handleSendMessage = async (content: string, _isVoice: boolean) => {
    await sendMessage(content);
  };

  const getMessage = useChatStore((s) => s.getMessage);
  const initiateRegeneration = useChatStore((s) => s.initiateRegeneration);

  const handleRetry = useCallback((messageId: MessageId) => {
    if (!selectedConversationId) return;

    const convId = createConversationId(selectedConversationId);
    const originalMessage = getMessage(convId, messageId);
    if (!originalMessage) {
      console.warn('[handleRetry] Message not found:', messageId);
      return;
    }

    const previousId = originalMessage.previous_id;

    // Will be replaced by server's StartAnswer
    const optimisticId = createMessageId(nanoid());
    const optimisticMessage = createEmptyMessage(optimisticId, convId, 'assistant');
    optimisticMessage.previous_id = previousId;

    initiateRegeneration(convId, optimisticMessage);

    wsSend({
      conversationId: selectedConversationId,
      type: MessageType.GenerationRequest,
      body: {
        conversationId: selectedConversationId,
        messageId: messageId as string,
        requestType: 'regenerate',
        enableTools: true,
        enableReasoning: false,
        enableStreaming: true,
      },
    });
  }, [selectedConversationId, wsSend, getMessage, initiateRegeneration]);

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

  const handlePanelChange = (panel: 'memory' | 'settings' | 'notes') => {
    setSidebarOpen(false);
    if (panel === 'memory') {
      navigate('/memory');
    } else if (panel === 'notes') {
      navigate('/notes');
    } else {
      navigate('/settings');
    }
  };

  return (
    <div className="app flex h-screen bg-app">
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

      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 lg:hidden animate-fade-in"
          style={{ zIndex: Z_INDEX.OVERLAY }}
          onClick={() => setSidebarOpen(false)}
        />
      )}

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

      <div className="flex-1 flex flex-col overflow-hidden">
        <Switch>
          <Route path="/notes">
            <NotesPage />
          </Route>
          <Route path="/memory/:memoryId">
            {(params) => (
              <div className="h-full bg-background">
                <MemoryDetail memoryId={params.memoryId} />
              </div>
            )}
          </Route>
          <Route path="/memory">
            <div className="h-full bg-background">
              <div className="p-6 md:px-8 border-b border-border bg-card flex items-center gap-3">
                <button
                  onClick={() => setSidebarOpen(true)}
                  className="lg:hidden p-2 -ml-2 hover:bg-elevated rounded-md transition-colors"
                  aria-label="Open sidebar"
                >
                  <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
                <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Memory</h1>
              </div>
              <div className="flex-1 overflow-y-auto p-4 md:p-8 h-[calc(100%-81px)]">
                <MemoryManager />
              </div>
            </div>
          </Route>
          <Route path="/settings">
            <Redirect to="/settings/mcp" />
          </Route>
          <Route path="/settings/:tab">
            <SettingsPage />
          </Route>
          <Route path="/chat/:conversationId">
            {selectedConversationId ? (
              <ChatWindow
                conversationId={selectedConversationId}
                conversationTitle="Conversation"
                onSendMessage={handleSendMessage}
                onBranchSwitch={(targetId) => switchBranch(createMessageId(targetId))}
                onRetry={handleRetry}
                onArchive={handleArchiveCurrentConversation}
                onDelete={handleDeleteCurrentConversation}
                showControls={true}
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
          <Route>
            <NotFoundPage />
          </Route>
        </Switch>
      </div>
    </div>
  );
}

function App() {
  return (
    <ConfigProvider>
      <WebSocketProvider>
        <AppContent />
      </WebSocketProvider>
    </ConfigProvider>
  );
}

export default App;
