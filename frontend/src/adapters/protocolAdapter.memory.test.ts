import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  handleStartAnswer,
  handleAssistantSentence,
  handleAudioChunk,
  resetAdapterState,
  cleanupConversationContext,
  getConversationContextCount,
  hasConversationContext,
} from './protocolAdapter';
import {
  StartAnswer,
  AssistantSentence,
  AudioChunk,
} from '../types/protocol';
import {
  NormalizedMessage,
  MessageSentence,
  MessageStatus,
  AudioRef,
  MessageId,
  SentenceId,
  ConversationId,
  createAudioRefId,
} from '../types/streaming';

// Mock audioManager
vi.mock('../utils/audioManager', () => ({
  audioManager: {
    store: vi.fn(),
  },
}));

// Import audioManager after mocking
import { audioManager } from '../utils/audioManager';

// Mock the store with a minimal implementation
const createMockStore = () => {
  const store = {
    messages: {} as Record<string, NormalizedMessage>,
    sentences: {} as Record<string, MessageSentence>,
    audioRefs: {} as Record<string, AudioRef>,
    currentStreamingMessageId: null as MessageId | null,
    currentConversationId: null as ConversationId | null,

    addMessage: vi.fn((message: NormalizedMessage) => {
      store.messages[message.id] = message;
    }),

    updateMessageStatus: vi.fn((id: MessageId, status: MessageStatus) => {
      if (store.messages[id]) {
        store.messages[id].status = status;
      }
    }),

    updateMessageContent: vi.fn((id: MessageId, content: string) => {
      if (store.messages[id]) {
        store.messages[id].content = content;
      }
    }),

    addSentence: vi.fn((sentence: MessageSentence) => {
      store.sentences[sentence.id] = sentence;
      const message = store.messages[sentence.messageId];
      if (message && !message.sentenceIds.includes(sentence.id)) {
        message.sentenceIds.push(sentence.id);
      }
    }),

    updateSentence: vi.fn((id: SentenceId, update: Partial<MessageSentence>) => {
      if (store.sentences[id]) {
        store.sentences[id] = { ...store.sentences[id], ...update };
      }
    }),

    addAudioRef: vi.fn((audioRef: AudioRef) => {
      store.audioRefs[audioRef.id] = audioRef;
    }),

    setCurrentStreamingMessageId: vi.fn((id: MessageId | null) => {
      store.currentStreamingMessageId = id;
    }),

    setCurrentConversationId: vi.fn((id: ConversationId | null) => {
      store.currentConversationId = id;
    }),

    getMessageSentences: vi.fn((messageId: MessageId): MessageSentence[] => {
      return Object.values(store.sentences)
        .filter((s) => s.messageId === messageId)
        .sort((a, b) => a.sequence - b.sequence);
    }),
  };

  return store;
};

// Mock useConversationStore
vi.mock('../stores/conversationStore', () => ({
  useConversationStore: {
    getState: vi.fn(),
  },
}));

import { useConversationStore } from '../stores/conversationStore';

