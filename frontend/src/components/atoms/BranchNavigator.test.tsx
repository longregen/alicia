import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import BranchNavigator from './BranchNavigator';
import { createMessageId } from '../../types/streaming';

describe('BranchNavigator', () => {
  const mockMessageId = createMessageId('msg-1');
  const mockOnNavigate = vi.fn();

  it('renders branch counter with current and total', () => {
    render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={0}
        totalBranches={3}
        onNavigate={mockOnNavigate}
      />
    );

    expect(screen.getByText('1/3')).toBeInTheDocument();
  });

  it('does not render when there is only one branch', () => {
    const { container } = render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={0}
        totalBranches={1}
        onNavigate={mockOnNavigate}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('does not render when there are no branches', () => {
    const { container } = render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={0}
        totalBranches={0}
        onNavigate={mockOnNavigate}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('disables previous button on first branch', () => {
    render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={0}
        totalBranches={3}
        onNavigate={mockOnNavigate}
      />
    );

    const prevButton = screen.getByLabelText('Previous branch');
    expect(prevButton).toBeDisabled();
  });

  it('disables next button on last branch', () => {
    render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={2}
        totalBranches={3}
        onNavigate={mockOnNavigate}
      />
    );

    const nextButton = screen.getByLabelText('Next branch');
    expect(nextButton).toBeDisabled();
  });

  it('calls onNavigate with "prev" when previous button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={1}
        totalBranches={3}
        onNavigate={mockOnNavigate}
      />
    );

    const prevButton = screen.getByLabelText('Previous branch');
    await user.click(prevButton);

    expect(mockOnNavigate).toHaveBeenCalledWith('prev');
  });

  it('calls onNavigate with "next" when next button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={1}
        totalBranches={3}
        onNavigate={mockOnNavigate}
      />
    );

    const nextButton = screen.getByLabelText('Next branch');
    await user.click(nextButton);

    expect(mockOnNavigate).toHaveBeenCalledWith('next');
  });

  it('displays correct index when navigating through branches', () => {
    const { rerender } = render(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={0}
        totalBranches={5}
        onNavigate={mockOnNavigate}
      />
    );

    expect(screen.getByText('1/5')).toBeInTheDocument();

    rerender(
      <BranchNavigator
        messageId={mockMessageId}
        currentIndex={2}
        totalBranches={5}
        onNavigate={mockOnNavigate}
      />
    );

    expect(screen.getByText('3/5')).toBeInTheDocument();
  });
});
