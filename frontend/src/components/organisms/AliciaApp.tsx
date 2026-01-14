import { useState, useEffect } from 'react';
import { Sidebar } from '../Sidebar';
import ChatWindow from './ChatWindow';
import { MemoryManager } from './MemoryManager';
import ServerInfoPanel from './ServerPanel/ServerInfoPanel';
import { Settings } from '../Settings';
import { useConversations } from '../../hooks/useConversations';
import { useConversationStore } from '../../stores/conversationStore';
import { createConversationId } from '../../types/streaming';
import type { VoiceState } from '../atoms/VoiceVisualizer';
import { sendControlStop, sendControlVariation } from '../../adapters/protocolAdapter';
import type { AppView } from '../../App';

/**
 * Panel types for the main navigation
 * @deprecated Use AppView from App.tsx instead
 */
export type Panel = AppView;

/**
 * AliciaApp - Main application layout component
 *
 * Orchestrates the overall application structure:
 * - Resizable sidebar with conversation list
 * - Panel navigation (chat, memory, server, settings)
 * - Main content area that renders the active panel
 * - Connection status management
 * - Voice state handling
 *
 * This is the top-level organism that composes the entire UI.
 */
export function AliciaApp() {
  // UI state
  // Reserved for future sidebar state management
  const [_sidebarOpen, _setSidebarOpen] = useState(true);
  const [activePanel, setActivePanel] = useState<Panel>('chat');

  // Conversation management
  const {
    conversations,
    loading: conversationsLoading,
    error: conversationsError,
    createConversation,
    deleteConversation,
    updateConversation,
  } = useConversations();
  const [activeConversationId, setActiveConversationId] = useState<string | null>(null);

  // Voice state
  // Reserved for future voice state integration
  const [_voiceState, _setVoiceState] = useState<VoiceState>('idle');

  // Store actions for clearing messages on conversation switch
  const clearConversation = useConversationStore((state) => state.clearConversation);
  const setCurrentConversationId = useConversationStore((state) => state.setCurrentConversationId);

  // Clear store and set current conversation when switching conversations
  useEffect(() => {
    if (activeConversationId) {
      clearConversation();
      setCurrentConversationId(createConversationId(activeConversationId));
    } else {
      clearConversation();
      setCurrentConversationId(null);
    }
  }, [activeConversationId, clearConversation, setCurrentConversationId]);

  // Handlers
  const handleNewConversation = async () => {
    const newConversation = await createConversation();
    if (newConversation) {
      setActiveConversationId(newConversation.id);
      setActivePanel('chat');
    }
  };

  const handleSelectConversation = (id: string) => {
    setActiveConversationId(id);
    setActivePanel('chat'); // Switch to chat when selecting conversation
  };

  const handleDeleteConversation = async (id: string) => {
    await deleteConversation(id);
    // If we deleted the active conversation, clear the selection
    if (activeConversationId === id) {
      setActiveConversationId(null);
    }
  };

  const handleRenameConversation = async (id: string, newTitle: string) => {
    await updateConversation(id, { title: newTitle });
  };

  const handleArchiveConversation = async (id: string) => {
    await updateConversation(id, { status: 'archived' });
    // If we archived the active conversation, clear the selection
    if (activeConversationId === id) {
      setActiveConversationId(null);
    }
  };

  const handleUnarchiveConversation = async (id: string) => {
    await updateConversation(id, { status: 'active' });
  };

  const handleSendMessage = (message: string, isVoice: boolean) => {
    // Message sending is delegated to ChatWindow component
    // This handler is for future cross-component message coordination
    console.log('Send message:', message, 'isVoice:', isVoice);
  };

  const handleStopStreaming = () => {
    if (!activeConversationId) {
      console.warn('Cannot stop streaming: no active conversation');
      return;
    }

    sendControlStop(activeConversationId, 'all');
  };

  const handleRegenerateResponse = () => {
    if (!activeConversationId) {
      console.warn('Cannot regenerate: no active conversation');
      return;
    }

    // Get the conversation store to find the last assistant message
    const store = useConversationStore.getState();
    const messages = Object.values(store.messages)
      .filter(msg => msg.conversationId === activeConversationId && msg.role === 'assistant')
      .sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());

    if (messages.length === 0) {
      console.warn('Cannot regenerate: no assistant messages found');
      return;
    }

    // Get the last assistant message
    const lastAssistantMessage = messages[messages.length - 1];

    // Send regenerate request
    sendControlVariation(activeConversationId, lastAssistantMessage.id, 'regenerate');
  };

  const handlePanelChange = (panel: 'memory' | 'server' | 'settings') => {
    setActivePanel(panel);
  };

  return (
    <div className="flex h-screen bg-background overflow-hidden">
      {/* Error banner */}
      {conversationsError && (
        <div className="fixed top-0 left-0 right-0 bg-destructive text-destructive-foreground p-3 text-center z-[1000]">
          {conversationsError}
        </div>
      )}

      {/* Sidebar with panel navigation */}
      <Sidebar
        conversations={conversations}
        selectedId={activeConversationId}
        activeView={activePanel}
        onSelect={handleSelectConversation}
        onNew={handleNewConversation}
        onDelete={handleDeleteConversation}
        onRenameConversation={handleRenameConversation}
        onArchive={handleArchiveConversation}
        onUnarchive={handleUnarchiveConversation}
        onSettings={() => setActivePanel('settings')}
        onPanelChange={handlePanelChange}
        loading={conversationsLoading}
      />

      {/* Main Content Area */}
      <main className="flex-1 layout-stack min-w-0 min-h-0">
        {activePanel === 'chat' && (
          <ChatWindow
            conversationId={activeConversationId}
            conversationTitle="Conversation"
            onSendMessage={handleSendMessage}
            onStopStreaming={handleStopStreaming}
            onRegenerateResponse={handleRegenerateResponse}
            showControls={true}
          />
        )}

        {activePanel === 'memory' && (
          <div className="flex-1 overflow-hidden p-6">
            <MemoryManager />
          </div>
        )}

        {activePanel === 'server' && (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="max-w-2xl mx-auto">
              <h1 className="text-2xl font-semibold text-foreground mb-6">Server Information</h1>
              <ServerInfoPanel />
            </div>
          </div>
        )}

        {activePanel === 'settings' && (
          <Settings
            conversationId={activeConversationId}
          />
        )}
      </main>
    </div>
  );
}
