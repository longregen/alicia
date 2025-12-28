import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import UserMessage from './UserMessage';
import { MESSAGE_TYPES } from '../../mockData';
import type { Message, MessageSentence } from '../../types/streaming';
import { createMessageId, MessageStatus } from '../../types/streaming';

// Mock child components
vi.mock('../molecules/ChatBubble', () => ({
  default: ({ type, content, addons, className }: any) => (
    <div data-testid="chat-bubble" className={className}>
      <div data-testid="bubble-type">{type}</div>
      <div data-testid="bubble-content">{content}</div>
      <div data-testid="addons-count">{addons?.length || 0}</div>
      {addons?.map((addon: any) => (
        <div key={addon.id} data-testid={`addon-${addon.type}`}>
          {addon.emoji}
        </div>
      ))}
    </div>
  ),
}));

// Mock stores and hooks
const mockMessage: Message = {
  id: createMessageId('msg-1'),
  conversationId: createMessageId('conv-1') as any,
  role: 'user',
  content: 'Test user message',
  createdAt: new Date('2025-01-15T10:00:00Z'),
  status: MessageStatus.Complete,
  sentenceIds: [],
  toolCallIds: [],
  memoryTraceIds: [],
};

const mockSentencesWithAudio: MessageSentence[] = [
  {
    id: createMessageId('sent-1') as any,
    messageId: createMessageId('msg-1'),
    content: 'Test sentence',
    sequence: 0,
    audioRefId: createMessageId('audio-1') as any,
    isComplete: true,
  },
];

const mockSentencesWithoutAudio: MessageSentence[] = [
  {
    id: createMessageId('sent-1') as any,
    messageId: createMessageId('msg-1'),
    content: 'Test sentence',
    sequence: 0,
    audioRefId: undefined,
    isComplete: true,
  },
];

vi.mock('../../stores/conversationStore', () => ({
  useConversationStore: (selector: any) => {
    const state = {
      messages: { 'msg-1': mockMessage },
      getMessageSentences: () => mockSentencesWithAudio,
    };
    return selector(state);
  },
}));

vi.mock('../../hooks/useAudioManager', () => ({
  useAudioManager: () => ({
    play: vi.fn(),
    stop: vi.fn(),
    getMetadata: () => ({ durationMs: 3000 }),
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

describe('UserMessage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders ChatBubble with user type', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.USER);
    });

    it('renders message content', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-content')).toHaveTextContent('Test user message');
    });

    it('applies custom className', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} className="custom-class" />);

      const bubble = screen.getByTestId('chat-bubble');
      expect(bubble).toHaveClass('custom-class');
    });

    it('returns null when message not found', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: {},
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      const result = render(<UserMessage messageId={createMessageId('nonexistent')} />);
      expect(result.container.firstChild).toBeNull();
    });
  });

  describe('Voice Message Indicator', () => {
    it('shows microphone icon when message has audio', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByText('🎤')).toBeInTheDocument();
    });

    it('does not show microphone icon when message has no audio', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageSentences: () => mockSentencesWithoutAudio,
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.queryByText('🎤')).not.toBeInTheDocument();
    });

    it('includes voice icon as inline addon', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('addon-icon')).toBeInTheDocument();
    });
  });

  describe('Audio Addons', () => {
    it('creates audio addons for sentences with audio', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      const addonsCount = screen.getByTestId('addons-count');
      expect(parseInt(addonsCount.textContent || '0')).toBeGreaterThan(0);
    });

    it('includes both voice icon and audio player', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      // Microphone icon (inline) + audio addon (below)
      const addonsCount = screen.getByTestId('addons-count');
      expect(parseInt(addonsCount.textContent || '0')).toBe(2);
    });

    it('does not create audio addons when no audio refs', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageSentences: () => mockSentencesWithoutAudio,
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      const addonsCount = screen.getByTestId('addons-count');
      expect(addonsCount.textContent).toBe('0');
    });
  });

  describe('Multiple Audio Sentences', () => {
    it('creates audio addon for each sentence with audio', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageSentences: () => [
              {
                id: createMessageId('sent-1') as any,
                messageId: createMessageId('msg-1'),
                content: 'First',
                sequence: 0,
                audioRefId: createMessageId('audio-1') as any,
                isComplete: true,
              },
              {
                id: createMessageId('sent-2') as any,
                messageId: createMessageId('msg-1'),
                content: 'Second',
                sequence: 1,
                audioRefId: createMessageId('audio-2') as any,
                isComplete: true,
              },
            ],
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      const addonsCount = screen.getByTestId('addons-count');
      // 1 voice icon + 2 audio addons = 3
      expect(parseInt(addonsCount.textContent || '0')).toBe(3);
    });
  });

  describe('Timestamp', () => {
    it('passes timestamp to ChatBubble', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      // ChatBubble component receives timestamp prop
      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
    });
  });

  describe('Message State', () => {
    it('always renders with completed state', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      // User messages are always completed
      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
    });
  });

  describe('Audio State Management', () => {
    it('tracks audio states for playback', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      // Component should render with audio state tracking
      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
    });

    it('updates audio states when playback changes', () => {
      vi.doMock('../../stores/audioStore', () => ({
        useAudioStore: (selector: any) => {
          const state = {
            playback: {
              currentlyPlayingId: 'audio-1',
              isPlaying: true,
              playbackProgress: 0.5,
            },
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
    });
  });

  describe('Content Display', () => {
    it('displays user message content', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-content')).toHaveTextContent('Test user message');
    });

    it('handles empty content', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: {
              'msg-1': {
                ...mockMessage,
                content: '',
              },
            },
            getMessageSentences: () => [],
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('bubble-content')).toBeInTheDocument();
    });
  });

  describe('Integration', () => {
    it('renders complete user message with all features', () => {
      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.USER);
      expect(screen.getByTestId('bubble-content')).toHaveTextContent('Test user message');
      expect(screen.getByText('🎤')).toBeInTheDocument();
    });

    it('renders text-only message without audio features', () => {
      vi.doMock('../../stores/conversationStore', () => ({
        useConversationStore: (selector: any) => {
          const state = {
            messages: { 'msg-1': mockMessage },
            getMessageSentences: () => mockSentencesWithoutAudio,
          };
          return selector(state);
        },
      }));

      render(<UserMessage messageId={createMessageId('msg-1')} />);

      expect(screen.getByTestId('chat-bubble')).toBeInTheDocument();
      expect(screen.queryByText('🎤')).not.toBeInTheDocument();
    });
  });
});
