import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import InputSendButton from './InputSendButton';

describe('InputSendButton', () => {
  const mockOnSend = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders a button', () => {
      render(<InputSendButton />);

      const button = screen.getByRole('button');
      expect(button).toBeInTheDocument();
    });

    it('renders with type submit', () => {
      render(<InputSendButton />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'submit');
    });

    it('renders paper airplane icon', () => {
      const { container } = render(<InputSendButton />);

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('applies custom className', () => {
      render(<InputSendButton className="custom-class" />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('custom-class');
    });
  });

  describe('Send State (canSend)', () => {
    it('applies primary styling when canSend is true', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('bg-primary-blue');
    });

    it('applies muted styling when canSend is false', () => {
      render(<InputSendButton canSend={false} />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('bg-surface-bg');
    });

    it('is disabled when canSend is false', () => {
      render(<InputSendButton canSend={false} />);

      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });

    it('is enabled when canSend is true', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      expect(button).not.toBeDisabled();
    });

    it('icon has reduced opacity when canSend is false', () => {
      const { container } = render(<InputSendButton canSend={false} />);

      const svg = container.querySelector('svg');
      expect(svg).toHaveClass('opacity-50');
    });
  });

  describe('Disabled State', () => {
    it('applies disabled styling when disabled', () => {
      render(<InputSendButton disabled />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('cursor-not-allowed');
    });

    it('is disabled when disabled prop is true', () => {
      render(<InputSendButton disabled />);

      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });

    it('is disabled when both disabled and canSend are set', () => {
      render(<InputSendButton disabled canSend={true} />);

      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });
  });

  describe('Click Interactions', () => {
    it('calls onSend when clicked and canSend is true', () => {
      render(<InputSendButton canSend={true} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(mockOnSend).toHaveBeenCalledTimes(1);
    });

    it('does not call onSend when clicked and canSend is false', () => {
      render(<InputSendButton canSend={false} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(mockOnSend).not.toHaveBeenCalled();
    });

    it('does not call onSend when disabled', () => {
      render(<InputSendButton disabled canSend={true} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(mockOnSend).not.toHaveBeenCalled();
    });
  });

  describe('Mouse Interactions', () => {
    it('applies pressed styling on mouse down when canSend', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);

      expect(button).toHaveClass('scale-95');
    });

    it('removes pressed styling on mouse up', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);
      fireEvent.mouseUp(button);

      expect(button).not.toHaveClass('scale-95');
    });

    it('removes pressed styling on mouse leave', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);
      fireEvent.mouseLeave(button);

      expect(button).not.toHaveClass('scale-95');
    });

    it('does not apply pressed styling when disabled', () => {
      render(<InputSendButton disabled />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);

      expect(button).not.toHaveClass('scale-95');
    });

    it('does not apply pressed styling when canSend is false', () => {
      render(<InputSendButton canSend={false} />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);

      expect(button).not.toHaveClass('scale-95');
    });
  });

  describe('Keyboard Interactions', () => {
    it('calls onSend on Enter key when canSend', () => {
      render(<InputSendButton canSend={true} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.keyDown(button, { key: 'Enter' });

      expect(mockOnSend).toHaveBeenCalledTimes(1);
    });

    it('calls onSend on Space key when canSend', () => {
      render(<InputSendButton canSend={true} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.keyDown(button, { key: ' ' });

      expect(mockOnSend).toHaveBeenCalledTimes(1);
    });

    it('does not call onSend on other keys', () => {
      render(<InputSendButton canSend={true} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.keyDown(button, { key: 'Tab' });

      expect(mockOnSend).not.toHaveBeenCalled();
    });

    it('does not call onSend on Enter when canSend is false', () => {
      render(<InputSendButton canSend={false} onSend={mockOnSend} />);

      const button = screen.getByRole('button');
      fireEvent.keyDown(button, { key: 'Enter' });

      expect(mockOnSend).not.toHaveBeenCalled();
    });
  });

  describe('Tooltip', () => {
    it('renders tooltip when tooltipText is provided', () => {
      const { container } = render(
        <InputSendButton tooltipText="Send message" />
      );

      expect(container.textContent).toContain('Send message');
    });

    it('does not render tooltip when tooltipText is null', () => {
      const { container } = render(
        <InputSendButton tooltipText={null} />
      );

      // Only the button content should be present
      const tooltipContent = container.querySelector('.absolute.bottom-full');
      expect(tooltipContent).not.toBeInTheDocument();
    });

    it('does not render tooltip when tooltipText is undefined', () => {
      const { container } = render(<InputSendButton />);

      const tooltipContent = container.querySelector('.absolute.bottom-full');
      expect(tooltipContent).not.toBeInTheDocument();
    });

    it('sets title attribute when tooltipText is provided', () => {
      render(<InputSendButton tooltipText="Send message" />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('title', 'Send message');
    });
  });

  describe('Accessibility', () => {
    it('has correct aria-label when canSend is true', () => {
      render(<InputSendButton canSend={true} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Send message');
    });

    it('has correct aria-label when canSend is false', () => {
      render(<InputSendButton canSend={false} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Send message (disabled - no message to send)');
    });
  });

  describe('Visual Effects', () => {
    it('has shine effect element', () => {
      const { container } = render(<InputSendButton canSend={true} />);

      const shineEffect = container.querySelector('.bg-primary-blue-glow');
      expect(shineEffect).toBeInTheDocument();
    });

    it('icon scales on hover when canSend', () => {
      const { container } = render(<InputSendButton canSend={true} />);

      const svg = container.querySelector('svg');
      expect(svg).toHaveClass('group-hover:scale-110');
    });
  });
});
