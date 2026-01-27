import {
  Envelope,
  MessageType,
  StartAnswer,
  AssistantSentence,
  AssistantMessage as ProtocolAssistantMessage,
  ErrorMessage as ProtocolErrorMessage,
  ToolUseRequest,
  ToolUseResult,
  MemoryTrace as ProtocolMemoryTrace,
  ReasoningStep,
  ThinkingSummary,
  BranchUpdate,
  GenerationComplete,
} from '../types/protocol';
import { useChatStore } from '../stores/chatStore';
import {
  ConversationId,
  createMessageId,
  createConversationId,
  createToolCallId,
  createMemoryTraceId,
  createEmptyMessage,
} from '../types/chat';

type ChatStore = ReturnType<typeof useChatStore.getState>;

function ensureStreamingMessage(
  store: ChatStore,
  messageId: ReturnType<typeof createMessageId>,
  conversationId: ConversationId,
): void {
  if (store.getMessage(conversationId, messageId)) return;
  const message = createEmptyMessage(messageId, conversationId, 'assistant');
  message.status = 'streaming';
  store.addMessage(conversationId, message);
  store.setTipMessageId(conversationId, messageId);
  store.startStreaming(conversationId, messageId);
}

export function handleChatProtocolMessage(envelope: Envelope): void {
  const store = useChatStore.getState();
  const envelopeConversationId = envelope.conversationId
    ? createConversationId(envelope.conversationId)
    : null;

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

    case MessageType.MemoryTrace:
      handleMemoryTrace(envelope.body as ProtocolMemoryTrace, store);
      break;

    case MessageType.AssistantMessage:
      handleAssistantMessage(envelope.body as ProtocolAssistantMessage, store);
      break;

    case MessageType.ErrorMessage:
      handleErrorMessage(envelope.body as ProtocolErrorMessage, store, envelopeConversationId);
      break;

    case MessageType.ReasoningStep:
      handleReasoningStep(envelope.body as ReasoningStep, store);
      break;

    case MessageType.ThinkingSummary:
      handleThinkingSummary(envelope.body as ThinkingSummary, store);
      break;

    case MessageType.BranchUpdate:
      handleBranchUpdate(envelope.body as BranchUpdate, store);
      break;

    case MessageType.GenerationComplete:
      handleGenerationComplete(envelope.body as GenerationComplete, store);
      break;
  }
}

function handleStartAnswer(msg: StartAnswer, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);
  const previousId = msg.previousId ? createMessageId(msg.previousId) : undefined;

  if (previousId && !store.getMessage(conversationId, previousId)) {
    console.warn('[chatAdapter] StartAnswer: previous_id references missing message:', previousId);
  }

  const convState = store.getConversationState(conversationId);
  const optimisticMessageId = convState?.optimisticMessageId;

  // Replace optimistic message with the real server ID if one exists
  if (optimisticMessageId) {
    const optimistic = store.getMessage(conversationId, optimisticMessageId);
    if (optimistic) {
      store.updateMessage(conversationId, optimisticMessageId, {
        id: messageId,
        previous_id: previousId,
      });
      store.setOptimisticMessageId(conversationId, null);
      store.setTipMessageId(conversationId, messageId);
      store.startStreaming(conversationId, messageId);
      return;
    }
    // If not found (message was cleared by navigation), fall through to create normally
    store.setOptimisticMessageId(conversationId, null);
  }

  const message = createEmptyMessage(messageId, conversationId, 'assistant');
  message.previous_id = previousId;

  store.addMessage(conversationId, message);
  store.setTipMessageId(conversationId, messageId);
  store.startStreaming(conversationId, messageId);
}

function handleAssistantSentence(msg: AssistantSentence, store: ChatStore): void {
  const conversationId = createConversationId(msg.conversationId);
  const convState = store.getConversationState(conversationId);
  const streamingId = convState?.streamingMessageId;

  if (!streamingId) {
    console.warn(
      '[chatAdapter] AssistantSentence received but no streaming message active. Text will be lost:',
      msg.text.substring(0, 50) + (msg.text.length > 50 ? '...' : '')
    );
    return;
  }

  const currentMsg = store.getMessage(conversationId, streamingId);
  if (!currentMsg) {
    console.warn(
      '[chatAdapter] AssistantSentence: streamingMessageId points to non-existent message:',
      streamingId
    );
    return;
  }

  if (currentMsg.content) {
    store.appendContent(conversationId, streamingId, ' ' + msg.text);
  } else {
    store.appendContent(conversationId, streamingId, msg.text);
  }

  if (msg.isFinal) {
    store.finishStreaming(conversationId);
  }
}

function handleToolUseRequest(msg: ToolUseRequest, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);

  ensureStreamingMessage(store, messageId, conversationId);

  store.addToolCall(conversationId, messageId, {
    id: createToolCallId(msg.id),
    tool_name: msg.toolName,
    arguments: msg.arguments,
    status: 'pending',
    created_at: new Date().toISOString(),
  });
}

