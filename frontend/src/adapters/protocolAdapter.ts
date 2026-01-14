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
  FeedbackConfirmation,
  NoteConfirmation,
  MemoryConfirmation,
  ServerInfo,
  SessionStats,
  EliteOptions,
  Feedback,
  UserNote,
  MemoryAction,
  DimensionPreference,
  EliteSelect,
  ControlStop,
  ControlVariation,
  StopType,
  VariationType,
} from '../types/protocol';
import {
  NormalizedMessage,
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
  SentenceId,
} from '../types/streaming';
import { useConversationStore } from '../stores/conversationStore';
import { audioManager } from '../utils/audioManager';
import { useFeedbackStore, type VotableType } from '../stores/feedbackStore';
import { useServerInfoStore, type ConnectionStatus, type MCPServerStatus } from '../stores/serverInfoStore';
import { useDimensionStore } from '../stores/dimensionStore';
import { messageRepository } from '../db/repository';

/**
 * Protocol adapter that transforms protocol messages into ConversationStore operations.
 * Handles the mapping between wire protocol (Envelope) and normalized store structure.
 */

// Cleanup timeout for stale entries (5 minutes)
const CLEANUP_TIMEOUT_MS = 5 * 60 * 1000;

// Stanza ID generation to prevent collisions within the same millisecond
let stanzaCounter = 0;
function generateStanzaId(): number {
  // Combine timestamp with counter to ensure uniqueness even within same millisecond
  // Use modulo to keep counter reasonable, timestamp changes every ms anyway
  stanzaCounter = (stanzaCounter + 1) % 1000;
  return Date.now() * 1000 + stanzaCounter;
}

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
      // TODO: Consider tracking failed audio chunks for retry or notification
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

    case MessageType.FeedbackConfirmation:
      handleFeedbackConfirmation(envelope.body as FeedbackConfirmation);
      break;

    case MessageType.NoteConfirmation:
      handleNoteConfirmation(envelope.body as NoteConfirmation);
      break;

    case MessageType.MemoryConfirmation:
      handleMemoryConfirmation(envelope.body as MemoryConfirmation);
      break;

    case MessageType.ServerInfo:
      handleServerInfo(envelope.body as ServerInfo);
      break;

    case MessageType.SessionStats:
      handleSessionStats(envelope.body as SessionStats);
      break;

    case MessageType.EliteOptions:
      handleEliteOptions(envelope.body as EliteOptions);
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

  const message: NormalizedMessage = {
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
    // Each sentence is complete when it arrives (not streamed character-by-character).
    // msg.isFinal (checked separately below) indicates the last sentence of the response.
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
  const sentences = store.getMessageSentences(currentMessageId);
  const content = sentences.map((s) => s.content).join(' ');
  store.updateMessageContent(currentMessageId, content);

  // If sentence is final and this is the last sentence, mark message as complete
  if (msg.isFinal) {
    store.updateMessageStatus(currentMessageId, MessageStatus.Complete);
    store.setCurrentStreamingMessageId(null);

    // Persist complete message to SQLite
    messageRepository.upsert({
      id: currentMessageId,
      conversation_id: message.conversationId,
      sequence_number: sentences.length,
      role: 'assistant',
      contents: content,
      local_id: currentMessageId,
      server_id: undefined,
      sync_status: 'synced',
      retry_count: 0,
      created_at: message.createdAt.toISOString(),
      updated_at: new Date().toISOString(),
    });
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
 * Includes sequence number and ID as data attributes for proper ordering and voting support.
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

  // Wrap reasoning content in <reasoning> tags for UI parsing
  const reasoningBlock = `<reasoning data-sequence="${msg.sequence}" data-id="${msg.id}">${msg.content}</reasoning>`;

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
      // Try to find the sentence by sequence number to get its actual ID
      // (which may be explicit or sequence-based)
      const sentences = store.getMessageSentences(currentMessageId);
      const matchingSentence = sentences.find((s) => s.sequence === msg.sequence);

      let sentenceId: SentenceId;
      if (matchingSentence) {
        // Use the ID of the existing sentence
        sentenceId = matchingSentence.id;
      } else {
        // Fallback to sequence-based ID (for when audio arrives first)
        sentenceId = createSentenceId(`${currentMessageId}_s${msg.sequence}`);
      }

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

    // Helper function to normalize content for comparison
    const normalizeContent = (content: string): string => {
      // Collapse multiple whitespace characters (spaces, tabs, newlines) to single space
      // and trim leading/trailing whitespace
      return content.replace(/\s+/g, ' ').trim();
    };

    const normalizedText = normalizeContent(msg.text);
    const currentTime = Date.now();
    const TIME_WINDOW_MS = 5000; // 5 seconds

    const duplicateByContent = allMessages.find((m) => {
      // Primary check: same message ID (most reliable)
      if (m.id === messageId) {
        return true;
      }

      // Secondary check: content-based with time window
      if (
        m.role === 'user' &&
        m.conversationId === conversationId &&
        normalizeContent(m.content) === normalizedText
      ) {
        // Only consider it a duplicate if it was created within the time window
        const messageAge = currentTime - m.createdAt.getTime();
        return messageAge <= TIME_WINDOW_MS;
      }

      return false;
    });

    if (duplicateByContent) {
      // Message already exists from REST API or recent duplicate, skip creating duplicate
      return;
    }
  }

  // Create new user message from transcription
  const message: NormalizedMessage = {
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
 * Handle FeedbackConfirmation: Update feedback store with confirmed vote and aggregates
 */
export function handleFeedbackConfirmation(msg: FeedbackConfirmation): void {
  const feedbackStore = useFeedbackStore.getState();

  // Update aggregates with server-confirmed data
  feedbackStore.setAggregates(
    msg.targetType as VotableType,
    msg.targetId,
    {
      upvotes: msg.aggregates.upvotes,
      downvotes: msg.aggregates.downvotes,
      special: msg.aggregates.specialVotes,
    }
  );
}

/**
 * Handle NoteConfirmation: Update notes store when note is confirmed
 */
export function handleNoteConfirmation(msg: NoteConfirmation): void {
  // Note is already in the store from the send operation
  // Could add additional confirmation logic here if needed
  if (!msg.success) {
    console.warn(`Note confirmation failed for note ${msg.noteId}`);
  }
}

/**
 * Handle MemoryConfirmation: Update memory store when action is confirmed
 */
export function handleMemoryConfirmation(msg: MemoryConfirmation): void {
  // Memory operations are already applied optimistically
  // Could add additional confirmation logic here if needed
  if (!msg.success) {
    console.warn(`Memory action ${msg.action} failed for memory ${msg.memoryId}`);
  }
}

/**
 * Handle ServerInfo: Update server info store with connection, model, and MCP server data
 */
export function handleServerInfo(msg: ServerInfo): void {
  const serverInfoStore = useServerInfoStore.getState();

  // Update connection status
  serverInfoStore.setConnectionStatus(msg.connection.status as ConnectionStatus);
  serverInfoStore.setLatency(msg.connection.latency);

  // Update model info
  serverInfoStore.setModelInfo({
    name: msg.model.name,
    provider: msg.model.provider,
  });

  // Update MCP servers
  serverInfoStore.setMCPServers(
    msg.mcpServers.map((server) => ({
      name: server.name,
      status: server.status as MCPServerStatus,
    }))
  );
}

/**
 * Handle SessionStats: Update server info store with session statistics
 */
export function handleSessionStats(msg: SessionStats): void {
  const serverInfoStore = useServerInfoStore.getState();

  serverInfoStore.setSessionStats({
    messageCount: msg.messageCount,
    toolCallCount: msg.toolCallCount,
    memoriesUsed: msg.memoriesUsed,
    sessionDuration: msg.sessionDuration,
  });
}

/**
 * Handle EliteOptions: Update dimension store with available elite solutions
 */
export function handleEliteOptions(msg: EliteOptions): void {
  const dimensionStore = useDimensionStore.getState();

  dimensionStore.updateElites(msg.elites, msg.currentEliteId);
}

/**
 * Clean up a specific conversation's context
 */
export function cleanupConversationContext(conversationId: string): void {
  const context = conversationContexts.get(conversationId);
  if (context) {
    // Clear all timers
    context.cleanupTimers.forEach((timer) => clearTimeout(timer));
    context.cleanupTimers.clear();
    context.sentenceAudioMap.clear();
    context.pendingSentences.clear();

    // Remove the context entirely
    conversationContexts.delete(conversationId);
  }
}

/**
 * Check if a conversation has an active context
 */
export function hasConversationContext(conversationId: string): boolean {
  return conversationContexts.has(conversationId);
}

/**
 * Get the number of active conversation contexts
 */
export function getConversationContextCount(): number {
  return conversationContexts.size;
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

/**
 * Called when WebSocket connection is lost to clean up streaming state.
 * This ensures no messages are left stuck in Streaming status.
 */
export function handleConnectionLost(): void {
  const store = useConversationStore.getState();
  const currentStreamingId = store.currentStreamingMessageId;

  if (currentStreamingId) {
    const message = store.messages[currentStreamingId];
    if (message && message.status === MessageStatus.Streaming) {
      console.warn(`Connection lost during streaming message: ${currentStreamingId}`);

      // Mark as error
      store.updateMessageStatus(currentStreamingId, MessageStatus.Error);

      // Clear streaming state
      store.setCurrentStreamingMessageId(null);
    }
  }

  // Also reset any adapter internal state
  resetAdapterState();
}

// ============================================================================
// Send Methods - Client to Server
// ============================================================================

// Reference to the active WebSocket or message sender (to be set by useLiveKit or similar)
let messageSender: ((envelope: Envelope) => void) | null = null;

/**
 * Set the message sender function for sending protocol messages.
 * @param sender - Function to send messages, or null when disconnecting
 */
export function setMessageSender(sender: ((envelope: Envelope) => void) | null): void {
  messageSender = sender;
}

/**
 * Send a feedback message to the server
 */
export function sendFeedback(feedback: Feedback): void {
  if (!messageSender) {
    console.warn('Cannot send feedback: no message sender available');
    return;
  }

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId: feedback.conversationId,
    type: MessageType.Feedback,
    body: feedback,
  };

  messageSender(envelope);
}

/**
 * Send a user note to the server
 */
export function sendUserNote(note: UserNote): void {
  if (!messageSender) {
    console.warn('Cannot send note: no message sender available');
    return;
  }

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId: '', // Notes don't have conversation context at envelope level
    type: MessageType.UserNote,
    body: note,
  };

  messageSender(envelope);
}

/**
 * Send a memory action to the server
 */
export function sendMemoryAction(action: MemoryAction): void {
  if (!messageSender) {
    console.warn('Cannot send memory action: no message sender available');
    return;
  }

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId: '', // Memory actions don't have conversation context at envelope level
    type: MessageType.MemoryAction,
    body: action,
  };

  messageSender(envelope);
}

