import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { AudioInput } from './AudioInput';

describe('AudioInput', () => {
  let mockMediaStream: MediaStream;
  let mockAudioTrack: MediaStreamTrack;
  let mockGetUserMedia: any;

  beforeEach(() => {
    // Mock AudioContext as a class
    class MockAudioContext {
      createAnalyser = vi.fn().mockReturnValue({
        fftSize: 0,
        frequencyBinCount: 128,
        connect: vi.fn(),
        getByteFrequencyData: vi.fn((arr) => {
          for (let i = 0; i < arr.length; i++) {
            arr[i] = Math.random() * 255;
          }
        }),
      });
      createMediaStreamSource = vi.fn().mockReturnValue({
        connect: vi.fn(),
      });
      close = vi.fn();
    }

    (global as any).AudioContext = MockAudioContext as any;

    // Mock requestAnimationFrame
    (global as any).requestAnimationFrame = vi.fn((cb) => {
      setTimeout(cb, 16);
      return 1;
    }) as any;
    (global as any).cancelAnimationFrame = vi.fn();

    // Mock media stream
    mockAudioTrack = {
      stop: vi.fn(),
      kind: 'audio',
    } as any;

    mockMediaStream = {
      getAudioTracks: vi.fn().mockReturnValue([mockAudioTrack]),
      getTracks: vi.fn().mockReturnValue([mockAudioTrack]),
    } as any;

    mockGetUserMedia = vi.fn().mockResolvedValue(mockMediaStream);

    Object.defineProperty(navigator, 'mediaDevices', {
      writable: true,
      value: {
        getUserMedia: mockGetUserMedia,
      },
    });
  });

  describe('rendering', () => {
    it('should render record button', () => {
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      expect(screen.getByRole('button')).toBeInTheDocument();
      expect(screen.getByTitle('Start recording')).toBeInTheDocument();
    });

    it('should disable button when disabled prop is true', () => {
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={true} />);

      expect(screen.getByRole('button')).toBeDisabled();
    });
  });

  describe('microphone permission handling', () => {
    it('should request microphone permission on record', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      expect(mockGetUserMedia).toHaveBeenCalledWith({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      });

      await waitFor(() => {
        expect(onTrackReady).toHaveBeenCalledWith(mockAudioTrack);
      });
    });

    it('should handle permission denial', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      mockGetUserMedia.mockRejectedValueOnce(new Error('Permission denied'));

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByText('Microphone access denied')).toBeInTheDocument();
        expect(screen.getByText(/Please enable microphone access/)).toBeInTheDocument();
      });

      expect(onTrackReady).not.toHaveBeenCalled();
    });
  });

  describe('recording controls', () => {
    it('should start recording when button is clicked', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });
    });

    it('should stop recording when button is clicked again', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');

      // Start recording
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      // Stop recording
      await user.click(button);

      await waitFor(() => {
        expect(onTrackStop).toHaveBeenCalled();
        expect(mockAudioTrack.stop).toHaveBeenCalled();
      });
    });

    it('should call onTrackReady with audio track', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(onTrackReady).toHaveBeenCalledWith(mockAudioTrack);
      });
    });
  });

  describe('audio level visualization', () => {
    it('should show audio level when recording', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      // Audio level visualizer should be present
      const levelContainer = document.querySelector('.audio-level-container');
      expect(levelContainer).toBeInTheDocument();
    });

    it('should hide audio level when not recording', () => {
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const levelContainer = document.querySelector('.audio-level-container');
      expect(levelContainer).not.toBeInTheDocument();
    });

    it('should update audio level during recording', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      // Wait for animation frame to update level
      await waitFor(() => {
        const levelFill = document.querySelector('.audio-level-fill') as HTMLElement;
        expect(levelFill).toBeInTheDocument();
        expect(levelFill.style.width).not.toBe('0%');
      }, { timeout: 100 });
    });
  });

  describe('cleanup', () => {
    it('should stop tracks on unmount', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      const { unmount } = render(
        <AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />
      );

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      unmount();

      expect(mockAudioTrack.stop).toHaveBeenCalled();
    });

    it('should cancel animation frame on unmount', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      const { unmount } = render(
        <AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />
      );

      const button = screen.getByRole('button');
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      unmount();

      expect((global as any).cancelAnimationFrame).toHaveBeenCalled();
    });

    it('should close audio context on stop', async () => {
      const user = userEvent.setup();
      const onTrackReady = vi.fn();
      const onTrackStop = vi.fn();

      // Spy on close method
      const closeSpy = vi.fn();
      const OriginalAudioContext = (global as any).AudioContext;
      class TrackedAudioContext extends OriginalAudioContext {
        close = closeSpy;
      }
      (global as any).AudioContext = TrackedAudioContext as any;

      render(<AudioInput onTrackReady={onTrackReady} onTrackStop={onTrackStop} disabled={false} />);

      const button = screen.getByRole('button');

      // Start recording
      await user.click(button);

      await waitFor(() => {
        expect(screen.getByTitle('Stop recording')).toBeInTheDocument();
      });

      // Stop recording
      await user.click(button);

      await waitFor(() => {
        expect(closeSpy).toHaveBeenCalled();
      });
    });
  });
});
