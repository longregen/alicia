import { describe, it, expect, beforeEach, vi } from 'vitest';
import { handleStartAnswer, handleAssistantSentence, setMessageSender } from './protocolAdapter';
import {
  NormalizedMessage,
  MessageSentence,
  ToolCall,
  MemoryTrace,
  AudioRef,
  MessageId,
  ConversationId,
  createMessageId,
  MessageStatus,
} from '../types/streaming';

// Mock the messageRepository to avoid database initialization
vi.mock('../db/repository', () => ({
  messageRepository: {
    upsert: vi.fn(),
    findById: vi.fn(),
  },
}));

const createMockStore = () => {
  const store = {
    messages: {} as Record<string, NormalizedMessage>,
    sentences: {} as Record<string, MessageSentence>,
    toolCalls: {} as Record<string, ToolCall>,
    audioRefs: {} as Record<string, AudioRef>,
    memoryTraces: {} as Record<string, MemoryTrace>,
    currentStreamingMessageId: null as MessageId | null,
    currentConversationId: null as ConversationId | null,

    addMessage: vi.fn((message: NormalizedMessage) => {
      store.messages[message.id] = message;
    }),

    updateMessage: vi.fn((id: MessageId, updates: Partial<NormalizedMessage>) => {
      const message = store.messages[id];
      if (message) {
        store.messages[id] = { ...message, ...updates };
      }
    }),

    updateMessageContent: vi.fn((id: MessageId, content: string) => {
      const message = store.messages[id];
      if (message) {
        store.messages[id] = { ...message, content };
      }
    }),

    updateMessageStatus: vi.fn((id: MessageId, status: MessageStatus) => {
      const message = store.messages[id];
      if (message) {
        store.messages[id] = { ...message, status };
      }
    }),

    addSentence: vi.fn((sentence: MessageSentence) => {
      store.sentences[sentence.id] = sentence;
      const message = store.messages[sentence.messageId];
      if (message) {
        message.sentenceIds.push(sentence.id);
      }
    }),

    updateSentence: vi.fn(),

    getMessageSentences: vi.fn((messageId: MessageId) => {
      return Object.values(store.sentences).filter(s => s.messageId === messageId);
    }),

    addToolCall: vi.fn((toolCall: ToolCall) => {
      store.toolCalls[toolCall.id] = toolCall;
    }),

    updateToolCall: vi.fn(),

    addAudioRef: vi.fn((audioRef: AudioRef) => {
      store.audioRefs[audioRef.id] = audioRef;
    }),

    addMemoryTrace: vi.fn((trace: MemoryTrace) => {
      store.memoryTraces[trace.id] = trace;
    }),

    setCurrentStreamingMessageId: vi.fn((id: MessageId | null) => {
      store.currentStreamingMessageId = id;
    }),

    setCurrentConversationId: vi.fn((id: ConversationId | null) => {
      store.currentConversationId = id;
    }),

    clearConversation: vi.fn(),
    loadConversation: vi.fn(),
    mergeMessages: vi.fn(),
    getMessageToolCalls: vi.fn(() => []),
    getMessageMemoryTraces: vi.fn(() => []),
    refreshRequestCounter: 0,
    requestMessagesRefresh: vi.fn(),
  };

  return store;
};

