import { useState } from 'react';
import { Sidebar } from '../Sidebar';
import ChatWindow from './ChatWindow';
import { MemoryManager } from './MemoryManager';
import ServerInfoPanel from './ServerPanel/ServerInfoPanel';
import { Settings } from '../Settings';
import { useConversations } from '../../hooks/useConversations';
import { useConversationStore } from '../../stores/conversationStore';
import type { VoiceState } from '../atoms/VoiceVisualizer';
import { sendControlStop, sendControlVariation } from '../../adapters/protocolAdapter';

/**
 * Panel types for the main navigation
 */
export type Panel = 'chat' | 'memory' | 'server' | 'settings';

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
  const [_voiceState, _setVoiceState] = useState<VoiceState>('idle');

  // Log conversation errors (could be enhanced with toast notifications)
  if (conversationsError) {
    console.error('Conversations error:', conversationsError);
  }

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
    // TODO: Implement message sending
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
      {/* Sidebar with panel navigation */}
      <Sidebar
        conversations={conversations}
        selectedId={activeConversationId}
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
      <main className="flex-1 flex flex-col min-w-0 min-h-0">
        {activePanel === 'chat' && (
          <ChatWindow
            conversationId={activeConversationId}
            conversationTitle="Conversation"
            onSendMessage={handleSendMessage}
            onStopStreaming={handleStopStreaming}
            onRegenerateResponse={handleRegenerateResponse}
            useSileroVAD={false}
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
              <h1 className="text-2xl font-semibold text-default mb-6">Server Information</h1>
              <ServerInfoPanel />
            </div>
          </div>
        )}

        {activePanel === 'settings' && (
          <Settings
            isOpen={true}
            onClose={() => setActivePanel('chat')}
            conversationId={activeConversationId}
          />
        )}
      </main>
    </div>
  );
}
