/**
 * MessageList Sort Order Tests
 *
 * This test suite demonstrates potential sort order issues in MessageList.tsx
 * when messages are sorted purely by createdAt timestamp.
 *
 * Issue: During streaming or with clock skew between client/server, timestamps
 * may not reflect the true conversation order. For example:
 * - User message sent at T
 * - Assistant response created at T-1ms (due to clock skew)
 * - Result: Assistant message appears before user message (WRONG)
 *
 * The first test in this suite is EXPECTED TO FAIL, demonstrating the bug.
 * Other tests show various edge cases where timestamp-only sorting fails.
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import MessageList from './MessageList';
import { useConversationStore } from '../../stores/conversationStore';
import {
  createMessageId,
  createConversationId,
  MessageStatus,
  type NormalizedMessage,
} from '../../types/streaming';

// Mock the child components to isolate MessageList sorting behavior
vi.mock('./UserMessage', () => ({
  default: ({ messageId }: any) => (
    <div data-testid={`user-message-${messageId}`}>User: {messageId}</div>
  ),
}));

vi.mock('./AssistantMessage', () => ({
  default: ({ messageId }: any) => (
    <div data-testid={`assistant-message-${messageId}`}>Assistant: {messageId}</div>
  ),
}));

vi.mock('./StreamingMessage', () => ({
  default: () => <div data-testid="streaming-message">Streaming...</div>,
}));

// Mock react-virtuoso to render all items without virtualization
vi.mock('react-virtuoso', () => ({
  Virtuoso: ({ totalCount, itemContent }: any) => (
    <div data-testid="virtuoso-list">
      {Array.from({ length: totalCount }, (_, index) => (
        <div key={index} data-testid={`virtuoso-item-${index}`}>
          {itemContent(index)}
        </div>
      ))}
    </div>
  ),
}));

describe('MessageList - Sort Order', () => {
  beforeEach(() => {
    // Reset store state before each test
    useConversationStore.getState().clearConversation();
  });

  describe('Clock skew scenarios', () => {
    it('should maintain correct conversation order when assistant message has earlier timestamp due to clock skew', () => {
      // Simulate a scenario where:
      // 1. User sends a message at time T
      // 2. Assistant response arrives with timestamp T-1ms (due to clock skew or server/client time difference)
      // Expected: User message should appear BEFORE assistant message regardless of timestamps

      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      // User message created at T
      const userMessage: NormalizedMessage = {
        id: createMessageId('user-msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'What is the weather?',
        status: MessageStatus.Complete,
        createdAt: baseTime,
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Assistant message created at T-1ms (clock skew simulation)
      // previousId links it to the user message
      const assistantMessage: NormalizedMessage = {
        id: createMessageId('assistant-msg-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'The weather is sunny.',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() - 1), // 1ms earlier
        previousId: createMessageId('user-msg-1'), // Links to user message
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Add messages to store
      useConversationStore.getState().addMessage(userMessage);
      useConversationStore.getState().addMessage(assistantMessage);

      // Render MessageList
      render(<MessageList />);

      // Get all virtuoso items
      const items = screen.getAllByTestId(/^virtuoso-item-/);
      expect(items).toHaveLength(2);

      // Verify order: user message should be FIRST (index 0), assistant SECOND (index 1)
      // This is the EXPECTED behavior for conversation flow
      const firstItem = screen.getByTestId('virtuoso-item-0');
      const secondItem = screen.getByTestId('virtuoso-item-1');

      // With previousId-based sorting, conversation order is preserved
      expect(firstItem.textContent).toContain('user-msg-1');
      expect(secondItem.textContent).toContain('assistant-msg-1');
    });

    it('should handle messages without previousId by falling back to timestamp sorting', () => {
      // When messages don't have previousId links, fall back to timestamp ordering
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      const userMessage: NormalizedMessage = {
        id: createMessageId('user-msg-2'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Hello',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 100), // T+100ms
        // No previousId
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const assistantMessage: NormalizedMessage = {
        id: createMessageId('assistant-msg-2'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Hi there',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 50), // T+50ms (earlier than user)
        // No previousId
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(userMessage);
      useConversationStore.getState().addMessage(assistantMessage);

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);
      expect(items).toHaveLength(2);

      // Get the actual order from the rendered list
      const firstItem = screen.getByTestId('virtuoso-item-0');
      const secondItem = screen.getByTestId('virtuoso-item-1');

      // Without previousId, falls back to timestamp sorting:
      // - assistant (T+50ms) will be at index 0
      // - user (T+100ms) will be at index 1
      expect(firstItem.textContent).toContain('assistant-msg-2');
      expect(secondItem.textContent).toContain('user-msg-2');
    });

    it('should handle rapid back-and-forth messages with slight timestamp drift', () => {
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      // Simulate a conversation where timestamps drift slightly due to network/clock issues
      const messages: NormalizedMessage[] = [
        {
          id: createMessageId('msg-1'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'First user message',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 0),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('msg-2'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'First assistant response',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 5), // Should be after msg-1
          previousId: createMessageId('msg-1'), // Links to msg-1
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('msg-3'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Second user message',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 3), // Clock skew: earlier than msg-2!
          previousId: createMessageId('msg-2'), // Links to msg-2
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('msg-4'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Second assistant response',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 10),
          previousId: createMessageId('msg-3'), // Links to msg-3
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
      ];

      messages.forEach(msg => useConversationStore.getState().addMessage(msg));

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);
      expect(items).toHaveLength(4);

      // With previousId-based sorting, conversation order is preserved despite clock skew
      const actualOrder = items.map(item => {
        const match = item.textContent?.match(/msg-\d+/);
        return match ? match[0] : '';
      });

      // CORRECT order maintained via previousId chain
      expect(actualOrder).toEqual(['msg-1', 'msg-2', 'msg-3', 'msg-4']);
    });

    it('should handle streaming message arrival before user message confirmation', () => {
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      // Scenario: User sends message, streaming starts immediately, but user message
      // confirmation arrives slightly later due to network latency

      const assistantMessage: NormalizedMessage = {
        id: createMessageId('assistant-streaming'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'Streaming response...',
        status: MessageStatus.Streaming,
        createdAt: new Date(baseTime.getTime() + 10), // Server processes fast
        previousId: createMessageId('user-delayed'), // Links to user message
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const userMessage: NormalizedMessage = {
        id: createMessageId('user-delayed'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'User question',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 50), // Arrives later due to network
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Add in the order they arrive at the client
      useConversationStore.getState().addMessage(assistantMessage);
      useConversationStore.getState().addMessage(userMessage);

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);
      expect(items).toHaveLength(2);

      // With previousId, conversation order is maintained
      const firstItem = screen.getByTestId('virtuoso-item-0');
      const secondItem = screen.getByTestId('virtuoso-item-1');

      // User message appears first due to previousId chain
      expect(firstItem.textContent).toContain('user-delayed');
      expect(secondItem.textContent).toContain('assistant-streaming');
    });
  });

  describe('Normal scenarios (should pass)', () => {
    it('should correctly sort messages with proper sequential timestamps', () => {
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      const messages: NormalizedMessage[] = [
        {
          id: createMessageId('msg-a'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'First',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 0),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('msg-b'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Second',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 100),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('msg-c'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Third',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 200),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
      ];

      messages.forEach(msg => useConversationStore.getState().addMessage(msg));

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);
      const actualOrder = items.map(item => {
        const match = item.textContent?.match(/msg-[a-c]/);
        return match ? match[0] : '';
      });

      // This should work correctly with timestamp sorting
      expect(actualOrder).toEqual(['msg-a', 'msg-b', 'msg-c']);
    });
  });
});