/**
 * Send a dimension preference to the server
 */
export function sendDimensionPreference(pref: DimensionPreference): void {
  if (!messageSender) {
    console.warn('Cannot send dimension preference: no message sender available');
    return;
  }

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId: pref.conversationId,
    type: MessageType.DimensionPreference,
    body: pref,
  };

  messageSender(envelope);
}

/**
 * Send an elite selection to the server
 */
export function sendEliteSelect(select: EliteSelect): void {
  if (!messageSender) {
    console.warn('Cannot send elite select: no message sender available');
    return;
  }

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId: select.conversationId,
    type: MessageType.EliteSelect,
    body: select,
  };

  messageSender(envelope);
}

/**
 * Send a control stop to the server
 */
export function sendControlStop(conversationId: string, stopType: StopType = 'all'): void {
  if (!messageSender) {
    console.warn('Cannot send control stop: no message sender available');
    return;
  }

  const controlStop: ControlStop = {
    conversationId,
    stopType,
  };

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId,
    type: MessageType.ControlStop,
    body: controlStop,
  };

  messageSender(envelope);

  // Update local state to reflect stopped streaming
  const store = useConversationStore.getState();
  if (store.currentStreamingMessageId) {
    const message = store.messages[store.currentStreamingMessageId];
    if (message && message.conversationId === conversationId) {
      store.setCurrentStreamingMessageId(null);
    }
  }
}

/**
 * Send a control variation (regenerate) to the server
 */
export function sendControlVariation(
  conversationId: string,
  targetId: string,
  variationType: VariationType = 'regenerate',
  newContent?: string
): void {
  if (!messageSender) {
    console.warn('Cannot send control variation: no message sender available');
    return;
  }

  const controlVariation: ControlVariation = {
    conversationId,
    targetId,
    mode: variationType,
    newContent,
  };

  const envelope: Envelope = {
    stanzaId: generateStanzaId(),
    conversationId,
    type: MessageType.ControlVariation,
    body: controlVariation,
  };

  messageSender(envelope);
}