describe('currentStreamingMessageId not cleared on error', () => {
  let mockStore: ReturnType<typeof createMockStore>;

  beforeEach(() => {
    mockStore = createMockStore();
    vi.clearAllMocks();

    // Clear message sender
    setMessageSender(null);
  });

  describe('Normal Flow (Control)', () => {
    it('clears currentStreamingMessageId when isFinal is received', () => {
      const messageId = createMessageId('msg-001');

      // Start streaming
      handleStartAnswer(
        {
          id: 'msg-001',
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      expect(mockStore.currentStreamingMessageId).toBe(messageId);

      // Set up message in store (simulating what handleStartAnswer does)
      mockStore.currentStreamingMessageId = messageId;

      // Send first sentence
      handleAssistantSentence(
        {
          id: 'sent-001',
          text: 'Hello',
          sequence: 0,
          isFinal: false,
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      // Should still be streaming
      expect(mockStore.setCurrentStreamingMessageId).not.toHaveBeenCalledWith(null);

      // Send final sentence
      handleAssistantSentence(
        {
          id: 'sent-002',
          text: 'world',
          sequence: 1,
          isFinal: true,
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      // Should clear streaming state
      expect(mockStore.setCurrentStreamingMessageId).toHaveBeenCalledWith(null);
    });
  });

  describe('Error Cases', () => {
    it('currentStreamingMessageId NOT cleared when WebSocket closes during streaming', () => {
      const messageId = createMessageId('msg-001');

      // Start streaming
      handleStartAnswer(
        {
          id: 'msg-001',
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      expect(mockStore.currentStreamingMessageId).toBe(messageId);
      mockStore.currentStreamingMessageId = messageId;

      // Send one sentence
      handleAssistantSentence(
        {
          id: 'sent-001',
          text: 'Hello',
          sequence: 0,
          isFinal: false,
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      // Still streaming
      expect(mockStore.currentStreamingMessageId).toBe(messageId);

      // WebSocket closes unexpectedly (no more messages arrive)
      // In real code, ws.onclose fires but there's no cleanup of currentStreamingMessageId

      // VERIFICATION: currentStreamingMessageId is still set
      expect(mockStore.currentStreamingMessageId).toBe(messageId);

      // This is the bug: the streaming state is stuck forever
      // New messages cannot stream because currentStreamingMessageId is not null
    });

    it('currentStreamingMessageId NOT cleared when network error mid-stream', () => {
      const messageId = createMessageId('msg-001');

      // Start streaming
      handleStartAnswer(
        {
          id: 'msg-001',
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      mockStore.currentStreamingMessageId = messageId;

      // Send several sentences
      for (let i = 0; i < 5; i++) {
        handleAssistantSentence(
          {
            id: `sent-${i}`,
            text: `Sentence ${i}`,
            sequence: i,
            isFinal: false,
            conversationId: 'conv-001',
            previousId: '',
          },
          mockStore
        );
      }

      // Network error occurs - connection drops mid-stream
      // No isFinal ever arrives

      // VERIFICATION: currentStreamingMessageId is still set
      expect(mockStore.currentStreamingMessageId).toBe(messageId);

      // Message is stuck in streaming state with partial content
      const message = mockStore.messages[messageId];
      expect(message.status).toBe(MessageStatus.Streaming);
      expect(message.sentenceIds.length).toBe(5);
    });
  });

  describe('Impact Analysis', () => {
    it('demonstrates that stuck streaming state blocks new streaming messages', () => {
      const messageId1 = createMessageId('msg-001');

      // Start first message streaming
      handleStartAnswer(
        {
          id: 'msg-001',
          conversationId: 'conv-001',
          previousId: '',
        },
        mockStore
      );

      mockStore.currentStreamingMessageId = messageId1;

      // Simulate error - WebSocket closes without isFinal
      // currentStreamingMessageId is stuck at messageId1

      // Try to start a new streaming message (e.g., after reconnect)
      handleStartAnswer(
        {
          id: 'msg-002',
          conversationId: 'conv-001',
          previousId: 'msg-001',
        },
        {
          ...mockStore,
          clearConversation: vi.fn(),
          loadConversation: vi.fn(),
          mergeMessages: vi.fn(),
          getMessageToolCalls: vi.fn(() => []),
          getMessageMemoryTraces: vi.fn(() => []),
        }
      );

      // The new message will start streaming and overwrite currentStreamingMessageId
      // but the old message is still stuck in Streaming status
      const oldMessage = mockStore.messages[messageId1];
      expect(oldMessage.status).toBe(MessageStatus.Streaming);

      // This creates inconsistent state:
      // - msg-001 is in Streaming status but not tracked by currentStreamingMessageId
      // - msg-002 is now the current streaming message
      // - msg-001 will never be marked Complete
    });
  });
});
