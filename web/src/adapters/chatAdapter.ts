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

  if (!store.getMessage(conversationId, messageId)) {
    const message = createEmptyMessage(messageId, conversationId, 'assistant');
    message.status = 'streaming';
    store.addMessage(conversationId, message);
    store.setTipMessageId(conversationId, messageId);
    store.startStreaming(conversationId, messageId);
  }

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
  const convState = store.getConversationState(conversationId);
  if (!convState) return;

  const messages = convState.messages;
  let matchedMessageId: ReturnType<typeof createMessageId> | undefined;

  for (const [, m] of messages) {
    if (m.tool_calls.some((tc) => tc.id === msg.requestId)) {
      matchedMessageId = m.id;
      break;
    }
  }

  if (!matchedMessageId) return;

  if (msg.success) {
    store.updateToolCall(conversationId, matchedMessageId, msg.requestId, {
      status: 'success',
      result: msg.result,
    });
  } else {
    store.updateToolCall(conversationId, matchedMessageId, msg.requestId, {
      status: 'error',
      error: msg.error || 'Unknown error',
    });
  }
}

function handleMemoryTrace(msg: ProtocolMemoryTrace, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);

  if (!store.getMessage(conversationId, messageId)) {
    const message = createEmptyMessage(messageId, conversationId, 'assistant');
    message.status = 'streaming';
    store.addMessage(conversationId, message);
    store.setTipMessageId(conversationId, messageId);
    store.startStreaming(conversationId, messageId);
  }

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
    // Preserve reasoning blocks that were added during streaming
    // thinking-summary tags are transient progress indicators, intentionally discarded
    const reasoningTags = existing.content.match(/<reasoning[^>]*>[\s\S]*?<\/reasoning>\n?/g) || [];
    const newContent = reasoningTags.join('') + msg.content;

    store.updateMessage(conversationId, messageId, {
      content: newContent,
      status: 'completed',
      previous_id: previousId,
    });
    store.setTipMessageId(conversationId, messageId);

    const convState = store.getConversationState(conversationId);
    if (convState?.streamingMessageId === messageId) {
      store.finishStreaming(conversationId);
    }
    return;
  }

  const message = createEmptyMessage(messageId, conversationId, 'assistant');
  message.content = msg.content;
  message.status = 'completed';
  message.created_at = new Date().toISOString();
  message.previous_id = previousId;

  store.addMessage(conversationId, message);
  store.setTipMessageId(conversationId, messageId);

  const convState = store.getConversationState(conversationId);
  if (convState?.streamingMessageId) {
    store.finishStreaming(conversationId);
  }
}

function handleErrorMessage(msg: ProtocolErrorMessage, store: ChatStore, conversationId: ConversationId | null): void {
  console.error(`[Protocol Error] ${msg.code}: ${msg.message}`);

  const targetConvId = conversationId || store.activeConversationId;
  if (!targetConvId) return;

  const convState = store.getConversationState(targetConvId);
  const streamingId = convState?.streamingMessageId;
  if (!streamingId) return;

  const message = store.getMessage(targetConvId, streamingId);
  if (message) {
    store.updateMessage(targetConvId, streamingId, {
      status: 'error',
      content: message.content + `\n\n[Error: ${msg.message}]`,
    });
    store.finishStreaming(targetConvId);
  }
}

function handleReasoningStep(msg: ReasoningStep, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);
  let message = store.getMessage(conversationId, messageId);

  if (!message) {
    const newMessage = createEmptyMessage(messageId, conversationId, 'assistant');
    newMessage.status = 'streaming';
    store.addMessage(conversationId, newMessage);
    store.setTipMessageId(conversationId, messageId);
    store.startStreaming(conversationId, messageId);
    message = newMessage;
  }

  const reasoningTag = `<reasoning data-sequence="${msg.sequence}" data-id="${msg.id}">${msg.content}</reasoning>`;
  store.appendContent(conversationId, messageId, '\n' + reasoningTag);
}

function handleThinkingSummary(msg: ThinkingSummary, store: ChatStore): void {
  const messageId = createMessageId(msg.messageId);
  const conversationId = createConversationId(msg.conversationId);

  if (!store.getMessage(conversationId, messageId)) {
    const newMessage = createEmptyMessage(messageId, conversationId, 'assistant');
    newMessage.status = 'streaming';
    store.addMessage(conversationId, newMessage);
    store.setTipMessageId(conversationId, messageId);
    store.startStreaming(conversationId, messageId);
  }

  const currentMessage = store.getMessage(conversationId, messageId);
  if (!currentMessage) return;

  const progressAttr = msg.progress !== undefined && msg.progress > 0 ? ` data-progress="${msg.progress}"` : '';
  const summaryTag = `<thinking-summary data-id="${msg.id}"${progressAttr}>${msg.content}</thinking-summary>\n`;
  const contentWithoutThinking = currentMessage.content.replace(/<thinking-summary[^>]*>[\s\S]*?<\/thinking-summary>\n?/g, '');
  store.updateMessage(conversationId, messageId, {
    content: summaryTag + contentWithoutThinking,
  });
}

function handleBranchUpdate(_msg: BranchUpdate, _store: ChatStore): void {
  // BranchUpdate notifies that a new sibling was created - UI can subscribe to this for branch navigation
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
