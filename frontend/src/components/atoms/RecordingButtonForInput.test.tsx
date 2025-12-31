import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import RecordingButtonForInput from './RecordingButtonForInput';
import { RECORDING_STATES } from '../../mockData';

describe('RecordingButtonForInput', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders a button', () => {
      render(<RecordingButtonForInput />);

      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('renders with type button', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'button');
    });

    it('applies custom className', () => {
      render(<RecordingButtonForInput className="custom-class" />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('custom-class');
    });

    it('renders idle state by default', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-pressed', 'false');
      expect(button).toHaveAttribute('aria-label', 'Start recording');
    });
  });

  describe('State Rendering', () => {
    it('renders idle state correctly', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.IDLE} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Start recording');
      expect(button).toHaveAttribute('aria-pressed', 'false');
    });

    it('renders recording state correctly', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.RECORDING} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Stop recording');
      expect(button).toHaveAttribute('aria-pressed', 'true');
      expect(button).toHaveClass('bg-success');
    });

    it('renders processing state correctly', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.PROCESSING} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Processing audio...');
      expect(button).toBeDisabled();
      expect(button).toHaveClass('bg-accent');
    });

    it('renders error state correctly', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.ERROR} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Recording failed - click to retry');
      expect(button).toHaveClass('bg-error');
    });

    it('renders completed state correctly', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.COMPLETED} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label', 'Recording completed');
      expect(button).toHaveClass('bg-success');
    });

    it('shows ripple effect when recording', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.RECORDING} />
      );

      const ripple = container.querySelector('.animate-ping');
      expect(ripple).toBeInTheDocument();
    });

    it('does not show ripple effect when not recording', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.IDLE} />
      );

      const ripple = container.querySelector('.animate-ping');
      expect(ripple).not.toBeInTheDocument();
    });
  });

  describe('Click Interactions', () => {
    it('calls onClick when provided', () => {
      const onClick = vi.fn();
      render(<RecordingButtonForInput onClick={onClick} />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it('calls onToggleRecording with RECORDING when idle and clicked', () => {
      const onToggleRecording = vi.fn();
      render(
        <RecordingButtonForInput
          state={RECORDING_STATES.IDLE}
          onToggleRecording={onToggleRecording}
        />
      );

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onToggleRecording).toHaveBeenCalledWith(RECORDING_STATES.RECORDING);
    });

    it('calls onToggleRecording with IDLE when recording and clicked', () => {
      const onToggleRecording = vi.fn();
      render(
        <RecordingButtonForInput
          state={RECORDING_STATES.RECORDING}
          onToggleRecording={onToggleRecording}
        />
      );

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onToggleRecording).toHaveBeenCalledWith(RECORDING_STATES.IDLE);
    });

    it('prefers onClick over onToggleRecording', () => {
      const onClick = vi.fn();
      const onToggleRecording = vi.fn();
      render(
        <RecordingButtonForInput
          onClick={onClick}
          onToggleRecording={onToggleRecording}
        />
      );

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onClick).toHaveBeenCalledTimes(1);
      expect(onToggleRecording).not.toHaveBeenCalled();
    });

    it('does not call handlers when disabled', () => {
      const onClick = vi.fn();
      render(<RecordingButtonForInput onClick={onClick} disabled />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onClick).not.toHaveBeenCalled();
    });

    it('does not call handlers when processing', () => {
      const onClick = vi.fn();
      render(
        <RecordingButtonForInput
          state={RECORDING_STATES.PROCESSING}
          onClick={onClick}
        />
      );

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe('Mouse Interactions', () => {
    it('applies pressed styling on mouse down', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);

      expect(button).toHaveClass('scale-95');
    });

    it('removes pressed styling on mouse up', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);
      fireEvent.mouseUp(button);

      expect(button).not.toHaveClass('scale-95');
    });

    it('removes pressed styling on mouse leave', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);
      fireEvent.mouseLeave(button);

      expect(button).not.toHaveClass('scale-95');
    });

    it('does not apply pressed styling when disabled', () => {
      render(<RecordingButtonForInput disabled />);

      const button = screen.getByRole('button');
      fireEvent.mouseDown(button);

      expect(button).not.toHaveClass('scale-95');
    });
  });

  describe('Disabled State', () => {
    it('is disabled when disabled prop is true', () => {
      render(<RecordingButtonForInput disabled />);

      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });

    it('applies disabled styling', () => {
      render(<RecordingButtonForInput disabled />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('cursor-not-allowed');
      expect(button).toHaveClass('bg-sunken');
    });

    it('is disabled during processing state', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.PROCESSING} />);

      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
    });
  });

  describe('Size Variants', () => {
    it('renders small size', () => {
      render(<RecordingButtonForInput size="sm" />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('w-8');
      expect(button).toHaveClass('h-8');
    });

    it('renders medium size by default', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('w-10');
      expect(button).toHaveClass('h-10');
    });

    it('renders large size', () => {
      render(<RecordingButtonForInput size="lg" />);

      const button = screen.getByRole('button');
      expect(button).toHaveClass('w-12');
      expect(button).toHaveClass('h-12');
    });
  });

  describe('Tooltip', () => {
    it('shows tooltip when showTooltip is true', () => {
      const { container } = render(
        <RecordingButtonForInput showTooltip={true} />
      );

      expect(container.textContent).toContain('Start recording');
    });

    it('hides tooltip when showTooltip is false', () => {
      const { container } = render(
        <RecordingButtonForInput showTooltip={false} />
      );

      // Tooltip element should not exist
      const tooltip = container.querySelector('.absolute.bottom-full');
      expect(tooltip).not.toBeInTheDocument();
    });

    it('shows correct tooltip text for each state', () => {
      const tooltipTexts = [
        { state: RECORDING_STATES.IDLE, text: 'Start recording' },
        { state: RECORDING_STATES.RECORDING, text: 'Stop recording' },
        { state: RECORDING_STATES.PROCESSING, text: 'Processing audio...' },
        { state: RECORDING_STATES.ERROR, text: 'Recording failed - click to retry' },
        { state: RECORDING_STATES.COMPLETED, text: 'Recording completed' },
      ];

      tooltipTexts.forEach(({ state, text }) => {
        const { unmount } = render(
          <RecordingButtonForInput state={state} showTooltip={true} />
        );

        expect(screen.getByTitle(text)).toBeInTheDocument();
        unmount();
      });
    });
  });

  describe('Accessibility', () => {
    it('has aria-pressed attribute', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-pressed');
    });

    it('aria-pressed is true when recording', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.RECORDING} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-pressed', 'true');
    });

    it('aria-pressed is false when not recording', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.IDLE} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-pressed', 'false');
    });

    it('has aria-label attribute', () => {
      render(<RecordingButtonForInput />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-label');
    });

    it('has title attribute matching aria-label', () => {
      render(<RecordingButtonForInput state={RECORDING_STATES.RECORDING} />);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('title', 'Stop recording');
      expect(button).toHaveAttribute('aria-label', 'Stop recording');
    });
  });

  describe('Icon Rendering', () => {
    it('renders microphone icon in idle state', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.IDLE} />
      );

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('renders pulsing dot in recording state', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.RECORDING} />
      );

      const pulsingDot = container.querySelector('.animate-pulse');
      expect(pulsingDot).toBeInTheDocument();
    });

    it('renders bouncing dots in processing state', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.PROCESSING} />
      );

      const bouncingDots = container.querySelectorAll('.animate-bounce');
      expect(bouncingDots).toHaveLength(3);
    });

    it('renders warning icon in error state', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.ERROR} />
      );

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('renders checkmark icon in completed state', () => {
      const { container } = render(
        <RecordingButtonForInput state={RECORDING_STATES.COMPLETED} />
      );

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });
  });
});
