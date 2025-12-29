import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  handleProtocolMessage,
  handleStartAnswer,
  handleAssistantSentence,
  handleToolUseRequest,
  handleToolUseResult,
  handleReasoningStep,
  handleAudioChunk,
  handleTranscription,
  handleMemoryTrace,
  resetAdapterState,
} from './protocolAdapter';
import {
  Envelope,
  MessageType,
  StartAnswer,
  AssistantSentence,
  ToolUseRequest,
  ToolUseResult,
  ReasoningStep,
  AudioChunk,
  Transcription,
  MemoryTrace as ProtocolMemoryTrace,
} from '../types/protocol';
import {
  Message,
  MessageSentence,
  ToolCall,
  MemoryTrace,
  MessageStatus,
  AudioRef,
  createMessageId,
  createSentenceId,
  createToolCallId,
  createMemoryTraceId,
  createConversationId,
  createAudioRefId,
  MessageId,
  SentenceId,
  ConversationId,
  AudioRefId,
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
    messages: {} as Record<string, Message>,
    sentences: {} as Record<string, MessageSentence>,
    toolCalls: {} as Record<string, ToolCall>,
    audioRefs: {} as Record<string, AudioRef>,
    memoryTraces: {} as Record<string, MemoryTrace>,
    currentStreamingMessageId: null as MessageId | null,
    currentConversationId: null as ConversationId | null,

    addMessage: vi.fn((message: Message) => {
      store.messages[message.id] = message;
    }),

    updateMessageStatus: vi.fn((id: MessageId, status: MessageStatus) => {
      if (store.messages[id]) {
        store.messages[id].status = status;
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

    addToolCall: vi.fn((toolCall: ToolCall) => {
      store.toolCalls[toolCall.id] = toolCall;
      const message = store.messages[toolCall.messageId];
      if (message && !message.toolCallIds.includes(toolCall.id)) {
        message.toolCallIds.push(toolCall.id);
      }
    }),

    updateToolCall: vi.fn((id: string, update: ToolCall) => {
      store.toolCalls[id] = update;
    }),

    addAudioRef: vi.fn((audioRef: AudioRef) => {
      store.audioRefs[audioRef.id] = audioRef;
    }),

    addMemoryTrace: vi.fn((trace: MemoryTrace) => {
      store.memoryTraces[trace.id] = trace;
      const message = store.messages[trace.messageId];
      if (message && !message.memoryTraceIds.includes(trace.id)) {
        message.memoryTraceIds.push(trace.id);
      }
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

describe('protocolAdapter', () => {
  let mockStore: ReturnType<typeof createMockStore>;

  beforeEach(() => {
    mockStore = createMockStore();
    vi.mocked(useConversationStore.getState).mockReturnValue(mockStore as any);
    vi.clearAllMocks();
    resetAdapterState();
  });

  afterEach(() => {
    resetAdapterState();
  });

  describe('handleStartAnswer', () => {
    it('creates a new streaming assistant message', () => {
      const startAnswer: StartAnswer = {
        id: 'msg-001',
        previousId: 'msg-000',
        conversationId: 'conv-001',
        answerType: 'text',
      };

      handleStartAnswer(startAnswer, mockStore as any);

      expect(mockStore.addMessage).toHaveBeenCalledTimes(1);
      const message = mockStore.addMessage.mock.calls[0][0];
      expect(message.id).toBe(createMessageId('msg-001'));
      expect(message.conversationId).toBe(createConversationId('conv-001'));
      expect(message.role).toBe('assistant');
      expect(message.content).toBe('');
      expect(message.status).toBe(MessageStatus.Streaming);
      expect(message.sentenceIds).toEqual([]);
      expect(message.toolCallIds).toEqual([]);
      expect(message.memoryTraceIds).toEqual([]);

      expect(mockStore.setCurrentStreamingMessageId).toHaveBeenCalledWith(
        createMessageId('msg-001')
      );
      expect(mockStore.setCurrentConversationId).toHaveBeenCalledWith(
        createConversationId('conv-001')
      );
    });
  });

  describe('handleAssistantSentence', () => {
    beforeEach(() => {
      // Set up a streaming message
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.currentStreamingMessageId = messageId;
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };
    });

    it('transforms protocol AssistantSentence to store MessageSentence format', () => {
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello there!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      expect(mockStore.addSentence).toHaveBeenCalledTimes(1);
      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      expect(addedSentence.content).toBe('Hello there!');
      expect(addedSentence.sequence).toBe(0);
      expect(addedSentence.isComplete).toBe(true);
      expect(addedSentence.messageId).toBe(createMessageId('msg-001'));
    });

    it('updates message content with sentence text', () => {
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello there!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      const message = mockStore.messages[createMessageId('msg-001')];
      expect(message.content).toBe('Hello there!');
    });

    it('marks message complete when isFinal is true', () => {
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Goodbye!',
        isFinal: true,
      };

      handleAssistantSentence(sentence, mockStore as any);

      expect(mockStore.updateMessageStatus).toHaveBeenCalledWith(
        createMessageId('msg-001'),
        MessageStatus.Complete
      );
      expect(mockStore.setCurrentStreamingMessageId).toHaveBeenCalledWith(null);
    });

    it('concatenates multiple sentences with spaces', () => {
      const messageId = createMessageId('msg-001');

      const sentence1: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'First sentence.',
        isFinal: false,
      };

      const sentence2: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 1,
        text: 'Second sentence.',
        isFinal: false,
      };

      handleAssistantSentence(sentence1, mockStore as any);
      handleAssistantSentence(sentence2, mockStore as any);

      const message = mockStore.messages[messageId];
      expect(message.content).toBe('First sentence. Second sentence.');
    });

    it('uses provided sentence ID if available', () => {
      const sentence: AssistantSentence = {
        id: 'sent-custom',
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      expect(addedSentence.id).toBe(createSentenceId('sent-custom'));
    });

    it('generates sentence ID from message ID and sequence if not provided', () => {
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 5,
        text: 'Hello!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      expect(addedSentence.id).toBe(createSentenceId('msg-001_s5'));
    });

    it('warns when no current streaming message exists', () => {
      mockStore.currentStreamingMessageId = null;
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'AssistantSentence received without active streaming message'
      );
      expect(mockStore.addSentence).not.toHaveBeenCalled();

      consoleWarnSpy.mockRestore();
    });
  });

  describe('handleAudioChunk', () => {
    beforeEach(() => {
      // Set up a streaming message
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.currentStreamingMessageId = messageId;
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };
    });

    it('stores audio and associates with sentence (audio arrives first)', async () => {
      const audioData = new Uint8Array([1, 2, 3, 4, 5]);
      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: audioData,
      };

      // Mock audioManager.store to return an ID
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      // Audio arrives first
      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).toHaveBeenCalledWith(audioData, {
        durationMs: 100,
        sampleRate: 24000,
      });

      expect(mockStore.addAudioRef).toHaveBeenCalledTimes(1);
      const audioRef = mockStore.addAudioRef.mock.calls[0][0];
      expect(audioRef.id).toBe(createAudioRefId('audio-001'));
      expect(audioRef.sizeBytes).toBe(5);
      expect(audioRef.durationMs).toBe(100);
      expect(audioRef.sampleRate).toBe(24000);

      // Now sentence arrives
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello there!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      // Sentence should have audio reference
      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      expect(addedSentence.audioRefId).toBe(createAudioRefId('audio-001'));
    });

    it('stores audio and associates with sentence (sentence arrives first)', async () => {
      // Sentence arrives first
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello there!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      // Verify sentence has no audio yet
      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      expect(addedSentence.audioRefId).toBeUndefined();

      // Mock audioManager.store
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      // Now audio arrives
      const audioData = new Uint8Array([1, 2, 3, 4, 5]);
      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: audioData,
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      // Sentence should be updated with audio reference
      expect(mockStore.updateSentence).toHaveBeenCalledWith(
        createSentenceId('msg-001_s0'),
        { audioRefId: createAudioRefId('audio-001') }
      );
    });

    it('parses sample rate from various format strings', async () => {
      const testCases = [
        { format: 'pcm_s16le_24000', expectedRate: 24000 },
        { format: 'opus_48000', expectedRate: 48000 },
        { format: 'pcm_16000', expectedRate: 16000 },
        { format: '24000', expectedRate: 24000 },
      ];

      for (const { format, expectedRate } of testCases) {
        vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

        const audioChunk: AudioChunk = {
          conversationId: 'conv-001',
          format,
          sequence: 0,
          durationMs: 100,
          trackSid: 'track-001',
          data: new Uint8Array([1, 2, 3]),
        };

        await handleAudioChunk(audioChunk, mockStore as any);

        expect(audioManager.store).toHaveBeenCalledWith(expect.any(Uint8Array), {
          durationMs: 100,
          sampleRate: expectedRate,
        });

        vi.clearAllMocks();
      }
    });

    it('uses default sample rate for malformed format strings', async () => {
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'invalid_format',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: new Uint8Array([1, 2, 3]),
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).toHaveBeenCalledWith(expect.any(Uint8Array), {
        durationMs: 100,
        sampleRate: 24000, // Default fallback
      });
    });

    it('uses default sample rate when format is undefined', async () => {
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: undefined as any,
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: new Uint8Array([1, 2, 3]),
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).toHaveBeenCalledWith(expect.any(Uint8Array), {
        durationMs: 100,
        sampleRate: 24000,
      });
    });

    it('rejects invalid sample rates outside safe range', async () => {
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_123456', // Too high
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: new Uint8Array([1, 2, 3]),
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).toHaveBeenCalledWith(expect.any(Uint8Array), {
        durationMs: 100,
        sampleRate: 24000, // Falls back to default
      });
    });

    it('does nothing when data is missing', async () => {
      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: undefined,
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).not.toHaveBeenCalled();
      expect(mockStore.addAudioRef).not.toHaveBeenCalled();
    });

    it('does nothing when trackSid is missing', async () => {
      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: undefined,
        data: new Uint8Array([1, 2, 3]),
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(audioManager.store).not.toHaveBeenCalled();
      expect(mockStore.addAudioRef).not.toHaveBeenCalled();
    });

    it('handles errors from audioManager.store gracefully', async () => {
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      vi.mocked(audioManager.store).mockRejectedValue(new Error('Storage failed'));

      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: new Uint8Array([1, 2, 3]),
      };

      await handleAudioChunk(audioChunk, mockStore as any);

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to store audio chunk:',
        expect.any(Error)
      );

      consoleErrorSpy.mockRestore();
    });
  });

  describe('handleToolUseRequest', () => {
    it('maps ToolUseRequest to pending ToolCall', () => {
      const toolUseRequest: ToolUseRequest = {
        id: 'tool-001',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        toolName: 'calculator',
        parameters: { operation: 'add', a: 1, b: 2 },
        execution: 'server',
      };

      handleToolUseRequest(toolUseRequest, mockStore as any);

      expect(mockStore.addToolCall).toHaveBeenCalledTimes(1);
      const toolCall = mockStore.addToolCall.mock.calls[0][0];
      expect(toolCall.status).toBe('pending');
      expect(toolCall.id).toBe(createToolCallId('tool-001'));
      expect(toolCall.toolName).toBe('calculator');
      expect(toolCall.arguments).toEqual({ operation: 'add', a: 1, b: 2 });
      expect(toolCall.messageId).toBe(createMessageId('msg-001'));
      expect(toolCall.startTimeMs).toBeGreaterThan(0);
    });
  });

  describe('handleToolUseResult', () => {
    beforeEach(() => {
      // Add a pending tool call
      const toolCall: ToolCall = {
        status: 'pending',
        id: createToolCallId('tool-001'),
        toolName: 'calculator',
        arguments: { operation: 'add', a: 1, b: 2 },
        messageId: createMessageId('msg-001'),
        startTimeMs: Date.now() - 1000,
      };
      mockStore.toolCalls[toolCall.id] = toolCall;
    });

    it('maps successful ToolUseResult to success status', () => {
      const toolUseResult: ToolUseResult = {
        id: 'result-001',
        requestId: 'tool-001',
        conversationId: 'conv-001',
        success: true,
        result: { answer: 3 },
      };

      handleToolUseResult(toolUseResult, mockStore as any);

      expect(mockStore.updateToolCall).toHaveBeenCalledTimes(1);
      const [id, updatedToolCall] = mockStore.updateToolCall.mock.calls[0];
      expect(id).toBe(createToolCallId('tool-001'));
      expect(updatedToolCall.status).toBe('success');
      expect((updatedToolCall as any).resultContent).toBe(JSON.stringify({ answer: 3 }));
      expect((updatedToolCall as any).endTimeMs).toBeGreaterThan(0);
    });

    it('maps failed ToolUseResult to error status', () => {
      const toolUseResult: ToolUseResult = {
        id: 'result-001',
        requestId: 'tool-001',
        conversationId: 'conv-001',
        success: false,
        errorCode: 'TIMEOUT',
        errorMessage: 'Tool execution timed out',
      };

      handleToolUseResult(toolUseResult, mockStore as any);

      expect(mockStore.updateToolCall).toHaveBeenCalledTimes(1);
      const [id, updatedToolCall] = mockStore.updateToolCall.mock.calls[0];
      expect(id).toBe(createToolCallId('tool-001'));
      expect(updatedToolCall.status).toBe('error');
      expect((updatedToolCall as any).error).toBe('Tool execution timed out');
      expect((updatedToolCall as any).endTimeMs).toBeGreaterThan(0);
    });

    it('uses error code as fallback when error message is missing', () => {
      const toolUseResult: ToolUseResult = {
        id: 'result-001',
        requestId: 'tool-001',
        conversationId: 'conv-001',
        success: false,
        errorCode: 'UNKNOWN_ERROR',
      };

      handleToolUseResult(toolUseResult, mockStore as any);

      const [, updatedToolCall] = mockStore.updateToolCall.mock.calls[0];
      expect(updatedToolCall.status).toBe('error');
      expect((updatedToolCall as any).error).toBe('Error code: UNKNOWN_ERROR');
    });

    it('handles string results directly without JSON stringify', () => {
      const toolUseResult: ToolUseResult = {
        id: 'result-001',
        requestId: 'tool-001',
        conversationId: 'conv-001',
        success: true,
        result: 'Simple string result',
      };

      handleToolUseResult(toolUseResult, mockStore as any);

      const [, updatedToolCall] = mockStore.updateToolCall.mock.calls[0];
      expect(updatedToolCall.status).toBe('success');
      expect((updatedToolCall as any).resultContent).toBe('Simple string result');
    });

    it('warns when tool call does not exist', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const toolUseResult: ToolUseResult = {
        id: 'result-001',
        requestId: 'nonexistent-tool',
        conversationId: 'conv-001',
        success: true,
        result: 'test',
      };

      handleToolUseResult(toolUseResult, mockStore as any);

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'ToolUseResult received for unknown tool call: nonexistent-tool'
      );
      expect(mockStore.updateToolCall).not.toHaveBeenCalled();

      consoleWarnSpy.mockRestore();
    });
  });

  describe('handleReasoningStep', () => {
    beforeEach(() => {
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };
    });

    it('wraps reasoning content in XML tags with sequence attribute', () => {
      const reasoningStep: ReasoningStep = {
        id: 'reason-001',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        sequence: 0,
        content: 'Let me think about this problem...',
      };

      handleReasoningStep(reasoningStep, mockStore as any);

      const message = mockStore.messages[createMessageId('msg-001')];
      expect(message.content).toBe(
        '<reasoning data-sequence="0">Let me think about this problem...</reasoning>'
      );
    });

    it('appends reasoning blocks to existing content', () => {
      const messageId = createMessageId('msg-001');
      mockStore.messages[messageId].content = 'Some existing content';

      const reasoningStep: ReasoningStep = {
        id: 'reason-001',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        sequence: 1,
        content: 'Additional reasoning...',
      };

      handleReasoningStep(reasoningStep, mockStore as any);

      const message = mockStore.messages[messageId];
      expect(message.content).toBe(
        'Some existing content <reasoning data-sequence="1">Additional reasoning...</reasoning>'
      );
    });

    it('handles multiple reasoning blocks with different sequences', () => {
      const messageId = createMessageId('msg-001');

      const step1: ReasoningStep = {
        id: 'reason-001',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        sequence: 0,
        content: 'First thought',
      };

      const step2: ReasoningStep = {
        id: 'reason-002',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        sequence: 1,
        content: 'Second thought',
      };

      handleReasoningStep(step1, mockStore as any);
      handleReasoningStep(step2, mockStore as any);

      const message = mockStore.messages[messageId];
      expect(message.content).toBe(
        '<reasoning data-sequence="0">First thought</reasoning> <reasoning data-sequence="1">Second thought</reasoning>'
      );
    });

    it('warns when message does not exist', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const reasoningStep: ReasoningStep = {
        id: 'reason-001',
        messageId: 'nonexistent-msg',
        conversationId: 'conv-001',
        sequence: 0,
        content: 'Some reasoning',
      };

      handleReasoningStep(reasoningStep, mockStore as any);

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'ReasoningStep received for unknown message:',
        'nonexistent-msg'
      );

      consoleWarnSpy.mockRestore();
    });
  });

  describe('handleTranscription', () => {
    it('creates new user message from final transcription', () => {
      const transcription: Transcription = {
        id: 'trans-001',
        conversationId: 'conv-001',
        text: 'Hello, how are you?',
        final: true,
      };

      handleTranscription(transcription, mockStore as any);

      expect(mockStore.addMessage).toHaveBeenCalledTimes(1);
      const message = mockStore.addMessage.mock.calls[0][0];
      expect(message.id).toBe(createMessageId('trans-001'));
      expect(message.conversationId).toBe(createConversationId('conv-001'));
      expect(message.role).toBe('user');
      expect(message.content).toBe('Hello, how are you?');
      expect(message.status).toBe(MessageStatus.Complete);
    });

    it('creates streaming user message from interim transcription', () => {
      const transcription: Transcription = {
        id: 'trans-001',
        conversationId: 'conv-001',
        text: 'Hello, how...',
        final: false,
      };

      handleTranscription(transcription, mockStore as any);

      const message = mockStore.addMessage.mock.calls[0][0];
      expect(message.status).toBe(MessageStatus.Streaming);
    });

    it('updates existing transcription instead of creating duplicate', () => {
      const messageId = createMessageId('trans-001');
      const conversationId = createConversationId('conv-001');
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'user',
        content: 'Hello, how...',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const transcription: Transcription = {
        id: 'trans-001',
        conversationId: 'conv-001',
        text: 'Hello, how are you?',
        final: true,
      };

      handleTranscription(transcription, mockStore as any);

      // Should not create new message
      expect(mockStore.addMessage).not.toHaveBeenCalled();

      // Should update existing message
      const message = mockStore.messages[messageId];
      expect(message.content).toBe('Hello, how are you?');
      expect(mockStore.updateMessageStatus).toHaveBeenCalledWith(
        messageId,
        MessageStatus.Complete
      );
    });

    it('prevents duplicate final transcriptions by content (REST API race condition)', () => {
      const conversationId = createConversationId('conv-001');

      // Existing message loaded from REST API with different ID
      const existingMessageId = createMessageId('rest-msg-001');
      mockStore.messages[existingMessageId] = {
        id: existingMessageId,
        conversationId,
        role: 'user',
        content: 'Hello, how are you?',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // WebSocket sends same transcription with different ID
      const transcription: Transcription = {
        id: 'trans-001',
        conversationId: 'conv-001',
        text: 'Hello, how are you?',
        final: true,
      };

      handleTranscription(transcription, mockStore as any);

      // Should not create duplicate
      expect(mockStore.addMessage).not.toHaveBeenCalled();
    });

    it('allows interim transcriptions even if content matches (whitespace differs)', () => {
      const conversationId = createConversationId('conv-001');

      // Existing complete message
      mockStore.messages[createMessageId('msg-001')] = {
        id: createMessageId('msg-001'),
        conversationId,
        role: 'user',
        content: 'Hello',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Interim transcription with different whitespace
      const transcription: Transcription = {
        id: 'trans-001',
        conversationId: 'conv-001',
        text: ' Hello ',
        final: false,
      };

      handleTranscription(transcription, mockStore as any);

      // Should create message because it's interim (not final)
      expect(mockStore.addMessage).toHaveBeenCalledTimes(1);
    });
  });

  describe('handleMemoryTrace', () => {
    beforeEach(() => {
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: 'Some content',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };
    });

    it('adds memory trace to message', () => {
      const memoryTrace: ProtocolMemoryTrace = {
        id: 'trace-001',
        messageId: 'msg-001',
        conversationId: 'conv-001',
        memoryId: 'mem-001',
        content: 'User prefers concise answers',
        relevance: 0.95,
      };

      handleMemoryTrace(memoryTrace, mockStore as any);

      expect(mockStore.addMemoryTrace).toHaveBeenCalledTimes(1);
      const trace = mockStore.addMemoryTrace.mock.calls[0][0];
      expect(trace.id).toBe(createMemoryTraceId('trace-001'));
      expect(trace.messageId).toBe(createMessageId('msg-001'));
      expect(trace.content).toBe('User prefers concise answers');
      expect(trace.relevance).toBe(0.95);
      expect(trace.source).toBe('mem-001');
    });
  });

  describe('handleProtocolMessage (routing)', () => {
    it('routes StartAnswer to handleStartAnswer', () => {
      const envelope: Envelope = {
        stanzaId: 1,
        conversationId: 'conv-001',
        type: MessageType.StartAnswer,
        body: {
          id: 'msg-001',
          previousId: 'msg-000',
          conversationId: 'conv-001',
        } as StartAnswer,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.addMessage).toHaveBeenCalledTimes(1);
    });

    it('routes AssistantSentence to handleAssistantSentence', () => {
      // Set up streaming message first
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.currentStreamingMessageId = messageId;
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const envelope: Envelope = {
        stanzaId: 2,
        conversationId: 'conv-001',
        type: MessageType.AssistantSentence,
        body: {
          previousId: 'msg-000',
          conversationId: 'conv-001',
          sequence: 0,
          text: 'Hello!',
        } as AssistantSentence,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.addSentence).toHaveBeenCalledTimes(1);
    });

    it('routes ToolUseRequest to handleToolUseRequest', () => {
      const envelope: Envelope = {
        stanzaId: 3,
        conversationId: 'conv-001',
        type: MessageType.ToolUseRequest,
        body: {
          id: 'tool-001',
          messageId: 'msg-001',
          conversationId: 'conv-001',
          toolName: 'test',
          parameters: {},
          execution: 'server',
        } as ToolUseRequest,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.addToolCall).toHaveBeenCalledTimes(1);
    });

    it('routes ToolUseResult to handleToolUseResult', () => {
      // Set up pending tool call
      mockStore.toolCalls[createToolCallId('tool-001')] = {
        status: 'pending',
        id: createToolCallId('tool-001'),
        toolName: 'test',
        arguments: {},
        messageId: createMessageId('msg-001'),
        startTimeMs: Date.now(),
      };

      const envelope: Envelope = {
        stanzaId: 4,
        conversationId: 'conv-001',
        type: MessageType.ToolUseResult,
        body: {
          id: 'result-001',
          requestId: 'tool-001',
          conversationId: 'conv-001',
          success: true,
          result: 'done',
        } as ToolUseResult,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.updateToolCall).toHaveBeenCalledTimes(1);
    });

    it('routes ReasoningStep to handleReasoningStep', () => {
      // Set up message
      const messageId = createMessageId('msg-001');
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId: createConversationId('conv-001'),
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const envelope: Envelope = {
        stanzaId: 5,
        conversationId: 'conv-001',
        type: MessageType.ReasoningStep,
        body: {
          id: 'reason-001',
          messageId: 'msg-001',
          conversationId: 'conv-001',
          sequence: 0,
          content: 'Thinking...',
        } as ReasoningStep,
      };

      handleProtocolMessage(envelope);

      const message = mockStore.messages[messageId];
      expect(message.content).toContain('reasoning');
    });

    it('routes AudioChunk to handleAudioChunk', async () => {
      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      const envelope: Envelope = {
        stanzaId: 6,
        conversationId: 'conv-001',
        type: MessageType.AudioChunk,
        body: {
          conversationId: 'conv-001',
          format: 'pcm_s16le_24000',
          sequence: 0,
          durationMs: 100,
          trackSid: 'track-001',
          data: new Uint8Array([1, 2, 3]),
        } as AudioChunk,
      };

      handleProtocolMessage(envelope);

      // Wait a bit for async operation
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(audioManager.store).toHaveBeenCalled();
    });

    it('routes Transcription to handleTranscription', () => {
      const envelope: Envelope = {
        stanzaId: 7,
        conversationId: 'conv-001',
        type: MessageType.Transcription,
        body: {
          id: 'trans-001',
          conversationId: 'conv-001',
          text: 'Hello',
          final: true,
        } as Transcription,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.addMessage).toHaveBeenCalledTimes(1);
    });

    it('routes MemoryTrace to handleMemoryTrace', () => {
      // Set up message
      const messageId = createMessageId('msg-001');
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId: createConversationId('conv-001'),
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const envelope: Envelope = {
        stanzaId: 8,
        conversationId: 'conv-001',
        type: MessageType.MemoryTrace,
        body: {
          id: 'trace-001',
          messageId: 'msg-001',
          conversationId: 'conv-001',
          memoryId: 'mem-001',
          content: 'Relevant info',
          relevance: 0.8,
        } as ProtocolMemoryTrace,
      };

      handleProtocolMessage(envelope);

      expect(mockStore.addMemoryTrace).toHaveBeenCalledTimes(1);
    });

    it('ignores unknown message types', () => {
      const envelope: Envelope = {
        stanzaId: 99,
        conversationId: 'conv-001',
        type: 999 as MessageType, // Unknown type
        body: {},
      };

      // Should not throw
      expect(() => handleProtocolMessage(envelope)).not.toThrow();
    });
  });

  describe('resetAdapterState', () => {
    it('clears all conversation contexts', () => {
      // Simulate some adapter usage that creates contexts
      const messageId = createMessageId('msg-001');
      const conversationId = createConversationId('conv-001');
      mockStore.currentStreamingMessageId = messageId;
      mockStore.messages[messageId] = {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: '',
        status: MessageStatus.Streaming,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      vi.mocked(audioManager.store).mockResolvedValue('audio-001' as AudioRefId);

      // Create some audio/sentence association state
      const audioChunk: AudioChunk = {
        conversationId: 'conv-001',
        format: 'pcm_s16le_24000',
        sequence: 0,
        durationMs: 100,
        trackSid: 'track-001',
        data: new Uint8Array([1, 2, 3]),
      };

      handleAudioChunk(audioChunk, mockStore as any);

      // Reset adapter state
      resetAdapterState();

      // After reset, new sentence should not find the audio association
      const sentence: AssistantSentence = {
        previousId: 'msg-000',
        conversationId: 'conv-001',
        sequence: 0,
        text: 'Hello!',
        isFinal: false,
      };

      handleAssistantSentence(sentence, mockStore as any);

      const addedSentence = mockStore.addSentence.mock.calls[0][0];
      // Audio association should be gone after reset
      expect(addedSentence.audioRefId).toBeUndefined();
    });
  });
});
