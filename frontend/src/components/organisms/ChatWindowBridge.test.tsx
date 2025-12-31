import { render } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ChatWindowBridge from './ChatWindowBridge';
import { useConversationStore } from '../../stores/conversationStore';
import {
  createMessageId,
  createSentenceId,
  createToolCallId,
  createConversationId,
  MessageStatus,
} from '../../types/streaming';
import type { Message } from '../../types/models';

// Mock ChatWindow component to isolate testing
vi.mock('./ChatWindow', () => ({
  default: () => <div data-testid="chat-window">ChatWindow</div>,
}));

// Mock connectionStore since ChatWindowBridge uses it
vi.mock('../../stores/connectionStore', () => ({
  useConnectionStore: (selector: any) => {
    const mockStore = {
      setConnectionStatus: vi.fn(),
      setError: vi.fn(),
    };
    return selector ? selector(mockStore) : mockStore;
  },
  ConnectionStatus: {
    Disconnected: 'disconnected',
    Connecting: 'connecting',
    Connected: 'connected',
    Error: 'error',
  },
}));

/**
 * This test suite demonstrates the dual state management bug in ChatWindowBridge.
 *
 * THE BUG:
 * When the `messages` prop changes (e.g., from useMessages refresh), ChatWindowBridge's
 * useEffect calls loadConversation() which COMPLETELY REPLACES the entire store state,
 * wiping out any streaming state (sentences, toolCalls, audioRefs) that the protocolAdapter
 * had previously added during streaming.
 *
 * ROOT CAUSE:
 * - ChatWindowBridge syncs prop data → store via loadConversation()
 * - loadConversation() resets all normalized stores (sentences, toolCalls, etc.)
 * - convertToStreamingMessage() creates messages with empty sentenceIds/toolCallIds arrays
 * - Any streaming data added by protocolAdapter is lost
 *
 * EXPECTED FAILURES:
 * Tests 1 and 3 should FAIL, demonstrating that streaming state is not preserved.
 * Test 2 should PASS but demonstrates the wiping behavior.
 */
