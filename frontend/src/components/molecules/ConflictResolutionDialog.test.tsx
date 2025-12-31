import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import ConflictResolutionDialog from './ConflictResolutionDialog';

describe('ConflictResolutionDialog', () => {
  const mockProps = {
    open: true,
    onOpenChange: vi.fn(),
    localContent: 'This is the local version of the message',
    serverContent: 'This is the server version of the message',
    conflict: {
      reason: 'Content mismatch with existing message',
      resolution: 'manual' as const,
    },
    onKeepLocal: vi.fn(),
    onKeepServer: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the dialog when open', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText('Sync Conflict Detected')).toBeInTheDocument();
    expect(screen.getByText(/This message was modified both locally and on the server/)).toBeInTheDocument();
  });

  it('displays local and server content', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText('Your Version')).toBeInTheDocument();
    expect(screen.getByText('Server Version')).toBeInTheDocument();
    expect(screen.getByText(mockProps.localContent)).toBeInTheDocument();
    expect(screen.getByText(mockProps.serverContent)).toBeInTheDocument();
  });

  it('displays conflict reason when provided', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText(/Content mismatch with existing message/)).toBeInTheDocument();
  });

  it('calls onKeepLocal when Keep Your Version is clicked', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    const keepLocalButton = screen.getByText('Keep Your Version');
    fireEvent.click(keepLocalButton);

    expect(mockProps.onKeepLocal).toHaveBeenCalledTimes(1);
  });

  it('calls onKeepServer when Keep Server Version is clicked', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    const keepServerButton = screen.getByText('Keep Server Version');
    fireEvent.click(keepServerButton);

    expect(mockProps.onKeepServer).toHaveBeenCalledTimes(1);
  });

  it('does not render when open is false', () => {
    render(<ConflictResolutionDialog {...mockProps} open={false} />);

    expect(screen.queryByText('Sync Conflict Detected')).not.toBeInTheDocument();
  });
});
