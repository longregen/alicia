import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import { ChatWindow } from './components/ChatWindow';
import { Settings } from './components/Settings';
import { useConversations } from './hooks/useConversations';
import { useMessages } from './hooks/useMessages';
import { MessageProvider } from './contexts/MessageContext';
import { Conversation } from './types/models';
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
    updateConversation,
  } = useConversations();

  // Get the current conversation object
  const currentConversation = conversations.find(c => c.id === selectedConversationId) || null;

  // useMessages depends on MessageContext, so it must be called inside MessageProvider
  const {
    messages,
    loading: messagesLoading,
    error: messagesError,
    sending,
    sendMessage,
    // Sync state
    isSyncing,
    lastSyncTime,
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

  const handleConversationUpdate = (updatedConversation: Conversation) => {
    // Update the conversation in the local state
    updateConversation(updatedConversation);
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

      <ChatWindow
        messages={messages}
        loading={messagesLoading}
        sending={sending}
        onSendMessage={handleSendMessage}
        conversationId={selectedConversationId}
        conversation={currentConversation}
        onConversationUpdate={handleConversationUpdate}
        isSyncing={isSyncing}
        lastSyncTime={lastSyncTime}
        syncError={syncError}
      />

      <Settings
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />
    </div>
  );
}

function App() {
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(() => {
    return storage.getSelectedConversationId();
  });
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <MessageProvider>
      <AppContent
        selectedConversationId={selectedConversationId}
        setSelectedConversationId={setSelectedConversationId}
        settingsOpen={settingsOpen}
        setSettingsOpen={setSettingsOpen}
      />
    </MessageProvider>
  );
}

export default App;
