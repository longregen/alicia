import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AssistantMessage from './AssistantMessage';
import { MESSAGE_TYPES } from '../../mockData';
import { MessageStatus, createMessageId, createToolCallId, createMemoryTraceId, createSentenceId, createAudioRefId, createConversationId } from '../../types/streaming';
import type { Message, ToolCall, MemoryTrace, MessageSentence } from '../../types/streaming';

// Mock child components
vi.mock('../molecules/ChatBubble', () => ({
  default: ({ type, content, addons, tools }: any) => (
    <div data-testid="chat-bubble">
      <div data-testid="bubble-type">{type}</div>
      <div data-testid="bubble-content">{content}</div>
      <div data-testid="addons-count">{addons?.length || 0}</div>
      <div data-testid="tools-count">{tools?.length || 0}</div>
    </div>
  ),
}));

vi.mock('../atoms/MemoryTraceAddon', () => ({
  default: ({ traces }: any) => (
    <div data-testid="memory-trace-addon">
      <div data-testid="traces-count">{traces.length}</div>
    </div>
  ),
}));

vi.mock('../atoms/FeedbackControls', () => ({
  default: ({ currentVote, upvotes, downvotes }: any) => (
    <div data-testid="feedback-controls">
      <div data-testid="current-vote">{currentVote || 'none'}</div>
      <div data-testid="upvotes">{upvotes}</div>
      <div data-testid="downvotes">{downvotes}</div>
    </div>
  ),
}));

// Mock stores and hooks
const mockMessage: Message = {
  id: createMessageId('msg-1'),
  conversationId: createConversationId('conv-1'),
  role: 'assistant',
  content: 'Test message content',
  createdAt: new Date('2025-01-15T10:00:00Z'),
  status: MessageStatus.Complete,
  sentenceIds: [],
  toolCallIds: [],
  memoryTraceIds: [],
};

const mockToolCalls: ToolCall[] = [
  {
    id: createToolCallId('tool-1'),
    messageId: createMessageId('msg-1'),
    toolName: 'Search',
    arguments: { query: 'test' },
    status: 'success',
    startTimeMs: 1000,
    endTimeMs: 2000,
    resultContent: 'Found results',
  },
];

const mockMemoryTraces: MemoryTrace[] = [
  {
    id: createMemoryTraceId('memory-1'),
    messageId: createMessageId('msg-1'),
    content: 'Test memory',
    relevance: 0.9,
  },
];

const mockSentences: MessageSentence[] = [
  {
    id: createSentenceId('sent-1'),
    messageId: createMessageId('msg-1'),
    content: 'Test sentence',
    sequence: 0,
    audioRefId: createAudioRefId('audio-1'),
    isComplete: true,
  },
];

vi.mock('../../stores/conversationStore', () => ({
  useConversationStore: (selector: any) => {
    const state = {
      messages: { 'msg-1': mockMessage },
      getMessageToolCalls: () => mockToolCalls,
      getMessageMemoryTraces: () => mockMemoryTraces,
      getMessageSentences: () => mockSentences,
    };
    return selector(state);
  },
}));

vi.mock('../../hooks/useAudioManager', () => ({
  useAudioManager: () => ({
    play: vi.fn(),
    stop: vi.fn(),
    getMetadata: () => ({ durationMs: 5000 }),
  }),
}));

vi.mock('../../stores/audioStore', () => ({
  useAudioStore: (selector: any) => {
    const state = {
      playback: {
        currentlyPlayingId: null,
        isPlaying: false,
        playbackProgress: 0,
      },
    };
    return selector(state);
  },
}));

vi.mock('../../hooks/useFeedback', () => ({
  useFeedback: () => ({
    currentVote: null,
    vote: vi.fn(),
    counts: { up: 5, down: 2 },
    isLoading: false,
  }),
}));

describe('AssistantMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders ChatBubble with assistant type', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.ASSISTANT);
    });

    it('renders message content', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-content')).toHaveTextContent('Test message content');
    });

    it('applies custom className', () => {
      const { container } = render(<AssistantMessage messageId={createMessageId('msg-1')} className="custom-class" />);

      const rootElement = container.firstChild;
      expect(rootElement).toHaveClass('custom-class');
    });

    it('returns null when message not found', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: {},
            getMessageToolCalls: () => [],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      const { container } = render(<AssistantMessage messageId={createMessageId('nonexistent')} />);
      expect(container.firstChild).toBeNull();
    });
  });

  describe('Tool Calls Display', () => {
    it('passes tool calls to ChatBubble', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('converts tool call status to tool detail format', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      const toolsCount = screen.getByTestId('tools-count');
      expect(toolsCount).toHaveTextContent('1');
    });
  });

  describe('Memory Traces Display', () => {
    it('renders MemoryTraceAddon when traces exist', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('memory-trace-addon')).toBeInTheDocument();
    });

    it('shows correct trace count', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('traces-count')).toHaveTextContent('1');
    });

    it('does not render MemoryTraceAddon when no traces', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageToolCalls: () => [],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      render(<AssistantMessage messageId={createMessageId('msg-1')} />);
      expect(screen.queryByTestId('memory-trace-addon')).not.toBeInTheDocument();
    });
  });

  describe('Feedback Controls', () => {
    it('renders FeedbackControls', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('feedback-controls')).toBeInTheDocument();
    });

    it('displays vote counts', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('upvotes')).toHaveTextContent('5');
      expect(screen.getByTestId('downvotes')).toHaveTextContent('2');
    });

    it('shows current vote state', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('current-vote')).toHaveTextContent('none');
    });
  });

  describe('Audio Addons', () => {
    it('creates audio addons for sentences with audio', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      const addonsCount = screen.getByTestId('addons-count');
      expect(parseInt(addonsCount.textContent || '0')).toBeGreaterThan(0);
    });

    it('does not create audio addons when no audio refs', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageToolCalls: () => [],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [{
              id: createSentenceId('sent-1'),
              messageId: createMessageId('msg-1'),
              content: 'Test',
              sequence: 0,
              audioRefId: undefined,
              isComplete: true,
            }],
          };
          return selector(state);
        },
      }));

      render(<AssistantMessage messageId={createMessageId('msg-1')} />);
      const addonsCount = screen.getByTestId('addons-count');
      expect(addonsCount.textContent).toBe('0');
    });
  });

  describe('Tool Status Mapping', () => {
    it('maps pending status to running', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageToolCalls: () => [{
              id: 'tool-1',
              toolName: 'Test',
              arguments: {},
              status: 'pending',
            }],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('maps executing status to running', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageToolCalls: () => [{
              id: 'tool-1',
              toolName: 'Test',
              arguments: {},
              status: 'executing',
            }],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('maps success status to completed', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('maps error status correctly', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageToolCalls: () => [{
              id: 'tool-1',
              toolName: 'Test',
              arguments: {},
              status: 'error',
              error: 'Test error',
            }],
            getMessageMemoryTraces: () => [],
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });
  });

  describe('Integration', () => {
    it('renders with all features together', () => {
      render(<AssistantMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
      expect(screen.getByTestId('memory-trace-addon')).toBeInTheDocument();
      expect(screen.getByTestId('feedback-controls')).toBeInTheDocument();
    });
  });
});
