/**
 * MessageList Branch Filtering Tests
 *
 * This test suite verifies that MessageList correctly shows only messages
 * in the "active branch" - the path from the conversation tip back to the root.
 *
 * When a user edits a message and creates a sibling, both the original and
 * the new sibling should NOT be displayed together. Instead, only the messages
 * in the active branch (determined by the most recent tip) should be shown.
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

// Mock the child components to isolate MessageList branch filtering behavior
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

describe('MessageList - Branch Filtering', () => {
  beforeEach(() => {
    useConversationStore.getState().clearConversation();
  });

  describe('Active branch selection', () => {
    it('should only show messages in the active branch when siblings exist', () => {
      /**
       * Tree structure:
       *   user-1 (root)
       *     ├── assistant-1a (original branch)
       *     └── assistant-1b (sibling - different branch)
       *
       * With assistant-1b being newer, the active branch should be:
       * user-1 -> assistant-1b
       *
       * assistant-1a should NOT be displayed.
       */
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      const userMessage: NormalizedMessage = {
        id: createMessageId('user-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'What is the weather?',
        status: MessageStatus.Complete,
        createdAt: baseTime,
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Original assistant response
      const assistantOriginal: NormalizedMessage = {
        id: createMessageId('assistant-1a'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'It is sunny.',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 100),
        previousId: createMessageId('user-1'),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Sibling assistant response (after edit/regenerate)
      const assistantSibling: NormalizedMessage = {
        id: createMessageId('assistant-1b'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'The weather is cloudy.',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 200), // Newer than original
        previousId: createMessageId('user-1'), // Same parent = siblings
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(userMessage);
      useConversationStore.getState().addMessage(assistantOriginal);
      useConversationStore.getState().addMessage(assistantSibling);

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);

      // Should show only 2 messages: user-1 and assistant-1b (the newer sibling)
      expect(items).toHaveLength(2);

      const actualIds = items.map(item => {
        const match = item.textContent?.match(/(user-1|assistant-1[ab])/);
        return match ? match[0] : '';
      });

      // Active branch should be user-1 -> assistant-1b
      expect(actualIds).toEqual(['user-1', 'assistant-1b']);

      // assistant-1a should NOT be shown
      expect(screen.queryByTestId('assistant-message-assistant-1a')).not.toBeInTheDocument();
    });

    it('should correctly filter when user edits a message and creates a new branch', () => {
      /**
       * Scenario: User sends message, gets response, then EDITS the user message.
       *
       * Tree structure:
       *   user-1 (original user message)
       *     └── assistant-1 (original response)
       *   user-1-edit (edited user message, sibling of user-1)
       *     └── assistant-2 (new response to edited message)
       *
       * Note: user-1-edit has no previousId (it's a root-level sibling)
       * assistant-2 is the tip (most recent leaf)
       *
       * Active branch should be: user-1-edit -> assistant-2
       */
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      // Original conversation
      const userOriginal: NormalizedMessage = {
        id: createMessageId('user-1'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Whats the weather',
        status: MessageStatus.Complete,
        createdAt: baseTime,
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const assistantOriginal: NormalizedMessage = {
        id: createMessageId('assistant-1'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'It is sunny.',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 100),
        previousId: createMessageId('user-1'),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      // Edited conversation (user edited their message)
      const userEdited: NormalizedMessage = {
        id: createMessageId('user-1-edit'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'What is the weather in New York?', // Edited content
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 200),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const assistantNew: NormalizedMessage = {
        id: createMessageId('assistant-2'),
        conversationId: createConversationId('conv-1'),
        role: 'assistant',
        content: 'In New York, it is currently 72F and partly cloudy.',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 300),
        previousId: createMessageId('user-1-edit'),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(userOriginal);
      useConversationStore.getState().addMessage(assistantOriginal);
      useConversationStore.getState().addMessage(userEdited);
      useConversationStore.getState().addMessage(assistantNew);

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);

      // Should show 2 messages: user-1-edit and assistant-2 (the new branch)
      expect(items).toHaveLength(2);

      const actualIds = items.map(item => {
        const match = item.textContent?.match(/(user-1-edit|user-1|assistant-[12])/);
        return match ? match[0] : '';
      });

      // Active branch: user-1-edit -> assistant-2
      expect(actualIds).toEqual(['user-1-edit', 'assistant-2']);
    });

    it('should show the complete chain from tip to root', () => {
      /**
       * Tree structure (deeper conversation):
       *   user-1
       *     └── assistant-1
       *           └── user-2
       *                 ├── assistant-2a (original)
       *                 └── assistant-2b (newer sibling)
       *
       * Active branch should be: user-1 -> assistant-1 -> user-2 -> assistant-2b
       */
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      const messages: NormalizedMessage[] = [
        {
          id: createMessageId('user-1'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Hello',
          status: MessageStatus.Complete,
          createdAt: baseTime,
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('assistant-1'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Hi there!',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 100),
          previousId: createMessageId('user-1'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('user-2'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'How are you?',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 200),
          previousId: createMessageId('assistant-1'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('assistant-2a'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'I am doing well.',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 300),
          previousId: createMessageId('user-2'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('assistant-2b'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'I am great, thanks for asking!', // Regenerated response
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 400), // Newer
          previousId: createMessageId('user-2'), // Same parent = sibling
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
      ];

      messages.forEach(msg => useConversationStore.getState().addMessage(msg));

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);

      // Should show 4 messages in the active branch
      expect(items).toHaveLength(4);

      const actualIds = items.map(item => {
        const match = item.textContent?.match(/(user-[12]|assistant-[12][ab]?)/);
        return match ? match[0] : '';
      });

      // Active branch: user-1 -> assistant-1 -> user-2 -> assistant-2b
      expect(actualIds).toEqual(['user-1', 'assistant-1', 'user-2', 'assistant-2b']);

      // assistant-2a should NOT be shown
      expect(screen.queryByTestId('assistant-message-assistant-2a')).not.toBeInTheDocument();
    });

    it('should handle multiple levels of branching', () => {
      /**
       * Complex tree with branches at multiple levels:
       *   user-1
       *     ├── assistant-1a (branch A)
       *     │     └── user-2a
       *     │           └── assistant-2a
       *     └── assistant-1b (branch B, newer)
       *           └── user-2b (tip)
       *
       * Active branch should follow the newest tip: user-1 -> assistant-1b -> user-2b
       */
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      const messages: NormalizedMessage[] = [
        // Root
        {
          id: createMessageId('user-1'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Start',
          status: MessageStatus.Complete,
          createdAt: baseTime,
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        // Branch A
        {
          id: createMessageId('assistant-1a'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Branch A response',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 100),
          previousId: createMessageId('user-1'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('user-2a'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Follow up A',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 200),
          previousId: createMessageId('assistant-1a'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('assistant-2a'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Branch A final',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 300),
          previousId: createMessageId('user-2a'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        // Branch B (newer, so this is the active branch)
        {
          id: createMessageId('assistant-1b'),
          conversationId: createConversationId('conv-1'),
          role: 'assistant',
          content: 'Branch B response',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 400),
          previousId: createMessageId('user-1'), // Sibling of assistant-1a
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
        {
          id: createMessageId('user-2b'),
          conversationId: createConversationId('conv-1'),
          role: 'user',
          content: 'Follow up B',
          status: MessageStatus.Complete,
          createdAt: new Date(baseTime.getTime() + 500), // Newest - this is the tip
          previousId: createMessageId('assistant-1b'),
          sentenceIds: [],
          toolCallIds: [],
          memoryTraceIds: [],
        },
      ];

      messages.forEach(msg => useConversationStore.getState().addMessage(msg));

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);

      // Active branch should have 3 messages
      expect(items).toHaveLength(3);

      const actualIds = items.map(item => {
        const match = item.textContent?.match(/(user-[12][ab]?|assistant-[12][ab]?)/);
        return match ? match[0] : '';
      });

      // Active branch: user-1 -> assistant-1b -> user-2b
      expect(actualIds).toEqual(['user-1', 'assistant-1b', 'user-2b']);
    });
  });

  describe('Edge cases', () => {
    it('should handle single message conversation', () => {
      const userMessage: NormalizedMessage = {
        id: createMessageId('user-only'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Just one message',
        status: MessageStatus.Complete,
        createdAt: new Date(),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(userMessage);

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);
      expect(items).toHaveLength(1);
      expect(items[0].textContent).toContain('user-only');
    });

    it('should handle empty conversation', () => {
      render(<MessageList />);

      // Should show the "No messages" placeholder
      expect(screen.getByText(/No messages yet/)).toBeInTheDocument();
    });

    it('should filter by current conversation ID', () => {
      const baseTime = new Date('2025-01-15T10:00:00.000Z');

      // Messages from two different conversations
      const msg1: NormalizedMessage = {
        id: createMessageId('conv1-user'),
        conversationId: createConversationId('conv-1'),
        role: 'user',
        content: 'Conv 1 message',
        status: MessageStatus.Complete,
        createdAt: baseTime,
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      const msg2: NormalizedMessage = {
        id: createMessageId('conv2-user'),
        conversationId: createConversationId('conv-2'),
        role: 'user',
        content: 'Conv 2 message',
        status: MessageStatus.Complete,
        createdAt: new Date(baseTime.getTime() + 100),
        sentenceIds: [],
        toolCallIds: [],
        memoryTraceIds: [],
      };

      useConversationStore.getState().addMessage(msg1);
      useConversationStore.getState().addMessage(msg2);
      useConversationStore.getState().setCurrentConversationId(createConversationId('conv-1'));

      render(<MessageList />);

      const items = screen.getAllByTestId(/^virtuoso-item-/);

      // Should only show message from conv-1
      expect(items).toHaveLength(1);
      expect(items[0].textContent).toContain('conv1-user');
    });
  });
});