describe('ChatWindowBridge - Dual State Management Issue', () => {
  beforeEach(() => {
    // Reset the store to a clean state before each test
    useConversationStore.setState({
      messages: {},
      sentences: {},
      toolCalls: {},
      audioRefs: {},
      memoryTraces: {},
      currentStreamingMessageId: null,
      currentConversationId: null,
    });
  });

  it('should PRESERVE streaming state (sentenceIds, toolCallIds) when messages prop changes', () => {
    const conversationId = 'conv-123';
    const messageId = 'msg-456';

    // Step 1: Setup initial messages from props
    const initialMessages: Message[] = [
      {
        id: messageId,
        conversation_id: conversationId,
        sequence_number: 1,
        role: 'assistant',
        contents: 'Hello, how can I help you?',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    // Step 2: Render with initial messages - this triggers loadConversation
    const { rerender } = render(
      <ChatWindowBridge
        messages={initialMessages}
        loading={false}
        sending={false}
        onSendMessage={vi.fn()}
        conversationId={conversationId}
      />
    );

    // Step 3: Simulate protocolAdapter adding streaming state
    // This mimics what happens during streaming when the protocol adapter
    // adds sentences and tool calls to the store
    const normalizedMessageId = createMessageId(messageId);
    const sentenceId = createSentenceId('sent-789');
    const toolCallId = createToolCallId('tool-101');

    useConversationStore.getState().addSentence({
      id: sentenceId,
      messageId: normalizedMessageId,
      content: 'This is a streamed sentence',
      sequence: 1,
      isComplete: true,
    });

    useConversationStore.getState().addToolCall({
      status: 'pending',
      id: toolCallId,
      toolName: 'web_search',
      arguments: { query: 'test' },
      messageId: normalizedMessageId,
      startTimeMs: Date.now(),
    });

    // Verify that the streaming state was added successfully
    const storeAfterStreaming = useConversationStore.getState();
    const messageAfterStreaming = storeAfterStreaming.messages[normalizedMessageId];

    expect(messageAfterStreaming).toBeDefined();
    expect(messageAfterStreaming.sentenceIds).toContain(sentenceId);
    expect(messageAfterStreaming.toolCallIds).toContain(toolCallId);
    expect(storeAfterStreaming.sentences[sentenceId]).toBeDefined();
    expect(storeAfterStreaming.toolCalls[toolCallId]).toBeDefined();

    // Step 4: Trigger re-render with "updated" messages
    // This simulates what happens when useMessages refreshes from the database
    // In reality, the message content hasn't changed, but React re-renders
    // because the messages array reference changed
    const updatedMessages: Message[] = [
      {
        id: messageId,
        conversation_id: conversationId,
        sequence_number: 1,
        role: 'assistant',
        contents: 'Hello, how can I help you?', // Same content
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    rerender(
      <ChatWindowBridge
        messages={updatedMessages}
        loading={false}
        sending={false}
        onSendMessage={vi.fn()}
        conversationId={conversationId}
      />
    );

    // Step 5: Assert that streaming state is PRESERVED
    // This is the critical assertion that demonstrates the bug
    const storeAfterRerender = useConversationStore.getState();
    const messageAfterRerender = storeAfterRerender.messages[normalizedMessageId];

    // BUG: These assertions will FAIL because loadConversation() wipes the store
    // Expected behavior: sentenceIds and toolCallIds should be preserved
    // Actual behavior: They are reset to empty arrays by convertToStreamingMessage
    expect(messageAfterRerender).toBeDefined();
    expect(messageAfterRerender.sentenceIds).toContain(sentenceId);
    expect(messageAfterRerender.toolCallIds).toContain(toolCallId);

    // Also verify the related entities weren't deleted
    expect(storeAfterRerender.sentences[sentenceId]).toBeDefined();
    expect(storeAfterRerender.sentences[sentenceId].content).toBe('This is a streamed sentence');
    expect(storeAfterRerender.toolCalls[toolCallId]).toBeDefined();
    expect(storeAfterRerender.toolCalls[toolCallId].toolName).toBe('web_search');
  });

  it('should demonstrate that loadConversation wipes sentences and toolCalls stores', () => {
    const conversationId = 'conv-123';
    const messageId = 'msg-456';

    const messages: Message[] = [
      {
        id: messageId,
        conversation_id: conversationId,
        sequence_number: 1,
        role: 'assistant',
        contents: 'Test message',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];

    // Render once
    const { rerender } = render(
      <ChatWindowBridge
        messages={messages}
        loading={false}
        sending={false}
        onSendMessage={vi.fn()}
        conversationId={conversationId}
      />
    );

    // Add streaming data
    const normalizedMessageId = createMessageId(messageId);
    const sentenceId = createSentenceId('sent-123');

    useConversationStore.getState().addSentence({
      id: sentenceId,
      messageId: normalizedMessageId,
      content: 'Streaming sentence',
      sequence: 1,
      isComplete: false,
    });

    // Verify it was added
    expect(useConversationStore.getState().sentences[sentenceId]).toBeDefined();

    // Re-render with same messages (simulating useMessages refresh)
    rerender(
      <ChatWindowBridge
        messages={messages}
        loading={false}
        sending={false}
        onSendMessage={vi.fn()}
        conversationId={conversationId}
      />
    );

    // BUG: The sentences store is completely wiped
    const storeAfterRerender = useConversationStore.getState();

    // This assertion will FAIL - the sentence is gone
    expect(storeAfterRerender.sentences[sentenceId]).toBeDefined();
    expect(Object.keys(storeAfterRerender.sentences).length).toBeGreaterThan(0);
  });

  it('should demonstrate that mergeMessages preserves streaming state', () => {
    const conversationId = createConversationId('conv-123');
    const messageId = createMessageId('msg-456');

    // Manually populate the store with streaming data
    useConversationStore.getState().addMessage({
      id: messageId,
      conversationId,
      role: 'assistant',
      content: 'Test',
      status: MessageStatus.Streaming,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    });

    const sentenceId = createSentenceId('sent-123');
    const toolCallId = createToolCallId('tool-456');

    useConversationStore.getState().addSentence({
      id: sentenceId,
      messageId,
      content: 'Sentence',
      sequence: 1,
      isComplete: false,
    });

    useConversationStore.getState().addToolCall({
      status: 'pending',
      id: toolCallId,
      toolName: 'test_tool',
      arguments: {},
      messageId,
      startTimeMs: Date.now(),
    });

    // Verify state is populated
    let store = useConversationStore.getState();
    expect(store.messages[messageId]).toBeDefined();
    expect(store.sentences[sentenceId]).toBeDefined();
    expect(store.toolCalls[toolCallId]).toBeDefined();

    // Now call mergeMessages (what ChatWindowBridge now uses on re-render)
    useConversationStore.getState().mergeMessages(conversationId, [
      {
        id: messageId,
        conversationId,
        role: 'assistant',
        content: 'Test',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [], // Empty arrays from convertToStreamingMessage
        toolCallIds: [],
        memoryTraceIds: [],
      },
    ]);

    // FIX: All normalized stores are PRESERVED
    store = useConversationStore.getState();

    // Message exists and streaming arrays are preserved from existing state
    expect(store.messages[messageId]).toBeDefined();
    expect(store.messages[messageId].sentenceIds).toContain(sentenceId);
    expect(store.messages[messageId].toolCallIds).toContain(toolCallId);

    // Streaming state is preserved
    expect(Object.keys(store.sentences).length).toBeGreaterThan(0);
    expect(Object.keys(store.toolCalls).length).toBeGreaterThan(0);
    expect(store.sentences[sentenceId]).toBeDefined();
    expect(store.toolCalls[toolCallId]).toBeDefined();
  });
});