describe('protocolAdapter - Memory Leak Tests', () => {
  let mockStore: ReturnType<typeof createMockStore>;

  beforeEach(() => {
    mockStore = createMockStore();
    vi.mocked(useConversationStore.getState).mockReturnValue(mockStore as any);
    vi.mocked(audioManager.store).mockResolvedValue(createAudioRefId('audio-ref-id'));

    // Reset adapter state before each test
    resetAdapterState();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('conversationContexts memory leak', () => {
    it('should clean up old conversation when switching to a new one', async () => {
      // Start with no contexts
      expect(getConversationContextCount()).toBe(0);

      // === Process messages for conversation A ===
      const messageA: StartAnswer = {
        id: 'msg-a',
        previousId: 'prev-a',
        conversationId: 'conv-a',
      };
      handleStartAnswer(messageA, mockStore as any);

      const sentenceA: AssistantSentence = {
        id: 'sent-a-1',
        previousId: 'msg-a',
        conversationId: 'conv-a',
        text: 'Hello from conversation A',
        sequence: 0,
        isFinal: false,
      };
      handleAssistantSentence(sentenceA, mockStore as any);

      // Conversation A now has a context
      expect(hasConversationContext('conv-a')).toBe(true);
      expect(getConversationContextCount()).toBe(1);

      const audioA: AudioChunk = {
        conversationId: 'conv-a',
        sequence: 0,
        data: new Uint8Array([1, 2, 3]),
        durationMs: 1000,
        trackSid: 'track-a',
        format: 'pcm_s16le_24000',
      };
      await handleAudioChunk(audioA, mockStore as any);

      // === Process messages for conversation B ===
      const messageB: StartAnswer = {
        id: 'msg-b',
        previousId: 'prev-b',
        conversationId: 'conv-b',
      };
      handleStartAnswer(messageB, mockStore as any);

      const sentenceB: AssistantSentence = {
        id: 'sent-b-1',
        previousId: 'msg-b',
        conversationId: 'conv-b',
        text: 'Hello from conversation B',
        sequence: 0,
        isFinal: false,
      };
      handleAssistantSentence(sentenceB, mockStore as any);

      // Now we have contexts for both conversations
      expect(hasConversationContext('conv-a')).toBe(true);
      expect(hasConversationContext('conv-b')).toBe(true);
      expect(getConversationContextCount()).toBe(2);

      // === Switch away from conversation A ===
      // Clean up conversation A's context when switching to conversation B
      cleanupConversationContext('conv-a');

      // Conversation A should be cleaned up
      expect(hasConversationContext('conv-a')).toBe(false);
      // Conversation B should still exist
      expect(hasConversationContext('conv-b')).toBe(true);
      expect(getConversationContextCount()).toBe(1);

      // Conversation B should still function normally
      const sentenceB2: AssistantSentence = {
        id: 'sent-b-2',
        previousId: 'msg-b',
        conversationId: 'conv-b',
        text: 'Still working',
        sequence: 1,
        isFinal: false,
      };
      handleAssistantSentence(sentenceB2, mockStore as any);

      expect(mockStore.addSentence).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'Still working',
        })
      );
    });

    it('should NOT clean up conversation B when only conversation A is cleaned up', async () => {
      const conversationIdA = 'conv-a';
      const conversationIdB = 'conv-b';

      // Create context for conversation A
      const messageA: StartAnswer = {
        id: 'msg-a',
        previousId: 'prev-a',
        conversationId: conversationIdA,
      };
      handleStartAnswer(messageA, mockStore as any);

      const sentenceA: AssistantSentence = {
        id: 'sent-a-1',
        previousId: 'msg-a',
        conversationId: conversationIdA,
        text: 'Hello from A',
        sequence: 0,
        isFinal: false,
      };
      handleAssistantSentence(sentenceA, mockStore as any);

      // Create context for conversation B
      const messageB: StartAnswer = {
        id: 'msg-b',
        previousId: 'prev-b',
        conversationId: conversationIdB,
      };
      handleStartAnswer(messageB, mockStore as any);

      const sentenceB: AssistantSentence = {
        id: 'sent-b-1',
        previousId: 'msg-b',
        conversationId: conversationIdB,
        text: 'Hello from B',
        sequence: 0,
        isFinal: false,
      };
      handleAssistantSentence(sentenceB, mockStore as any);

      // Both contexts exist
      expect(hasConversationContext(conversationIdA)).toBe(true);
      expect(hasConversationContext(conversationIdB)).toBe(true);
      expect(getConversationContextCount()).toBe(2);

      // Clean up only conversation A
      cleanupConversationContext(conversationIdA);

      // Conversation A should be cleaned up
      expect(hasConversationContext(conversationIdA)).toBe(false);
      // Conversation B should still exist
      expect(hasConversationContext(conversationIdB)).toBe(true);
      expect(getConversationContextCount()).toBe(1);

      // After cleanup, conversation B should still work normally
      const sentenceB2: AssistantSentence = {
        id: 'sent-b-2',
        previousId: 'msg-b',
        conversationId: conversationIdB,
        text: 'Still working',
        sequence: 1,
        isFinal: false,
      };
      handleAssistantSentence(sentenceB2, mockStore as any);

      expect(mockStore.addSentence).toHaveBeenCalledWith(
        expect.objectContaining({
          content: 'Still working',
        })
      );
    });

    it('should handle cleanup when switching through multiple conversations', async () => {
      expect(getConversationContextCount()).toBe(0);

      const conversationIds = ['conv-1', 'conv-2', 'conv-3', 'conv-4', 'conv-5'];

      // Simulate user switching through 5 different conversations
      for (let i = 0; i < conversationIds.length; i++) {
        const convId = conversationIds[i];

        const message: StartAnswer = {
          id: `msg-${convId}`,
          previousId: `prev-${convId}`,
          conversationId: convId,
        };
        handleStartAnswer(message, mockStore as any);

        const sentence: AssistantSentence = {
          id: `sent-${convId}-1`,
          previousId: `msg-${convId}`,
          conversationId: convId,
          text: `Message from ${convId}`,
          sequence: 0,
          isFinal: false,
        };
        handleAssistantSentence(sentence, mockStore as any);

        const audio: AudioChunk = {
          conversationId: convId,
          sequence: 0,
          data: new Uint8Array([1, 2, 3]),
          durationMs: 1000,
          trackSid: `track-${convId}`,
          format: 'pcm_s16le_24000',
        };
        await handleAudioChunk(audio, mockStore as any);

        // Clean up previous conversation when switching (except on first iteration)
        if (i > 0) {
          const previousConvId = conversationIds[i - 1];
          cleanupConversationContext(previousConvId);
          expect(hasConversationContext(previousConvId)).toBe(false);
        }
      }

      // Only the last conversation should have a context
      expect(getConversationContextCount()).toBe(1);
      expect(hasConversationContext('conv-5')).toBe(true);
      expect(hasConversationContext('conv-1')).toBe(false);
      expect(hasConversationContext('conv-2')).toBe(false);
      expect(hasConversationContext('conv-3')).toBe(false);
      expect(hasConversationContext('conv-4')).toBe(false);

      expect(mockStore.addSentence).toHaveBeenCalledTimes(5);
    });

  });

  describe('proposed solution verification', () => {
    it('should have a cleanupConversationContext function', () => {
      // Verify that cleanupConversationContext works correctly

      const conversationId = 'conv-test';

      const message: StartAnswer = {
        id: 'msg-test',
        previousId: 'prev-test',
        conversationId,
      };
      handleStartAnswer(message, mockStore as any);

      const sentence: AssistantSentence = {
        id: 'sent-test',
        previousId: 'msg-test',
        conversationId,
        text: 'Test',
        sequence: 0,
        isFinal: false,
      };
      handleAssistantSentence(sentence, mockStore as any);

      // Context exists
      expect(hasConversationContext(conversationId)).toBe(true);
      expect(getConversationContextCount()).toBe(1);

      // Clean up the conversation context
      cleanupConversationContext(conversationId);

      // Context should be removed
      expect(hasConversationContext(conversationId)).toBe(false);
      expect(getConversationContextCount()).toBe(0);
    });

    it('demonstrates resetAdapterState is too aggressive for selective cleanup', () => {
      // resetAdapterState() clears ALL contexts, not just inactive ones
      // This makes it unsuitable for use during normal operation

      const activeConversation = 'conv-active';
      const inactiveConversation = 'conv-inactive';

      // Create two conversations with sentences (this creates contexts)
      handleStartAnswer(
        { id: 'msg-active', previousId: 'prev-active', conversationId: activeConversation },
        mockStore as any
      );
      handleAssistantSentence(
        {
          id: 'sent-active',
          previousId: 'msg-active',
          conversationId: activeConversation,
          text: 'Active',
          sequence: 0,
          isFinal: false,
        },
        mockStore as any
      );

      handleStartAnswer(
        { id: 'msg-inactive', previousId: 'prev-inactive', conversationId: inactiveConversation },
        mockStore as any
      );
      handleAssistantSentence(
        {
          id: 'sent-inactive',
          previousId: 'msg-inactive',
          conversationId: inactiveConversation,
          text: 'Inactive',
          sequence: 0,
          isFinal: false,
        },
        mockStore as any
      );

      expect(getConversationContextCount()).toBe(2);

      // User is actively using activeConversation
      // We want to clean up inactiveConversation only

      // But resetAdapterState() clears BOTH:
      resetAdapterState();

      expect(hasConversationContext(activeConversation)).toBe(false); // TOO AGGRESSIVE!
      expect(hasConversationContext(inactiveConversation)).toBe(false);
      expect(getConversationContextCount()).toBe(0);

      // This breaks the active conversation's audio sentence associations
    });
  });
});
