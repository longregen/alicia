import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import ChatWindowBridge from './components/organisms/ChatWindowBridge';
import { Settings } from './components/Settings';
import { useConversations } from './hooks/useConversations';
import { useMessages } from './hooks/useMessages';
import { useDatabase } from './hooks/useDatabase';
import { MessageProvider } from './contexts/MessageContext';
import { ConfigProvider } from './contexts/ConfigContext';
import { storage } from './utils/storage';

interface AppContentProps {
  selectedConversationId: string | null;
  setSelectedConversationId: (id: string | null) => void;
  settingsOpen: boolean;
  setSettingsOpen: (open: boolean) => void;
}

function AppContent({
  selectedConversationId,
  setSelectedConversationId,
  settingsOpen,
  setSettingsOpen,
}: AppContentProps) {
  const {
    conversations,
    loading: conversationsLoading,
    error: conversationsError,
    createConversation,
    deleteConversation,
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

  const handleNewConversation = async () => {
    // Generate a default title - backend requires a non-empty title
    const defaultTitle = `New Chat`;
    const newConv = await createConversation(defaultTitle);
    if (newConv) {
      setSelectedConversationId(newConv.id);
    }
  };

  const handleDeleteConversation = async (id: string) => {
    if (id === selectedConversationId) {
      setSelectedConversationId(null);
    }
    await deleteConversation(id);
  };

  const handleSendMessage = async (content: string) => {
    await sendMessage(content);
  };

  return (
    <div className="app">
      {conversationsError && (
        <div className="error-banner">{conversationsError}</div>
      )}
      {messagesError && (
        <div className="error-banner">{messagesError}</div>
      )}

      <Sidebar
        conversations={conversations}
        selectedId={selectedConversationId}
        onSelect={setSelectedConversationId}
        onNew={handleNewConversation}
        onDelete={handleDeleteConversation}
        onSettings={() => setSettingsOpen(true)}
        loading={conversationsLoading}
      />

      <ChatWindowBridge
        messages={messages}
        loading={messagesLoading}
        sending={sending}
        onSendMessage={handleSendMessage}
        conversationId={selectedConversationId}
        syncError={syncError}
      />

      <Settings
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        conversationId={selectedConversationId}
      />
    </div>
  );
}

function App() {
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(() => {
    return storage.getSelectedConversationId();
  });
  const [settingsOpen, setSettingsOpen] = useState(false);

  // Initialize database
  const { isReady, error: dbError } = useDatabase();

  if (dbError) {
    return (
      <div className="app-error">
        <h1>Database Error</h1>
        <p>{dbError.message}</p>
        <p>Please refresh the page to try again.</p>
      </div>
    );
  }

  if (!isReady) {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>Initializing database...</p>
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
        />
      </MessageProvider>
    </ConfigProvider>
  );
}

export default App;
