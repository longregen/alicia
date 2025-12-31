import { render, screen, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import ChatBubble from './ChatBubble';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import type { MessageAddon, ToolData } from '../../types/components';

// Mock child components to isolate ChatBubble tests
vi.mock('../atoms/MessageBubble', () => ({
  default: ({ type, content, timestamp, state }: any) => (
    <div data-testid="message-bubble">
      <div data-testid="bubble-type">{type}</div>
      <div data-testid="bubble-content">{content}</div>
      <div data-testid="bubble-timestamp">{timestamp.toISOString()}</div>
      <div data-testid="bubble-state">{state}</div>
    </div>
  ),
}));

vi.mock('../atoms/ComplexAddons', () => ({
  default: ({ addons, toolDetails, timestamp }: any) => (
    <div data-testid="complex-addons">
      <div data-testid="addons-count">{addons.length}</div>
      <div data-testid="tools-count">{toolDetails.length}</div>
      <div data-testid="addons-timestamp">{timestamp.toISOString()}</div>
    </div>
  ),
}));

describe('ChatBubble', () => {
  const mockTimestamp = new Date('2025-01-15T10:30:00Z');

  beforeEach(() => {
    vi.clearAllTimers();
    vi.useFakeTimers();
  });

  describe('Basic Rendering', () => {
    it('renders with default props', () => {
      render(<ChatBubble />);

      expect(screen.getByTestId('message-bubble')).toBeInTheDocument();
      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.USER);
      expect(screen.getByTestId('bubble-state')).toHaveTextContent(MESSAGE_STATES.COMPLETED);
    });

    it('renders with custom content', () => {
      render(<ChatBubble content="Hello, world!" />);

      const content = screen.getByTestId('bubble-content');
      expect(content).toBeInTheDocument();
      expect(content.textContent).toContain('Hello, world!');
    });

    it('renders with custom timestamp', () => {
      render(<ChatBubble timestamp={mockTimestamp} />);

      expect(screen.getByTestId('bubble-timestamp')).toHaveTextContent('2025-01-15T10:30:00.000Z');
      expect(screen.getByTestId('addons-timestamp')).toHaveTextContent('2025-01-15T10:30:00.000Z');
    });

    it('applies custom className', () => {
      const { container } = render(<ChatBubble className="custom-class" />);

      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('custom-class');
    });
  });

  describe('Role-based Styling', () => {
    it('renders user message with correct type', () => {
      render(<ChatBubble type={MESSAGE_TYPES.USER} content="User message" />);

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.USER);
    });

    it('renders assistant message with correct type', () => {
      render(<ChatBubble type={MESSAGE_TYPES.ASSISTANT} content="Assistant message" />);

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.ASSISTANT);
    });

    it('renders system message with correct type', () => {
      render(<ChatBubble type={MESSAGE_TYPES.SYSTEM} content="System message" />);

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.SYSTEM);
    });
  });

  describe('Content Rendering with Markdown', () => {
    it('renders plain text content', () => {
      render(<ChatBubble content="Simple text message" />);

      const content = screen.getByTestId('bubble-content');
      expect(content.textContent).toContain('Simple text message');
    });

    it('renders multi-line content', () => {
      const multilineContent = 'Line 1\nLine 2\nLine 3';
      render(<ChatBubble content={multilineContent} />);

      const content = screen.getByTestId('bubble-content');
      expect(content.textContent).toContain('Line 1');
      expect(content.textContent).toContain('Line 2');
      expect(content.textContent).toContain('Line 3');
    });

    it('renders content with special characters', () => {
      const specialContent = 'Content with <tags> and & symbols';
      render(<ChatBubble content={specialContent} />);

      const content = screen.getByTestId('bubble-content');
      expect(content.textContent).toContain('Content with <tags> and & symbols');
    });
  });

  describe('Reasoning Block Parsing', () => {
    it('extracts and renders single reasoning block', () => {
      vi.useRealTimers();
      const contentWithReasoning = 'Before reasoning <reasoning>Step 1: Analyze\nStep 2: Conclude</reasoning> After reasoning';
      render(<ChatBubble content={contentWithReasoning} />);

      expect(screen.getByText('Reasoning')).toBeInTheDocument();
      expect(screen.getByText('Before reasoning')).toBeInTheDocument();
      expect(screen.getByText('After reasoning')).toBeInTheDocument();
      vi.useFakeTimers();
    });

    it('renders reasoning block in collapsed state by default', () => {
      vi.useRealTimers();
      const contentWithReasoning = 'Text <reasoning>This is a very long reasoning content that should be truncated in the preview because it exceeds one hundred characters of text</reasoning> More text';
      render(<ChatBubble content={contentWithReasoning} />);

      const button = screen.getByRole('button', { name: /expand reasoning/i });
      expect(button).toHaveAttribute('aria-expanded', 'false');
      vi.useFakeTimers();
    });

    it('expands reasoning block when clicked', async () => {
      vi.useRealTimers();
      const user = userEvent.setup();
      const reasoningContent = 'This is a very long reasoning content that should be truncated in the preview because it exceeds one hundred characters of text';
      const contentWithReasoning = `Text <reasoning>${reasoningContent}</reasoning> More text`;

      render(<ChatBubble content={contentWithReasoning} />);

      const button = screen.getByRole('button', { name: /expand reasoning/i });
      await user.click(button);

      expect(button).toHaveAttribute('aria-expanded', 'true');
      expect(screen.getByText(reasoningContent)).toBeInTheDocument();
      vi.useFakeTimers();
    });

    it('collapses reasoning block when clicked again', async () => {
      vi.useRealTimers();
      const user = userEvent.setup();
      const reasoningContent = 'This is a very long reasoning content that should be truncated in the preview because it exceeds one hundred characters of text';
      const contentWithReasoning = `Text <reasoning>${reasoningContent}</reasoning> More text`;

      render(<ChatBubble content={contentWithReasoning} />);

      const button = screen.getByRole('button', { name: /expand reasoning/i });

      // Expand
      await user.click(button);
      expect(button).toHaveAttribute('aria-expanded', 'true');

      // Collapse
      await user.click(button);
      expect(button).toHaveAttribute('aria-expanded', 'false');
      vi.useFakeTimers();
    });

    it('shows "Show more" button for long reasoning content', () => {
      const longReasoning = 'A'.repeat(150);
      const contentWithReasoning = `Text <reasoning>${longReasoning}</reasoning> More`;

      render(<ChatBubble content={contentWithReasoning} />);

      expect(screen.getByText('Show more')).toBeInTheDocument();
    });

    it('does not show "Show more" for short reasoning content', () => {
      const shortReasoning = 'Short reasoning';
      const contentWithReasoning = `Text <reasoning>${shortReasoning}</reasoning> More`;

      render(<ChatBubble content={contentWithReasoning} />);

      expect(screen.queryByText('Show more')).not.toBeInTheDocument();
    });

    it('handles multiple reasoning blocks', () => {
      const contentWithMultipleReasoning =
        'Start <reasoning data-sequence="2">Second reasoning</reasoning> Middle <reasoning data-sequence="1">First reasoning</reasoning> End';

      render(<ChatBubble content={contentWithMultipleReasoning} />);

      const reasoningButtons = screen.getAllByText('Reasoning');
      expect(reasoningButtons).toHaveLength(2);
    });

    it('sorts multiple reasoning blocks by sequence number', () => {
      const contentWithMultipleReasoning =
        '<reasoning data-sequence="3">Third</reasoning><reasoning data-sequence="1">First</reasoning><reasoning data-sequence="2">Second</reasoning>';

      const { container } = render(<ChatBubble content={contentWithMultipleReasoning} />);

      const reasoningBlocks = container.querySelectorAll('[class*="bg-reasoning"]');
      expect(reasoningBlocks).toHaveLength(3);

      // Check order by looking at content
      const blockContents = Array.from(reasoningBlocks).map(block =>
        block.textContent?.replace('Reasoning', '').trim()
      );
      expect(blockContents[0]).toContain('First');
      expect(blockContents[1]).toContain('Second');
      expect(blockContents[2]).toContain('Third');
    });

    it('handles reasoning blocks without sequence attribute', () => {
      const contentWithReasoning = '<reasoning>No sequence</reasoning>';
      render(<ChatBubble content={contentWithReasoning} />);

      expect(screen.getByText('Reasoning')).toBeInTheDocument();
      expect(screen.getByText('No sequence')).toBeInTheDocument();
    });

    it('renders text before and after reasoning blocks', () => {
      const contentWithReasoning = 'Prefix text <reasoning>Reasoning content</reasoning> Suffix text';
      render(<ChatBubble content={contentWithReasoning} />);

      expect(screen.getByText('Prefix text')).toBeInTheDocument();
      expect(screen.getByText('Suffix text')).toBeInTheDocument();
    });
  });

  describe('Streaming State and Typing Animation', () => {
    it('shows streaming badge for assistant messages in streaming state', () => {
      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          state={MESSAGE_STATES.STREAMING}
          content="Streaming content"
        />
      );

      expect(screen.getByText('Streaming')).toBeInTheDocument();
    });

    it('does not show streaming badge for user messages', () => {
      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          state={MESSAGE_STATES.STREAMING}
          content="User streaming"
        />
      );

      expect(screen.queryByText('Streaming')).not.toBeInTheDocument();
    });

    it('does not show streaming badge when not in streaming state', () => {
      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          state={MESSAGE_STATES.COMPLETED}
          content="Completed content"
        />
      );

      expect(screen.queryByText('Streaming')).not.toBeInTheDocument();
    });

    it('animates typing effect in streaming state', async () => {
      const streamingText = 'Hello, world!';
      render(
        <ChatBubble
          state={MESSAGE_STATES.STREAMING}
          streamingText={streamingText}
        />
      );

      // Initially, content should be empty or minimal
      const content = screen.getByTestId('bubble-content');

      // Advance timers to simulate typing animation
      await act(async () => {
        await vi.advanceTimersByTimeAsync(30); // First character
      });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(300); // More characters
      });

      // Content should be progressively revealed
      expect(content.textContent).toBeTruthy();
      const currentLength = content.textContent?.length || 0;
      expect(currentLength).toBeGreaterThan(0);
    });

    it('shows cursor during streaming', () => {
      render(
        <ChatBubble
          state={MESSAGE_STATES.STREAMING}
          streamingText="Typing..."
        />
      );

      const content = screen.getByTestId('bubble-content');
      const wrapper = content.parentElement;

      // Check for the cursor element (animated pulse element)
      expect(wrapper?.querySelector('[class*="animate-pulse"]')).toBeInTheDocument();
    });

    it('uses streamingText prop over content in streaming state', async () => {
      render(
        <ChatBubble
          state={MESSAGE_STATES.STREAMING}
          content="Original content"
          streamingText="Streaming text"
        />
      );

      // Let typing animation progress through multiple characters
      for (let i = 0; i < 12; i++) {
        await act(async () => {
          await vi.advanceTimersByTimeAsync(30);
        });
      }

      const content = screen.getByTestId('bubble-content');
      // Check that we're typing "Streaming text" not "Original content"
      // After 12 iterations, should have "Streaming te" or similar
      expect(content.textContent).toContain('Streaming');
      expect(content.textContent).not.toContain('Original');
    });

    it('resets typing animation when switching to completed state', async () => {
      const { rerender } = render(
        <ChatBubble
          state={MESSAGE_STATES.STREAMING}
          content="Hello"
        />
      );

      await act(async () => {
        await vi.advanceTimersByTimeAsync(100);
      });

      rerender(
        <ChatBubble
          state={MESSAGE_STATES.COMPLETED}
          content="Hello, world!"
        />
      );

      const content = screen.getByTestId('bubble-content');
      expect(content.textContent).toContain('Hello, world!');
    });
  });

  describe('Timestamp Display', () => {
    it('displays timestamp via ComplexAddons', () => {
      render(<ChatBubble timestamp={mockTimestamp} />);

      expect(screen.getByTestId('addons-timestamp')).toHaveTextContent('2025-01-15T10:30:00.000Z');
    });

    it('updates when timestamp changes', () => {
      const initialTimestamp = new Date('2025-01-15T10:00:00Z');
      const updatedTimestamp = new Date('2025-01-15T11:00:00Z');

      const { rerender } = render(<ChatBubble timestamp={initialTimestamp} />);
      expect(screen.getByTestId('addons-timestamp')).toHaveTextContent('2025-01-15T10:00:00.000Z');

      rerender(<ChatBubble timestamp={updatedTimestamp} />);
      expect(screen.getByTestId('addons-timestamp')).toHaveTextContent('2025-01-15T11:00:00.000Z');
    });
  });

  describe('Addon Rendering', () => {
    it('renders inline addons', () => {
      const addons: MessageAddon[] = [
        {
          id: 'addon1',
          type: 'icon',
          position: 'inline',
          emoji: '🔍',
          tooltip: 'Search',
        },
      ];

      render(<ChatBubble addons={addons} />);

      expect(screen.getByTestId('addons-count')).toHaveTextContent('1');
    });

    it('renders below addons separately', () => {
      const addons: MessageAddon[] = [
        {
          id: 'addon1',
          type: 'icon',
          position: 'below',
          emoji: '📎',
          tooltip: 'Attachment',
          content: <div>Attachment content</div>,
        },
      ];

      render(<ChatBubble addons={addons} />);

      expect(screen.getByText('Attachment content')).toBeInTheDocument();
    });

    it('separates inline and below addons correctly', () => {
      const addons: MessageAddon[] = [
        {
          id: 'addon1',
          type: 'icon',
          position: 'inline',
          emoji: '🔍',
          tooltip: 'Search',
        },
        {
          id: 'addon2',
          type: 'tool',
          position: 'below',
          emoji: '🔧',
          tooltip: 'Tool',
          content: <div>Tool content</div>,
        },
        {
          id: 'addon3',
          type: 'icon',
          position: 'inline',
          emoji: '📌',
          tooltip: 'Pin',
        },
      ];

      render(<ChatBubble addons={addons} />);

      // ComplexAddons should receive only inline addons (2)
      expect(screen.getByTestId('addons-count')).toHaveTextContent('2');

      // Below addon content should be rendered
      expect(screen.getByText('Tool content')).toBeInTheDocument();
    });

    it('renders audio addons', () => {
      const addons: MessageAddon[] = [
        {
          id: 'audio1',
          type: 'audio',
          position: 'inline',
          emoji: '🔊',
          tooltip: 'Audio',
        },
      ];

      render(<ChatBubble addons={addons} />);

      expect(screen.getByTestId('addons-count')).toHaveTextContent('1');
    });

    it('renders multiple addons of different types', () => {
      const addons: MessageAddon[] = [
        {
          id: 'icon1',
          type: 'icon',
          position: 'inline',
          emoji: '🌍',
          tooltip: 'Translation',
        },
        {
          id: 'audio1',
          type: 'audio',
          position: 'inline',
          emoji: '🔊',
          tooltip: 'Audio',
        },
        {
          id: 'tool1',
          type: 'tool',
          position: 'inline',
          emoji: '🔧',
          tooltip: 'Tool',
        },
      ];

      render(<ChatBubble addons={addons} />);

      expect(screen.getByTestId('addons-count')).toHaveTextContent('3');
    });

    it('handles empty addons array', () => {
      render(<ChatBubble addons={[]} />);

      expect(screen.getByTestId('addons-count')).toHaveTextContent('0');
    });
  });

  describe('Tool Status Display', () => {
    it('renders tools via ComplexAddons', () => {
      const tools: ToolData[] = [
        {
          id: 'tool1',
          name: 'Search',
          description: 'Search the web',
          status: 'completed',
          result: 'Found 10 results',
        },
      ];

      render(<ChatBubble tools={tools} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('renders multiple tools with different statuses', () => {
      const tools: ToolData[] = [
        {
          id: 'tool1',
          name: 'Search',
          description: 'Search the web',
          status: 'running',
        },
        {
          id: 'tool2',
          name: 'Calculator',
          description: 'Perform calculations',
          status: 'completed',
          result: '42',
        },
        {
          id: 'tool3',
          name: 'Database',
          description: 'Query database',
          status: 'error',
        },
      ];

      render(<ChatBubble tools={tools} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('3');
    });

    it('handles empty tools array', () => {
      render(<ChatBubble tools={[]} />);

      expect(screen.getByTestId('tools-count')).toHaveTextContent('0');
    });

    it('converts tools to ToolDetail format for ComplexAddons', () => {
      const tools: ToolData[] = [
        {
          id: 'tool1',
          name: 'Test Tool',
          description: 'A test tool',
          status: 'completed',
          result: 'Success',
        },
      ];

      render(<ChatBubble tools={tools} />);

      // Verify that ComplexAddons receives the tools
      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });
  });

  describe('Integration Tests', () => {
    it('renders complete message with all features', () => {
      const addons: MessageAddon[] = [
        {
          id: 'addon1',
          type: 'icon',
          position: 'inline',
          emoji: '🌍',
          tooltip: 'Translation',
        },
      ];

      const tools: ToolData[] = [
        {
          id: 'tool1',
          name: 'Search',
          description: 'Web search',
          status: 'completed',
          result: 'Found results',
        },
      ];

      const content = 'Here is my analysis: <reasoning>Step 1: Gather data\nStep 2: Process</reasoning> The result is conclusive.';

      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          content={content}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          addons={addons}
          tools={tools}
        />
      );

      // Verify all parts are rendered
      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.ASSISTANT);
      expect(screen.getByText('Here is my analysis:')).toBeInTheDocument();
      expect(screen.getByText('Reasoning')).toBeInTheDocument();
      expect(screen.getByText('The result is conclusive.')).toBeInTheDocument();
      expect(screen.getByTestId('addons-count')).toHaveTextContent('1');
      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');
    });

    it('handles user message with multiple addons and no tools', () => {
      const addons: MessageAddon[] = [
        {
          id: 'addon1',
          type: 'icon',
          position: 'inline',
          emoji: '📎',
          tooltip: 'Attachment',
        },
        {
          id: 'addon2',
          type: 'audio',
          position: 'inline',
          emoji: '🎤',
          tooltip: 'Voice',
        },
      ];

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="User message with attachments"
          addons={addons}
          tools={[]}
        />
      );

      expect(screen.getByTestId('bubble-type')).toHaveTextContent(MESSAGE_TYPES.USER);
      expect(screen.getByTestId('addons-count')).toHaveTextContent('2');
      expect(screen.getByTestId('tools-count')).toHaveTextContent('0');
    });

    it('handles assistant message during streaming with tools', async () => {
      const tools: ToolData[] = [
        {
          id: 'tool1',
          name: 'Calculator',
          description: 'Calculating...',
          status: 'running',
        },
      ];

      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          state={MESSAGE_STATES.STREAMING}
          streamingText="Computing the answer..."
          tools={tools}
        />
      );

      expect(screen.getByText('Streaming')).toBeInTheDocument();
      expect(screen.getByTestId('tools-count')).toHaveTextContent('1');

      await act(async () => {
        await vi.advanceTimersByTimeAsync(100);
        await vi.advanceTimersByTimeAsync(500);
      });

      const content = screen.getByTestId('bubble-content');
      expect(content.textContent).toBeTruthy();
    });
  });
});
