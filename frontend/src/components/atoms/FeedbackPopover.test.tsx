import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import FeedbackPopover from './FeedbackPopover';

describe('FeedbackPopover', () => {
  it('renders trigger button', () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    expect(trigger).toBeInTheDocument();
  });

  it('opens popover on trigger click', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByText('Provide Feedback')).toBeInTheDocument();
    });
  });

  it('renders all feedback options', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByLabelText('Helpful')).toBeInTheDocument();
      expect(screen.getByLabelText('Not helpful')).toBeInTheDocument();
      expect(screen.getByLabelText('Incorrect')).toBeInTheDocument();
      expect(screen.getByLabelText('Harmful')).toBeInTheDocument();
    });
  });

  it('submits feedback with type only', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByLabelText('Helpful')).toBeInTheDocument();
    });

    const helpfulRadio = screen.getByLabelText('Helpful');
    fireEvent.click(helpfulRadio);

    const submitButton = screen.getByRole('button', { name: /submit/i });
    fireEvent.click(submitButton);

    expect(onSubmit).toHaveBeenCalledWith({
      type: 'helpful',
      comment: undefined,
    });
  });

  it('submits feedback with type and comment', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByLabelText('Helpful')).toBeInTheDocument();
    });

    const helpfulRadio = screen.getByLabelText('Helpful');
    fireEvent.click(helpfulRadio);

    const textarea = screen.getByPlaceholderText('Tell us more...');
    fireEvent.change(textarea, { target: { value: 'Great response!' } });

    const submitButton = screen.getByRole('button', { name: /submit/i });
    fireEvent.click(submitButton);

    expect(onSubmit).toHaveBeenCalledWith({
      type: 'helpful',
      comment: 'Great response!',
    });
  });

  it('disables submit button when no type selected', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /submit/i })).toBeInTheDocument();
    });

    const submitButton = screen.getByRole('button', { name: /submit/i });
    expect(submitButton).toBeDisabled();
  });

  it('closes popover on cancel', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByText('Provide Feedback')).toBeInTheDocument();
    });

    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    fireEvent.click(cancelButton);

    await waitFor(() => {
      expect(screen.queryByText('Provide Feedback')).not.toBeInTheDocument();
    });
  });

  it('resets form after submission', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });

    // First submission
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByLabelText('Helpful')).toBeInTheDocument();
    });

    const helpfulRadio = screen.getByLabelText('Helpful');
    fireEvent.click(helpfulRadio);

    const textarea = screen.getByPlaceholderText('Tell us more...');
    fireEvent.change(textarea, { target: { value: 'Test comment' } });

    const submitButton = screen.getByRole('button', { name: /submit/i });
    fireEvent.click(submitButton);

    // Reopen and verify form is reset
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Tell us more...')).toHaveValue('');
    });
  });

  it('renders as default button variant when specified', () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} variant="default" triggerLabel="Custom Label" />);

    const trigger = screen.getByRole('button', { name: /custom label/i });
    expect(trigger).toBeInTheDocument();
  });

  it('shows loading state on submit button', async () => {
    const onSubmit = vi.fn();
    render(<FeedbackPopover onSubmit={onSubmit} isLoading={true} />);

    const trigger = screen.getByRole('button', { name: /give feedback/i });
    fireEvent.click(trigger);

    await waitFor(() => {
      const submitButton = screen.getByRole('button', { name: /submit/i });
      expect(submitButton).toBeDisabled();
    });
  });
});
