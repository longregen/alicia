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
  AudioRefId,
  createAudioRefId,
} from '../types/streaming';
import { useConversationStore } from '../stores/conversationStore';
import { audioManager } from '../utils/audioManager';

/**
 * Protocol adapter that transforms protocol messages into ConversationStore operations.
 * Handles the mapping between wire protocol (Envelope) and normalized store structure.
 */

// Cleanup timeout for stale entries (5 minutes)
const CLEANUP_TIMEOUT_MS = 5 * 60 * 1000;

/**
 * Per-conversation context to track sentence audio associations
 */
interface SentenceAudioContext {
  // Internal state to track sentence audio associations
  sentenceAudioMap: Map<string, AudioRefId>;
  // Track sentences waiting for audio (arrived before audio was stored)
  pendingSentences: Set<string>;
  // Cleanup timers for stale entries
  cleanupTimers: Map<string, NodeJS.Timeout>;
}

// Conversation-scoped contexts to prevent state leakage
const conversationContexts = new Map<string, SentenceAudioContext>();

/**
 * Get or create context for a conversation
 */
function getConversationContext(conversationId: string): SentenceAudioContext {
  let context = conversationContexts.get(conversationId);
  if (!context) {
    context = {
      sentenceAudioMap: new Map(),
      pendingSentences: new Set(),
      cleanupTimers: new Map(),
    };
    conversationContexts.set(conversationId, context);
  }
  return context;
}

/**
 * Parse sample rate from audio format string (e.g., "pcm_s16le_24000", "opus_48000")
 * Returns undefined if format cannot be parsed
 */
function parseSampleRateFromFormat(format: string | undefined): number | undefined {
  if (!format) return undefined;
  // Common patterns: "pcm_s16le_24000", "opus_48000", "24000", etc.
  const match = format.match(/(\d{4,6})(?:$|[^0-9])/);
  if (match) {
    const rate = parseInt(match[1], 10);
    // Sanity check: valid sample rates are typically 8000-48000
    if (rate >= 8000 && rate <= 96000) {
      return rate;
    }
  }
  return undefined;
}

/**
 * Main handler that routes protocol messages to appropriate handlers
 */
export function handleProtocolMessage(envelope: Envelope): void {
  const store = useConversationStore.getState();

  switch (envelope.type) {
    case MessageType.StartAnswer:
      handleStartAnswer(envelope.body as StartAnswer, store);
      break;

    case MessageType.AssistantSentence:
      handleAssistantSentence(envelope.body as AssistantSentence, store);
      break;

    case MessageType.ToolUseRequest:
      handleToolUseRequest(envelope.body as ToolUseRequest, store);
      break;

    case MessageType.ToolUseResult:
      handleToolUseResult(envelope.body as ToolUseResult, store);
      break;

    case MessageType.ReasoningStep:
      handleReasoningStep(envelope.body as ReasoningStep, store);
      break;

    case MessageType.AudioChunk:
      // Fire-and-forget: audio storage shouldn't block message processing
      handleAudioChunk(envelope.body as AudioChunk, store).catch((error) => {
        console.error('Failed to handle audio chunk:', error);
      });
      break;

    case MessageType.Transcription:
      handleTranscription(envelope.body as Transcription, store);
      break;

    case MessageType.MemoryTrace:
      handleMemoryTrace(envelope.body as ProtocolMemoryTrace, store);
      break;

    default:
      // Ignore other message types
      break;
  }
}

/**
 * Handle StartAnswer: Create new streaming assistant message
 */
