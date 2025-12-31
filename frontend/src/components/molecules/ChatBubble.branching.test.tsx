import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import ChatBubble from './ChatBubble';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { useBranchStore } from '../../stores/branchStore';
import { createMessageId } from '../../types/streaming';

/**
 * Integration tests for message branching functionality in ChatBubble.
 * Tests the full workflow: branch creation, navigation, and display.
 */
describe('ChatBubble - Message Branching', () => {
  const messageId = createMessageId('msg-test-123');
  const mockTimestamp = new Date('2025-01-15T10:30:00Z');

  beforeEach(() => {
    // Reset branch store before each test
    useBranchStore.setState({
      branches: new Map(),
      currentVersionIndex: new Map(),
    });
  });

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

      // BranchNavigator should not render when totalBranches <= 1
      expect(screen.queryByLabelText('Previous branch')).not.toBeInTheDocument();
      expect(screen.queryByLabelText('Next branch')).not.toBeInTheDocument();
    });

    it('shows branch navigator when user message has multiple branches', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      // Initialize with original content
      initializeBranch(messageId, 'Original message');
      // Create a second branch
      createBranch(messageId, 'Edited message v2');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Wait for the component to render with branch state
      await waitFor(() => {
        expect(screen.getByLabelText('Previous branch')).toBeInTheDocument();
        expect(screen.getByLabelText('Next branch')).toBeInTheDocument();
      });
    });

    it('does not show branch navigator for assistant messages', () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();
      initializeBranch(messageId, 'Original');
      createBranch(messageId, 'Edited');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.ASSISTANT}
          content="Assistant response"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Assistant messages don't support branching
      expect(screen.queryByLabelText('Previous branch')).not.toBeInTheDocument();
    });
  });

  describe('Branch Display and Counter', () => {
    it('displays correct branch counter "1/3" format', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'Version 1');
      createBranch(messageId, 'Version 2');
      createBranch(messageId, 'Version 3');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      await waitFor(() => {
        // Should show "3/3" since we created 2 branches (total 3)
        // and current index should be at the last one
        expect(screen.getByText('3/3')).toBeInTheDocument();
      });
    });

    it('updates counter when navigating between branches', async () => {
      const user = userEvent.setup();
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'Version 1');
      createBranch(messageId, 'Version 2');
      createBranch(messageId, 'Version 3');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Currently at "3/3"
      await waitFor(() => {
        expect(screen.getByText('3/3')).toBeInTheDocument();
      });

      // Navigate to previous
      const prevButton = screen.getByLabelText('Previous branch');
      await user.click(prevButton);

      // Force re-render to pick up store changes
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should now show "2/3"
      await waitFor(() => {
        expect(screen.getByText('2/3')).toBeInTheDocument();
      });
    });
  });

  describe('Branch Content Display', () => {
    it('displays current branch content', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First version of the message');
      createBranch(messageId, 'Second version of the message');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First version of the message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should display the latest branch (second version)
      await waitFor(() => {
        expect(screen.getByText('Second version of the message')).toBeInTheDocument();
      });
    });

    it('switches content when navigating to previous branch', async () => {
      const user = userEvent.setup();
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First version');
      createBranch(messageId, 'Second version');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First version"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Currently showing second version
      await waitFor(() => {
        expect(screen.getByText('Second version')).toBeInTheDocument();
      });

      // Navigate to previous branch
      const prevButton = screen.getByLabelText('Previous branch');
      await user.click(prevButton);

      // Re-render to pick up changes
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First version"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should now show first version
      await waitFor(() => {
        expect(screen.getByText('First version')).toBeInTheDocument();
      });
    });

    it('switches content when navigating to next branch', async () => {
      const user = userEvent.setup();
      const { initializeBranch, createBranch, navigateToBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First version');
      createBranch(messageId, 'Second version');
      // Navigate back to first to test "next" navigation
      navigateToBranch(messageId, 'prev');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First version"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Currently showing first version
      await waitFor(() => {
        expect(screen.getByText('First version')).toBeInTheDocument();
      });

      // Navigate to next branch
      const nextButton = screen.getByLabelText('Next branch');
      await user.click(nextButton);

      // Re-render to pick up changes
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First version"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should now show second version
      await waitFor(() => {
        expect(screen.getByText('Second version')).toBeInTheDocument();
      });
    });
  });

  describe('Branch Navigation Constraints', () => {
    it('disables previous button on first branch', async () => {
      const { initializeBranch, createBranch, navigateToBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First');
      createBranch(messageId, 'Second');
      // Navigate to first
      navigateToBranch(messageId, 'prev');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      await waitFor(() => {
        const prevButton = screen.getByLabelText('Previous branch');
        expect(prevButton).toBeDisabled();
      });
    });

    it('disables next button on last branch', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First');
      createBranch(messageId, 'Second');
      // Currently at last branch by default

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      await waitFor(() => {
        const nextButton = screen.getByLabelText('Next branch');
        expect(nextButton).toBeDisabled();
      });
    });

    it('enables both buttons when in middle of branch list', async () => {
      const { initializeBranch, createBranch, navigateToBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'First');
      createBranch(messageId, 'Second');
      createBranch(messageId, 'Third');
      // Navigate to middle (index 1)
      navigateToBranch(messageId, 'prev');

      render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="First"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
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

  describe('Branch Creation via Store', () => {
    it('creates new branch when createBranch is called directly', () => {
      const { initializeBranch, createBranch, getBranchCount, getCurrentBranch } = useBranchStore.getState();

      // Initialize first branch
      initializeBranch(messageId, 'Original message');
      expect(getBranchCount(messageId)).toBe(1);

      // Create new branch via store
      createBranch(messageId, 'Edited message');

      // Should have 2 branches now
      expect(getBranchCount(messageId)).toBe(2);

      // Current branch should be the newly created one
      const current = getCurrentBranch(messageId);
      expect(current?.content).toBe('Edited message');
    });

    it('displays newly created branch content in ChatBubble', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      // Initialize first branch
      initializeBranch(messageId, 'Original message');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Initially shows original
      expect(screen.getByText('Original message')).toBeInTheDocument();

      // Create new branch
      createBranch(messageId, 'Edited message v2');

      // Re-render to pick up store changes
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Original message"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should now show the new branch content
      await waitFor(() => {
        expect(screen.getByText('Edited message v2')).toBeInTheDocument();
      });
    });

    it('can create multiple branches and navigate between them', async () => {
      const { initializeBranch, createBranch, navigateToBranch } = useBranchStore.getState();

      // Create a branch tree
      initializeBranch(messageId, 'Version 1');
      createBranch(messageId, 'Version 2');
      createBranch(messageId, 'Version 3');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should show latest (Version 3)
      await waitFor(() => {
        expect(screen.getByText('Version 3')).toBeInTheDocument();
      });

      // Navigate to previous
      navigateToBranch(messageId, 'prev');
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should show Version 2
      await waitFor(() => {
        expect(screen.getByText('Version 2')).toBeInTheDocument();
      });

      // Navigate to previous again
      navigateToBranch(messageId, 'prev');
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should show Version 1
      await waitFor(() => {
        expect(screen.getByText('Version 1')).toBeInTheDocument();
      });
    });
  });

  describe('Branch State Persistence', () => {
    it('maintains branch state across re-renders', async () => {
      const { initializeBranch, createBranch } = useBranchStore.getState();

      initializeBranch(messageId, 'Version 1');
      createBranch(messageId, 'Version 2');
      createBranch(messageId, 'Version 3');

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Verify initial state (at version 3)
      await waitFor(() => {
        expect(screen.getByText('3/3')).toBeInTheDocument();
        expect(screen.getByText('Version 3')).toBeInTheDocument();
      });

      // Re-render with same props
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Version 1"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // State should be preserved
      await waitFor(() => {
        expect(screen.getByText('3/3')).toBeInTheDocument();
        expect(screen.getByText('Version 3')).toBeInTheDocument();
      });
    });

    it('initializes branch only once on mount', () => {
      const { getBranchCount } = useBranchStore.getState();

      const { rerender } = render(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Initial content"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should have 1 branch after first render
      expect(getBranchCount(messageId)).toBe(1);

      // Re-render multiple times
      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Initial content"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      rerender(
        <ChatBubble
          type={MESSAGE_TYPES.USER}
          content="Initial content"
          messageId={messageId}
          state={MESSAGE_STATES.COMPLETED}
          timestamp={mockTimestamp}
        />
      );

      // Should still have only 1 branch (not re-initialized)
      expect(getBranchCount(messageId)).toBe(1);
    });
  });
});
