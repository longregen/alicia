import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FeedbackControls from './FeedbackControls';

describe('FeedbackControls', () => {
  const mockOnVote = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders upvote and downvote buttons', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      expect(screen.getByRole('button', { name: 'Upvote' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Downvote' })).toBeInTheDocument();
    });

    it('applies custom className', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} className="custom-class" />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('does not show counts when zero', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} upvotes={0} downvotes={0} />);

      const buttons = screen.getAllByRole('button');
      buttons.forEach(button => {
        expect(button).not.toHaveTextContent('0');
      });
    });

    it('shows upvote count when greater than zero', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} upvotes={5} />);

      expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('shows downvote count when greater than zero', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} downvotes={3} />);

      expect(screen.getByText('3')).toBeInTheDocument();
    });
  });

  describe('Vote State Display', () => {
    it('shows upvote as active when currentVote is up', () => {
      render(<FeedbackControls currentVote="up" onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Remove upvote' });
      expect(upvoteButton).toHaveClass('bg-success-subtle');
    });

    it('shows downvote as active when currentVote is down', () => {
      render(<FeedbackControls currentVote="down" onVote={mockOnVote} />);

      const downvoteButton = screen.getByRole('button', { name: 'Remove downvote' });
      expect(downvoteButton).toHaveClass('bg-error-subtle');
    });

    it('shows neutral state when currentVote is null', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      const downvoteButton = screen.getByRole('button', { name: 'Downvote' });

      expect(upvoteButton).not.toHaveClass('bg-success-subtle');
      expect(downvoteButton).not.toHaveClass('bg-error-subtle');
    });
  });

  describe('Click Interactions', () => {
    it('calls onVote with "up" when upvote button is clicked', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      fireEvent.click(upvoteButton);

      expect(mockOnVote).toHaveBeenCalledWith('up');
    });

    it('calls onVote with "down" when downvote button is clicked', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      const downvoteButton = screen.getByRole('button', { name: 'Downvote' });
      fireEvent.click(downvoteButton);

      expect(mockOnVote).toHaveBeenCalledWith('down');
    });

    it('calls onVote when clicking already active upvote', () => {
      render(<FeedbackControls currentVote="up" onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Remove upvote' });
      fireEvent.click(upvoteButton);

      expect(mockOnVote).toHaveBeenCalledWith('up');
    });

    it('calls onVote when clicking already active downvote', () => {
      render(<FeedbackControls currentVote="down" onVote={mockOnVote} />);

      const downvoteButton = screen.getByRole('button', { name: 'Remove downvote' });
      fireEvent.click(downvoteButton);

      expect(mockOnVote).toHaveBeenCalledWith('down');
    });
  });

  describe('Loading State', () => {
    it('disables buttons when loading', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} isLoading={true} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      const downvoteButton = screen.getByRole('button', { name: 'Downvote' });

      expect(upvoteButton).toBeDisabled();
      expect(downvoteButton).toBeDisabled();
    });

    it('does not call onVote when loading', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} isLoading={true} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      fireEvent.click(upvoteButton);

      expect(mockOnVote).not.toHaveBeenCalled();
    });

    it('shows loading spinner when loading', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} isLoading={true} />
      );

      const spinner = container.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });

    it('applies reduced opacity when loading', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} isLoading={true} />);

      const buttons = screen.getAllByRole('button');
      buttons.forEach(button => {
        expect(button).toHaveClass('opacity-50');
      });
    });
  });

  describe('Compact Mode', () => {
    it('applies compact styling when compact is true', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} compact={true} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      expect(upvoteButton).toHaveClass('px-1.5');
      expect(upvoteButton).toHaveClass('py-0.5');
      expect(upvoteButton).toHaveClass('text-xs');
    });

    it('applies regular styling when compact is false', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} compact={false} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      expect(upvoteButton).toHaveClass('px-2');
      expect(upvoteButton).toHaveClass('py-1');
      expect(upvoteButton).toHaveClass('text-sm');
    });

    it('renders smaller icons in compact mode', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} compact={true} />
      );

      const icons = container.querySelectorAll('svg');
      icons.forEach(icon => {
        expect(icon).toHaveClass('w-3');
        expect(icon).toHaveClass('h-3');
      });
    });

    it('renders regular icons in normal mode', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} compact={false} />
      );

      const icons = container.querySelectorAll('svg');
      icons.forEach(icon => {
        expect(icon).toHaveClass('w-4');
        expect(icon).toHaveClass('h-4');
      });
    });
  });

  describe('Accessibility', () => {
    it('has correct aria-label for upvote button when not voted', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Upvote' });
      expect(upvoteButton).toHaveAttribute('aria-label', 'Upvote');
    });

    it('has correct aria-label for upvote button when voted up', () => {
      render(<FeedbackControls currentVote="up" onVote={mockOnVote} />);

      const upvoteButton = screen.getByRole('button', { name: 'Remove upvote' });
      expect(upvoteButton).toHaveAttribute('aria-label', 'Remove upvote');
    });

    it('has correct aria-label for downvote button when not voted', () => {
      render(<FeedbackControls currentVote={null} onVote={mockOnVote} />);

      const downvoteButton = screen.getByRole('button', { name: 'Downvote' });
      expect(downvoteButton).toHaveAttribute('aria-label', 'Downvote');
    });

    it('has correct aria-label for downvote button when voted down', () => {
      render(<FeedbackControls currentVote="down" onVote={mockOnVote} />);

      const downvoteButton = screen.getByRole('button', { name: 'Remove downvote' });
      expect(downvoteButton).toHaveAttribute('aria-label', 'Remove downvote');
    });
  });

  describe('Icon Fill State', () => {
    it('fills upvote icon when currentVote is up', () => {
      const { container } = render(
        <FeedbackControls currentVote="up" onVote={mockOnVote} />
      );

      const upvoteSvg = container.querySelector('button[aria-label="Remove upvote"] svg');
      expect(upvoteSvg).toHaveAttribute('fill', 'currentColor');
    });

    it('does not fill upvote icon when currentVote is not up', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} />
      );

      const upvoteSvg = container.querySelector('button[aria-label="Upvote"] svg');
      expect(upvoteSvg).toHaveAttribute('fill', 'none');
    });

    it('fills downvote icon when currentVote is down', () => {
      const { container } = render(
        <FeedbackControls currentVote="down" onVote={mockOnVote} />
      );

      const downvoteSvg = container.querySelector('button[aria-label="Remove downvote"] svg');
      expect(downvoteSvg).toHaveAttribute('fill', 'currentColor');
    });

    it('does not fill downvote icon when currentVote is not down', () => {
      const { container } = render(
        <FeedbackControls currentVote={null} onVote={mockOnVote} />
      );

      const downvoteSvg = container.querySelector('button[aria-label="Downvote"] svg');
      expect(downvoteSvg).toHaveAttribute('fill', 'none');
    });
  });
});
