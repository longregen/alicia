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
      resolution: 'manual',
    },
    onKeepLocal: vi.fn(),
    onKeepServer: vi.fn(),
  };

  it('renders the dialog when open', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText('Message Conflict')).toBeInTheDocument();
    expect(screen.getByText(/This message has a conflict with the server version/)).toBeInTheDocument();
  });

  it('displays local and server content', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText('Local Version')).toBeInTheDocument();
    expect(screen.getByText('Server Version')).toBeInTheDocument();
    expect(screen.getByText(mockProps.localContent)).toBeInTheDocument();
    expect(screen.getByText(mockProps.serverContent)).toBeInTheDocument();
  });

  it('displays conflict reason when provided', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    expect(screen.getByText(/Content mismatch with existing message/)).toBeInTheDocument();
  });

  it('calls onKeepLocal and closes dialog when Keep Local is clicked', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    const keepLocalButton = screen.getByText('Keep Local');
    fireEvent.click(keepLocalButton);

    expect(mockProps.onKeepLocal).toHaveBeenCalledTimes(1);
    expect(mockProps.onOpenChange).toHaveBeenCalledWith(false);
  });

  it('calls onKeepServer and closes dialog when Keep Server is clicked', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    const keepServerButton = screen.getByText('Keep Server');
    fireEvent.click(keepServerButton);

    expect(mockProps.onKeepServer).toHaveBeenCalledTimes(1);
    expect(mockProps.onOpenChange).toHaveBeenCalledWith(false);
  });

  it('closes dialog when Cancel is clicked', () => {
    render(<ConflictResolutionDialog {...mockProps} />);

    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    expect(mockProps.onOpenChange).toHaveBeenCalledWith(false);
    expect(mockProps.onKeepLocal).not.toHaveBeenCalled();
    expect(mockProps.onKeepServer).not.toHaveBeenCalled();
  });

  it('does not render when open is false', () => {
    render(<ConflictResolutionDialog {...mockProps} open={false} />);

    expect(screen.queryByText('Message Conflict')).not.toBeInTheDocument();
  });
});
