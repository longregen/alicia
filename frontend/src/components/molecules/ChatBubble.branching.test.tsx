import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import ChatBubble from './ChatBubble';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { useBranchStore } from '../../stores/branchStore';
import { createMessageId } from '../../types/streaming';

// Mock fetch for API calls
const mockFetch = vi.fn();
global.fetch = mockFetch;

/**
 * Integration tests for message branching functionality in ChatBubble.
 * Tests the full workflow: branch display, navigation via API, and content switching.
 *
 * Note: The new branch system syncs with backend - branches are fetched from server
 * and branch switching updates the conversation tip on the server.
 */
describe('ChatBubble - Message Branching (Server-Synced)', () => {
  const messageId = createMessageId('msg-test-123');
  const mockTimestamp = new Date('2025-01-15T10:30:00Z');

  beforeEach(() => {
    // Reset branch store before each test
    useBranchStore.setState({
      branchStates: new Map(),
    });
    mockFetch.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  /**
   * Helper to set up branch state with siblings
   */
  const setupBranchState = (siblings: Array<{ id: string; content: string }>, currentIndex = 0) => {
    useBranchStore.setState({
      branchStates: new Map([
        [messageId, {
          siblings: siblings.map((s, i) => ({
            id: s.id,
            content: s.content,
            createdAt: `2025-01-15T10:0${i}:00Z`,
          })),
          currentIndex,
          loading: false,
          error: null,
        }],
      ]),
    });
  };

  describe('Branch Navigator Visibility', () => {
    it('does not show branch navigator when message has no branches', () => {
      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // BranchNavigator should not render when no siblings
      expect(screen.queryByLabelText('Previous branch')).not.toBeInTheDocument();
      expect(screen.queryByLabelText('Next branch')).not.toBeInTheDocument();
    });

    it('shows branch navigator when message has multiple siblings', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'Original message' },
        { id: 'msg-2', content: 'Edited message v2' },
      ], 1);

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      await waitFor(() => {
        expect(screen.getByLabelText('Previous branch')).toBeInTheDocument();
        expect(screen.getByLabelText('Next branch')).toBeInTheDocument();
      });
    });

    it('shows branch navigator for assistant messages with siblings', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'Response A' },
        { id: 'msg-2', content: 'Response B' },
      ], 0);

      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          content="Response A"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      // Assistant messages can also have branches (regenerated responses)
      await waitFor(() => {
        expect(screen.getByLabelText('Previous branch')).toBeInTheDocument();
      });
    });
  });

  describe('Branch Display and Counter', () => {
    it('displays correct branch counter "1/3" format', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
        { id: 'msg-3', content: 'Version 3' },
      ], 0);

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      await waitFor(() => {
        // Should show "1/3" - at first branch of 3
        expect(screen.getByText('1/3')).toBeInTheDocument();
      });
    });

    it('updates counter when navigating between branches', async () => {
      const user = userEvent.setup();

      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
        { id: 'msg-3', content: 'Version 3' },
      ], 0);

      // Mock successful branch switch
      mockFetch.mockResolvedValue({ ok: true });

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
          onBranchSwitch={vi.fn()}
        />
      );

      // Currently at "1/3"
      await waitFor(() => {
        expect(screen.getByText('1/3')).toBeInTheDocument();
      });

      // Navigate to next
      const nextButton = screen.getByLabelText('Next branch');
      await user.click(nextButton);

      // Should now show "2/3" after API call succeeds
      await waitFor(() => {
        expect(screen.getByText('2/3')).toBeInTheDocument();
      });
    });
  });

  describe('Branch Navigation API Calls', () => {
    it('calls switch-branch API when navigating', async () => {
      const user = userEvent.setup();
      const onBranchSwitch = vi.fn();

      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
      ], 0);

      mockFetch.mockResolvedValue({ ok: true });

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
          onBranchSwitch={onBranchSwitch}
        />
      );

      const nextButton = screen.getByLabelText('Next branch');
      await user.click(nextButton);

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('/switch-branch'),
          expect.objectContaining({
            method: 'PUT',
            body: JSON.stringify({ tipMessageId: 'msg-2' }),
          })
        );
      });

      // Should call onBranchSwitch callback with new message ID
      expect(onBranchSwitch).toHaveBeenCalledWith('msg-2');
    });

    it('handles API errors gracefully', async () => {
      const user = userEvent.setup();

      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
      ], 0);

      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        text: () => Promise.resolve('Server Error'),
      });

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      const nextButton = screen.getByLabelText('Next branch');
      await user.click(nextButton);

      // Should still be at index 0 after error
      await waitFor(() => {
        expect(screen.getByText('1/2')).toBeInTheDocument();
      });
    });
  });

  describe('Branch Navigation Constraints', () => {
    it('disables previous button on first branch', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'First' },
        { id: 'msg-2', content: 'Second' },
      ], 0); // At first branch

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      await waitFor(() => {
        const prevButton = screen.getByLabelText('Previous branch');
        expect(prevButton).toBeDisabled();
      });
    });

    it('disables next button on last branch', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'First' },
        { id: 'msg-2', content: 'Second' },
      ], 1); // At last branch

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Second"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      await waitFor(() => {
        const nextButton = screen.getByLabelText('Next branch');
        expect(nextButton).toBeDisabled();
      });
    });

    it('enables both buttons when in middle of branch list', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'First' },
        { id: 'msg-2', content: 'Second' },
        { id: 'msg-3', content: 'Third' },
      ], 1); // At middle branch

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Second"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      await waitFor(() => {
        const prevButton = screen.getByLabelText('Previous branch');
        const nextButton = screen.getByLabelText('Next branch');
        expect(prevButton).not.toBeDisabled();
        expect(nextButton).not.toBeDisabled();
      });
    });
  });

  describe('Loading State', () => {
    it('shows loading indicator when switching branches', async () => {
      const user = userEvent.setup();

      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
      ], 0);

      // Create a promise we can control
      let resolveSwitch: () => void;
      const switchPromise = new Promise<void>((resolve) => {
        resolveSwitch = resolve;
      });

      mockFetch.mockReturnValue(
        switchPromise.then(() => ({ ok: true }))
      );

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
          onBranchSwitch={vi.fn()}
        />
      );

      // Click next
      const nextButton = screen.getByLabelText('Next branch');
      await user.click(nextButton);

      // Should show loading state
      await waitFor(() => {
        const branchState = useBranchStore.getState().branchStates.get(messageId);
        expect(branchState?.loading).toBe(true);
      });

      // Resolve the switch
      resolveSwitch!();

      // Should no longer be loading
      await waitFor(() => {
        const branchState = useBranchStore.getState().branchStates.get(messageId);
        expect(branchState?.loading).toBe(false);
      });
    });
  });

  describe('Branch State Persistence', () => {
    it('maintains branch state across re-renders', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'Version 1' },
        { id: 'msg-2', content: 'Version 2' },
        { id: 'msg-3', content: 'Version 3' },
      ], 2);

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 3"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      // Verify initial state (at version 3)
      await waitFor(() => {
        expect(screen.getByText('3/3')).toBeInTheDocument();
      });

      // Re-render with same props
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 3"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      // State should be preserved
      await waitFor(() => {
        expect(screen.getByText('3/3')).toBeInTheDocument();
      });
    });
  });

  describe('Content Display from Siblings', () => {
    it('uses message content prop when no sibling state', () => {
      // No branch state set up

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original content from props"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      expect(screen.getByText('Original content from props')).toBeInTheDocument();
    });

    it('uses sibling content when branch state exists', async () => {
      setupBranchState([
        { id: 'msg-1', content: 'Sibling version A' },
        { id: 'msg-2', content: 'Sibling version B' },
      ], 1); // Currently showing version B

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original content from props"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
          conversationId="conv-1"
        />
      );

      // Should show sibling content, not props content
      await waitFor(() => {
        expect(screen.getByText('Sibling version B')).toBeInTheDocument();
      });
    });
  });
});