function handleToolUseResult(msg: ToolUseResult, store: ChatStore): void {
  const conversationId = createConversationId(msg.conversationId);
  const update = msg.success
    ? { status: 'success' as const, result: msg.result }
    : { status: 'error' as const, error: msg.error || 'Unknown error' };

  // Fast path: direct lookup when messageId is provided by the agent
  if (msg.messageId) {
    const messageId = createMessageId(msg.messageId);
    ensureStreamingMessage(store, messageId, conversationId);
    store.updateToolCall(conversationId, messageId, msg.requestId, update);
    return;
  }

  // Fallback: linear scan for backward compatibility (no messageId in payload)
  const convState = store.getConversationState(conversationId);
  if (!convState) return;

  for (const [, m] of convState.messages) {
    if (m.tool_calls.some((tc) => tc.id === msg.requestId)) {
      store.updateToolCall(conversationId, m.id, msg.requestId, update);
      return;
    }
  }
}

function handleMemoryTrace(msg: ProtocolMemoryTrace, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);

  ensureStreamingMessage(store, messageId, conversationId);

  store.addMemoryTrace(conversationId, messageId, {
    id: createMemoryTraceId(msg.id),
    memory_id: msg.memoryId,
    content: msg.content,
    relevance: msg.relevance,
  });
}

function handleAssistantMessage(msg: ProtocolAssistantMessage, store: ChatStore): void {
  const messageId = createMessageId(msg.id);
  const conversationId = createConversationId(msg.conversationId);
  const previousId = msg.previousId ? createMessageId(msg.previousId) : undefined;

  if (previousId && !store.getMessage(conversationId, previousId)) {
    console.warn('[chatAdapter] AssistantMessage: previous_id references missing message:', previousId);
  }

  const existing = store.getMessage(conversationId, messageId);
  if (existing) {
    // Drop tool calls that never received a result (pending/running) — they were likely
    // abandoned by the agent. Keep completed and errored ones.
    const resolvedToolCalls = existing.tool_calls.filter(
      (tc) => tc.status === 'success' || tc.status === 'error'
    );

    store.updateMessage(conversationId, messageId, {
      content: msg.content,
      status: 'completed',
      previous_id: previousId,
      tool_calls: resolvedToolCalls,
    });
    store.setTipMessageId(conversationId, messageId);
    store.finishStreaming(conversationId);
    return;
  }

  const message = createEmptyMessage(messageId, conversationId, 'assistant');
  message.content = msg.content;
  message.status = 'completed';
  message.created_at = new Date().toISOString();
  message.previous_id = previousId;

  store.addMessage(conversationId, message);
  store.setTipMessageId(conversationId, messageId);
  store.finishStreaming(conversationId);
}

function handleErrorMessage(msg: ProtocolErrorMessage, store: ChatStore, conversationId: ConversationId | null): void {
  console.error(`[Protocol Error] ${msg.code}: ${msg.message}`);

  const targetConvId = msg.conversationId
    ? createConversationId(msg.conversationId)
    : conversationId || store.activeConversationId;
  if (!targetConvId) return;

  const targetMessageId = msg.messageId
    ? createMessageId(msg.messageId)
    : store.getConversationState(targetConvId)?.streamingMessageId;
  if (!targetMessageId) return;

  const message = store.getMessage(targetConvId, targetMessageId);
  if (message) {
    store.updateMessage(targetConvId, targetMessageId, {
      status: 'error',
      content: message.content + `\n\n[Error: ${msg.message}]`,
    });
    store.finishStreaming(targetConvId);
  }
}

function handleReasoningStep(msg: ReasoningStep, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);
  ensureStreamingMessage(store, messageId, conversationId);

  store.addReasoningStep(conversationId, messageId, {
    id: msg.id,
    sequence: msg.sequence,
    content: msg.content,
  });
}

function handleThinkingSummary(msg: ThinkingSummary, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);

  ensureStreamingMessage(store, messageId, conversationId);

  store.setThinking(conversationId, messageId, {
    id: msg.id,
    content: msg.content,
    progress: msg.progress,
  });
}

function handleBranchUpdate(_msg: BranchUpdate, _store: ChatStore): void {
  // BranchUpdate notifies that a new sibling was created - UI can subscribe to this for branch navigation
}

function handleGenerationComplete(msg: GenerationComplete, store: ChatStore): void {
  const conversationId = createConversationId(msg.conversationId);

  if (msg.success) {
    store.finishStreaming(conversationId);
  } else {
    const convState = store.getConversationState(conversationId);
    const streamingId = convState?.streamingMessageId;
    if (streamingId) {
      const message = store.getMessage(conversationId, streamingId);
      if (message) {
        store.updateMessage(conversationId, streamingId, {
          status: 'error',
          content: message.content + `\n\n[Error: ${msg.error || 'generation failed'}]`,
        });
      }
    }
    store.finishStreaming(conversationId);
  }
}

export function handleChatConnectionLost(): void {
  const store = useChatStore.getState();
  const conversations = store.conversations;

  for (const [conversationId, convState] of conversations) {
    const streamingId = convState.streamingMessageId;
    if (!streamingId) continue;

    const message = store.getMessage(conversationId, streamingId);
    if (message && message.status === 'streaming') {
      store.updateMessage(conversationId, streamingId, { status: 'error' });
      store.finishStreaming(conversationId);
    }
  }
}
