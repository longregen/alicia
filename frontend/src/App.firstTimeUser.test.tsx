import { render, screen, waitFor, cleanup, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import App from './App';
import * as useConversationsModule from './hooks/useConversations';
import * as useDatabaseModule from './hooks/useDatabase';
import * as useMessagesModule from './hooks/useMessages';

// Mock the hooks
vi.mock('./hooks/useConversations');
vi.mock('./hooks/useDatabase');
vi.mock('./hooks/useMessages');
vi.mock('./hooks/useAsync');

// Mock ConfigProvider and MessageProvider contexts
vi.mock('./contexts/ConfigContext', () => ({
  ConfigProvider: ({ children }: { children: React.ReactNode }) => children,
  useConfig: () => ({ config: null }),
}));

vi.mock('./contexts/MessageContext', () => ({
  MessageProvider: ({ children }: { children: React.ReactNode }) => children,
  useMessage: () => ({
    messages: [],
    addMessage: vi.fn(),
    updateMessage: vi.fn(),
    clearMessages: vi.fn(),
  }),
  useMessageContext: () => ({
    streamingMessages: new Map(),
    currentTranscription: '',
    isGenerating: false,
    currentGeneratingMessageId: null,
    reasoningSteps: [],
    toolUsages: [],
    errors: [],
    memoryTraces: [],
    commentaries: [],
    acknowledgements: [],
    messages: [],
    updateStreamingSentence: vi.fn(),
    clearStreamingSentences: vi.fn(),
    finalizeStreamingMessage: vi.fn(),
    addMessage: vi.fn(),
    setTranscription: vi.fn(),
    clearTranscription: vi.fn(),
    setIsGenerating: vi.fn(),
    addError: vi.fn(),
    addReasoningStep: vi.fn(),
    addToolUsage: vi.fn(),
    updateToolUsageResult: vi.fn(),
    handleAcknowledgement: vi.fn(),
    addMemoryTrace: vi.fn(),
    addCommentary: vi.fn(),
    clearProtocolMessages: vi.fn(),
    clearMessages: vi.fn(),
  }),
}));

// Mock storage utility - simple mock that always returns null
// This ensures each test starts with no selected conversation
vi.mock('./utils/storage', () => ({
  storage: {
    getSelectedConversationId: vi.fn(() => null),
    setSelectedConversationId: vi.fn(),
    getVoiceMode: vi.fn(() => false),
    setVoiceMode: vi.fn(),
  },
}));

// Mock the Sidebar component to simplify testing
vi.mock('./components/Sidebar', () => ({
  Sidebar: ({ conversations, onNew }: { conversations: unknown[]; onNew: () => void }) => (
    <div data-testid="sidebar">
      <button onClick={onNew} data-testid="new-conversation-btn">New Conversation</button>
      <div data-testid="conversations-list">
        {conversations.length === 0 ? (
          <div data-testid="empty-conversations">No conversations</div>
        ) : (
          <div>{conversations.length} conversations</div>
        )}
      </div>
    </div>
  ),
}));

// Mock ChatWindowBridge to simplify testing
vi.mock('./components/organisms/ChatWindowBridge', () => ({
  default: ({ conversationId }: { conversationId: string | null }) => (
    <div data-testid="chat-window">
      {conversationId ? (
        <div data-testid="active-conversation">Active conversation: {conversationId}</div>
      ) : (
        <div data-testid="empty-state">
          <p>No conversation selected</p>
        </div>
      )}
    </div>
  ),
}));

// Mock WelcomeScreen component
vi.mock('./components/organisms/WelcomeScreen', () => ({
  default: ({ onNewConversation, conversations, onSelectConversation }: {
    onNewConversation: () => void;
    conversations: { id: string; title: string }[];
    onSelectConversation: (id: string) => void;
  }) => (
    <div data-testid="welcome-screen">
      <h1>Welcome to Alicia</h1>
      <button onClick={onNewConversation} data-testid="start-new-chat-btn">
        Start New Chat
      </button>
      {conversations.length > 0 && (
        <div data-testid="recent-conversations">
          {conversations.map((c) => (
            <button
              key={c.id}
              onClick={() => onSelectConversation(c.id)}
              data-testid={`conversation-${c.id}`}
            >
              {c.title}
            </button>
          ))}
        </div>
      )}
    </div>
  ),
}));

// Mock Settings component
vi.mock('./components/Settings', () => ({
  Settings: () => <div data-testid="settings">Settings Panel</div>,
}));

describe('App - First Time User UX with Welcome Screen', () => {
  const mockCreateConversation = vi.fn();
  const mockDeleteConversation = vi.fn();
  const mockUpdateConversation = vi.fn();
  const mockSendMessage = vi.fn();
  const mockRefresh = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    // Mock useDatabase to return ready state
    vi.spyOn(useDatabaseModule, 'useDatabase').mockReturnValue({
      isReady: true,
      error: null,
    });

    // Mock useConversations to return empty array (first-time user scenario)
    vi.spyOn(useConversationsModule, 'useConversations').mockReturnValue({
      conversations: [],
      loading: false,
      error: null,
      createConversation: mockCreateConversation,
      deleteConversation: mockDeleteConversation,
      updateConversation: mockUpdateConversation,
      refetch: vi.fn(),
    });

    // Mock useMessages
    vi.spyOn(useMessagesModule, 'useMessages').mockReturnValue({
      messages: [],
      loading: false,
      error: null,
      sending: false,
      sendMessage: mockSendMessage,
      isSyncing: false,
      lastSyncTime: null,
      syncError: null,
      syncNow: vi.fn(),
      refresh: mockRefresh,
    });
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  describe('First-time user experience', () => {
    it('should show Welcome Screen for first-time users instead of auto-creating', async () => {
      // ARRANGE & ACT: Render the App as a first-time user would see it
      render(<App />);

      // ASSERT: Wait for database to be ready
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // EXPECTED BEHAVIOR: Welcome Screen should be shown, NOT auto-create
      expect(screen.getByTestId('welcome-screen')).toBeInTheDocument();
      expect(screen.getByText('Welcome to Alicia')).toBeInTheDocument();
      expect(screen.getByTestId('start-new-chat-btn')).toBeInTheDocument();

      // No conversation should be auto-created
      expect(mockCreateConversation).not.toHaveBeenCalled();
    });

    it('should create conversation when user clicks Start New Chat', async () => {
      // ARRANGE
      const newConversation = {
        id: 'new-conv-1',
        title: 'New Chat',
        status: 'active' as const,
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      mockCreateConversation.mockResolvedValue(newConversation);

      render(<App />);

      // Wait for Welcome Screen
      await waitFor(() => {
        expect(screen.getByTestId('welcome-screen')).toBeInTheDocument();
      });

      // ACT: Click "Start New Chat" button
      fireEvent.click(screen.getByTestId('start-new-chat-btn'));

      // ASSERT: createConversation should be called
      await waitFor(() => {
        expect(mockCreateConversation).toHaveBeenCalledTimes(1);
        expect(mockCreateConversation).toHaveBeenCalledWith('New Chat');
      });
    });

    it('should not show Welcome Screen when conversation is selected', async () => {
      // ARRANGE: Set up with an existing conversation selected
      const existingConversation = {
        id: 'existing-conv-1',
        title: 'Existing Chat',
        status: 'active' as const,
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      vi.spyOn(useConversationsModule, 'useConversations').mockReturnValue({
        conversations: [existingConversation],
        loading: false,
        error: null,
        createConversation: mockCreateConversation,
        deleteConversation: mockDeleteConversation,
        updateConversation: mockUpdateConversation,
        refetch: vi.fn(),
      });

      // Mock storage to return existing conversation ID
      const storageModule = await import('./utils/storage');
      vi.mocked(storageModule.storage.getSelectedConversationId).mockReturnValue('existing-conv-1');

      render(<App />);

      // Wait for app to load
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // ASSERT: Chat window should be shown, NOT Welcome Screen
      expect(screen.queryByTestId('welcome-screen')).not.toBeInTheDocument();
      expect(screen.getByTestId('chat-window')).toBeInTheDocument();
    });
  });

  describe('Returning user with stale localStorage', () => {
    it('should show Welcome Screen when stored conversation ID does not exist on server', async () => {
      // ARRANGE: Storage has a conversation ID that doesn't exist in conversations list
      const storageModule = await import('./utils/storage');
      vi.mocked(storageModule.storage.getSelectedConversationId).mockReturnValue('stale-conv-id');

      // Server returns empty conversations (or conversations that don't include the stale ID)
      vi.spyOn(useConversationsModule, 'useConversations').mockReturnValue({
        conversations: [],
        loading: false,
        error: null,
        createConversation: mockCreateConversation,
        deleteConversation: mockDeleteConversation,
        updateConversation: mockUpdateConversation,
        refetch: vi.fn(),
      });

      render(<App />);

      // Wait for app to process
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // ASSERT: Welcome Screen should be shown after stale ID is cleared
      await waitFor(() => {
        expect(screen.getByTestId('welcome-screen')).toBeInTheDocument();
      });

      // No auto-create should happen
      expect(mockCreateConversation).not.toHaveBeenCalled();
    });
  });
});