export function handleStartAnswer(
  msg: StartAnswer,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const messageId = createMessageId(msg.id);
  const conversationId = createConversationId(msg.conversationId);

  const message: Message = {
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

  store.addMessage(message);
  store.setCurrentStreamingMessageId(messageId);
  store.setCurrentConversationId(conversationId);
}

/**
 * Handle AssistantSentence: Add sentence to streaming message
 */
export function handleAssistantSentence(
  msg: AssistantSentence,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const currentMessageId = store.currentStreamingMessageId;
  if (!currentMessageId) {
    console.warn('AssistantSentence received without active streaming message');
    return;
  }

  const sentenceId = msg.id
    ? createSentenceId(msg.id)
    : createSentenceId(`${currentMessageId}_s${msg.sequence}`);

  const sentence: MessageSentence = {
    id: sentenceId,
    messageId: currentMessageId,
    content: msg.text,
    sequence: msg.sequence,
    // Each AssistantSentence is complete when it arrives.
    // isFinal indicates if this is the LAST sentence of the response, not sentence completeness.
    isComplete: true,
  };

  // Get conversation context for this message
  const message = store.messages[currentMessageId];
  if (!message) {
    console.warn('AssistantSentence received for unknown message');
    return;
  }
  const context = getConversationContext(message.conversationId);

  // Check if there's audio associated with this sentence
  const audioRefId = context.sentenceAudioMap.get(sentenceId);
  if (audioRefId) {
    sentence.audioRefId = audioRefId;
    // Keep audio entry for now, cleanup will handle it
    clearCleanupTimer(sentenceId, context);
  } else {
    // No audio yet - register sentence as waiting for audio
    context.pendingSentences.add(sentenceId);
    scheduleCleanup(sentenceId, context);
  }

  store.addSentence(sentence);

  // Update message content to include this sentence
  if (message) {
    const sentences = store.getMessageSentences(currentMessageId);
    message.content = sentences.map((s) => s.content).join(' ');

    // If sentence is final and this is the last sentence, mark message as complete
    if (msg.isFinal) {
      store.updateMessageStatus(currentMessageId, MessageStatus.Complete);
      store.setCurrentStreamingMessageId(null);
    }
  }
}

/**
 * Handle ToolUseRequest: Add pending tool call
 */
export function handleToolUseRequest(
  msg: ToolUseRequest,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const toolCallId = createToolCallId(msg.id);
  const messageId = createMessageId(msg.messageId);

  const toolCall: ToolCall = {
    status: 'pending',
    id: toolCallId,
    toolName: msg.toolName,
    arguments: msg.parameters,
    messageId,
    startTimeMs: Date.now(),
  };

  store.addToolCall(toolCall);
}

/**
 * Handle ToolUseResult: Update tool call with result
 */
export function handleToolUseResult(
  msg: ToolUseResult,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const toolCallId = createToolCallId(msg.requestId);
  const existingToolCall = store.toolCalls[toolCallId];

  if (!existingToolCall) {
    console.warn(`ToolUseResult received for unknown tool call: ${msg.requestId}`);
    return;
  }

  const endTimeMs = Date.now();

  if (msg.success) {
    const updatedToolCall: ToolCall = {
      status: 'success',
      id: existingToolCall.id,
      toolName: existingToolCall.toolName,
      arguments: existingToolCall.arguments,
      messageId: existingToolCall.messageId,
      startTimeMs: existingToolCall.startTimeMs,
      endTimeMs,
      resultContent: typeof msg.result === 'string' ? msg.result : JSON.stringify(msg.result),
    };
    store.updateToolCall(toolCallId, updatedToolCall);
  } else {
    const updatedToolCall: ToolCall = {
      status: 'error',
      id: existingToolCall.id,
      toolName: existingToolCall.toolName,
      arguments: existingToolCall.arguments,
      messageId: existingToolCall.messageId,
      startTimeMs: existingToolCall.startTimeMs,
      endTimeMs,
      error: msg.errorMessage || `Error code: ${msg.errorCode}`,
    };
    store.updateToolCall(toolCallId, updatedToolCall);
  }
}

/**
 * Handle ReasoningStep: Wrap reasoning content in <reasoning> tags and append to message
 * ChatBubble will parse and render these as blue-bordered collapsible blocks.
 * Includes sequence number as data attribute for proper ordering when multiple blocks exist.
 */
export function handleReasoningStep(
  msg: ReasoningStep,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const messageId = createMessageId(msg.messageId);
  const message = store.messages[messageId];

  if (!message) {
    console.warn('ReasoningStep received for unknown message:', msg.messageId);
    return;
  }

  // Wrap reasoning content in <reasoning> tags with sequence for ChatBubble to parse and sort
  const reasoningBlock = `<reasoning data-sequence="${msg.sequence}">${msg.content}</reasoning>`;

  // Append to message content (with a space separator if content exists)
  message.content = message.content
    ? `${message.content} ${reasoningBlock}`
    : reasoningBlock;
}

/**
 * Handle AudioChunk: Store audio and associate with sentence
 */
export async function handleAudioChunk(
  msg: AudioChunk,
  store: ReturnType<typeof useConversationStore.getState>
): Promise<void> {
  if (!msg.data || !msg.trackSid) {
    return;
  }

  try {
    // Store the actual audio data in IndexedDB and get the AudioRefId
    // Parse sample rate from format string (e.g., "pcm_s16le_24000" or "opus_48000")
    const sampleRate = parseSampleRateFromFormat(msg.format) || 24000;
    const audioRefId = await audioManager.store(msg.data, {
      durationMs: msg.durationMs,
      sampleRate,
    });

    // Create AudioRef metadata and add to store
    const audioRef: AudioRef = {
      id: createAudioRefId(audioRefId),
      sizeBytes: msg.data.byteLength,
      durationMs: msg.durationMs,
      sampleRate,
    };
    store.addAudioRef(audioRef);

    // Get conversation context for sentence audio association
    const ctx = getConversationContext(msg.conversationId);

    // Associate with sentence if we have a current streaming message
    const currentMessageId = store.currentStreamingMessageId;
    if (currentMessageId) {
      const sentenceId = createSentenceId(`${currentMessageId}_s${msg.sequence}`);

      // Check if sentence already exists and is waiting for audio
      if (ctx.pendingSentences.has(sentenceId)) {
        // Sentence arrived first - update it directly
        store.updateSentence(sentenceId, { audioRefId: createAudioRefId(audioRefId) });
        ctx.pendingSentences.delete(sentenceId);
        clearCleanupTimer(sentenceId, ctx);
      } else {
        // Audio arrived first - store for when sentence arrives
        ctx.sentenceAudioMap.set(sentenceId, createAudioRefId(audioRefId));
        scheduleCleanup(sentenceId, ctx);
      }
    }
  } catch (error) {
    console.error('Failed to store audio chunk:', error);
  }
}

/**
 * Handle Transcription: Update user message or create new one
 */
export function handleTranscription(
  msg: Transcription,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const messageId = createMessageId(msg.id);
  const conversationId = createConversationId(msg.conversationId);

  // Check if message already exists by ID (for interim transcriptions)
  const existingMessage = store.messages[messageId];

  if (existingMessage) {
    // Update existing transcription
    existingMessage.content = msg.text;
    if (msg.final) {
      store.updateMessageStatus(messageId, MessageStatus.Complete);
    }
    return;
  }

  // Check for duplicate by content (race condition prevention)
  // This handles the case where REST API already loaded this message with a different ID
  if (msg.final) {
    const allMessages = Object.values(store.messages);
    const duplicateByContent = allMessages.find(
      (m) =>
        m.role === 'user' &&
        m.conversationId === conversationId &&
        m.content.trim() === msg.text.trim() &&
        m.status === MessageStatus.Complete
    );

    if (duplicateByContent) {
      // Message already exists from REST API, skip creating duplicate
      return;
    }
  }

  // Create new user message from transcription
  const message: Message = {
    id: messageId,
    conversationId,
    role: 'user',
    content: msg.text,
    status: msg.final ? MessageStatus.Complete : MessageStatus.Streaming,
    createdAt: new Date(),
    sentenceIds: [],
    toolCallIds: [],
    memoryTraceIds: [],
  };

  store.addMessage(message);
}

/**
 * Handle MemoryTrace: Add memory trace to message
 */
export function handleMemoryTrace(
  msg: ProtocolMemoryTrace,
  store: ReturnType<typeof useConversationStore.getState>
): void {
  const traceId = createMemoryTraceId(msg.id);
  const messageId = createMessageId(msg.messageId);

  const trace: MemoryTrace = {
    id: traceId,
    messageId,
    content: msg.content,
    relevance: msg.relevance,
    source: msg.memoryId,
  };

  store.addMemoryTrace(trace);
}

/**
 * Schedule cleanup for a sentence/audio entry
 */
function scheduleCleanup(sentenceId: string, context: SentenceAudioContext): void {
  clearCleanupTimer(sentenceId, context);
  const timer = setTimeout(() => {
    context.sentenceAudioMap.delete(sentenceId);
    context.pendingSentences.delete(sentenceId);
    context.cleanupTimers.delete(sentenceId);
  }, CLEANUP_TIMEOUT_MS);
  context.cleanupTimers.set(sentenceId, timer);
}

/**
 * Clear cleanup timer for a sentence
 */
function clearCleanupTimer(sentenceId: string, context: SentenceAudioContext): void {
  const timer = context.cleanupTimers.get(sentenceId);
  if (timer) {
    clearTimeout(timer);
    context.cleanupTimers.delete(sentenceId);
  }
}

/**
 * Reset adapter state (useful for conversation cleanup)
 */
export function resetAdapterState(): void {
  // Clean up all conversation contexts
  conversationContexts.forEach((context) => {
    context.cleanupTimers.forEach((timer) => clearTimeout(timer));
    context.cleanupTimers.clear();
    context.sentenceAudioMap.clear();
    context.pendingSentences.clear();
  });
  conversationContexts.clear();
}
