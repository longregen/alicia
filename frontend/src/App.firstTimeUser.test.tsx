import { render, screen, waitFor, cleanup } from '@testing-library/react';
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
          <p>This is just an empty state with no clear call-to-action</p>
        </div>
      )}
    </div>
  ),
}));

// Mock Settings component
vi.mock('./components/Settings', () => ({
  Settings: () => <div data-testid="settings">Settings Panel</div>,
}));

describe('App - First Time User UX', () => {
  const mockCreateConversation = vi.fn();
  const mockDeleteConversation = vi.fn();
  const mockUpdateConversation = vi.fn();
  const mockSendMessage = vi.fn();
  const mockRefresh = vi.fn();

  beforeEach(() => {
    // Set up default mock return value for createConversation BEFORE clearing mocks
    const autoCreatedConversation = {
      id: 'auto-created-1',
      title: 'New Chat',
      status: 'active' as const,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    vi.clearAllMocks();
    mockCreateConversation.mockResolvedValue(autoCreatedConversation);

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
    it('should auto-create a conversation for first-time users', async () => {
      // ARRANGE: Render the App component as a first-time user would see it
      // - Empty database (no conversations)
      // - No localStorage state (no selectedConversationId)
      const autoCreatedConversation = {
        id: 'auto-created-1',
        title: 'New Chat',
        status: 'active' as const,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      mockCreateConversation.mockResolvedValue(autoCreatedConversation);

      render(<App />);

      // ASSERT: Wait for the app to finish loading
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // EXPECTED BEHAVIOR: A conversation should be auto-created for the user
      await waitFor(() => {
        expect(mockCreateConversation).toHaveBeenCalledTimes(1);
        expect(mockCreateConversation).toHaveBeenCalledWith('New Chat');
      });
    });



    it('should verify database is ready and auto-create happens', async () => {
      // ARRANGE
      render(<App />);

      // ASSERT: Database should be ready
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // Auto-creation should have been triggered
      await waitFor(() => {
        expect(mockCreateConversation).toHaveBeenCalledTimes(1);
      });

      // This test verifies the FIX: ready database + empty conversations = auto-create conversation
    });
  });

  describe('Expected behavior (what we want to fix)', () => {
    it('should auto-create a conversation for first-time users with empty database', async () => {
      // This test documents the DESIRED behavior that we want to implement

      // ARRANGE: Set up mock to simulate auto-creation
      const autoCreatedConversation = {
        id: 'auto-created-1',
        title: 'New Chat',
        status: 'active' as const,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      mockCreateConversation.mockResolvedValue(autoCreatedConversation);

      // ACT
      render(<App />);

      // ASSERT: After initial load with empty conversations, should auto-create one
      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // Expected: createConversation should be called automatically
      // This will FAIL with current implementation
      expect(mockCreateConversation).toHaveBeenCalledTimes(1);
      expect(mockCreateConversation).toHaveBeenCalledWith('New Chat');
    });

    it('should not require manual user action to start first conversation', async () => {
      // Verify that the auto-creation approach eliminates the need for manual action

      // ARRANGE & ACT
      render(<App />);

      await waitFor(() => {
        expect(screen.queryByText('Initializing database...')).not.toBeInTheDocument();
      });

      // ASSERT: createConversation should be called automatically
      // User doesn't need to click anything
      await waitFor(() => {
        expect(mockCreateConversation).toHaveBeenCalledTimes(1);
      });
    });
  });
});
