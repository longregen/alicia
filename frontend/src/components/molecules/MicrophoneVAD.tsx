import React, { useEffect, useRef } from 'react';
import { cls } from '../../utils/cls';
import { MicrophoneStatus } from '../../types/streaming';

export interface MicrophoneVADProps {
  /** Current microphone status */
  microphoneStatus?: MicrophoneStatus;
  /** Whether speech is currently detected */
  isSpeaking?: boolean;
  /** Speech probability (0-1) for visualization */
  speechProbability?: number;
  /** Click handler for the button */
  onClick?: () => void;
  /** Whether the button is disabled */
  disabled?: boolean;
  /** Additional CSS classes */
  className?: string;
  /** Callback to start VAD (from useVAD hook) */
  onStartVAD?: () => Promise<void>;
  /** Callback to stop VAD (from useVAD hook) */
  onStopVAD?: () => void;
}

const MicrophoneVAD: React.FC<MicrophoneVADProps> = ({
  microphoneStatus = MicrophoneStatus.Inactive,
  isSpeaking = false,
  speechProbability = 0,
  onClick,
  disabled = false,
  className = '',
  onStartVAD,
  onStopVAD,
}) => {
  const isActive = microphoneStatus === MicrophoneStatus.Recording || microphoneStatus === MicrophoneStatus.Sending;
  const isLoading = microphoneStatus === MicrophoneStatus.Loading || microphoneStatus === MicrophoneStatus.RequestingPermission;
  const isReady = microphoneStatus === MicrophoneStatus.Active;
  const isError = microphoneStatus === MicrophoneStatus.Error;
  const isSending = microphoneStatus === MicrophoneStatus.Sending;

  // Create refs for direct DOM manipulation (rings animation)
  const ringRefs = useRef<(SVGCircleElement | null)[]>([]);
  const animationFrameRef = useRef<number>(0);
  const ringStatesRef = useRef<Array<{ active: boolean; radius: number; opacity: number }>>(
    Array.from({ length: 10 }, () => ({ active: false, radius: 0, opacity: 0 }))
  );
  const frameCountRef = useRef<number>(0);
  const lastRingProbabilityRef = useRef<number>(0);

  // Handle click with VAD integration
  const handleClick = async () => {
    // Always call external onClick first (for parent to manage state)
    onClick?.();

    try {
      if (isActive) {
        onStopVAD?.();
      } else {
        await onStartVAD?.();
      }
    } catch (error) {
      console.error('Failed to toggle VAD recording:', error);
    }
  };

  // Rings animation effect
  useEffect(() => {
    if (!isActive || isSending) {
      // Reset all rings when not active or sending
      ringStatesRef.current.forEach((ring, i) => {
        ring.active = false;
        const circleRef = ringRefs.current[i];
        if (circleRef) {
          circleRef.setAttribute('r', '0');
          circleRef.setAttribute('opacity', '0');
        }
      });
      frameCountRef.current = 0;
      lastRingProbabilityRef.current = 0;
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      return;
    }

    // Animation constants
    const RING_RADIUS_INCREMENT = 0.4;
    const RING_OPACITY_DECAY = 0.96;
    const RING_MAX_RADIUS = 21;

    const animate = () => {
      const rings = ringStatesRef.current;
      frameCountRef.current++;

      // Add new ring every 5 frames if speech detected
      if (frameCountRef.current % 5 === 0 && speechProbability > 0) {
        // Logarithmic thresholds: 100, 98, 95, 90, 75, 50, 0
        const thresholds = [1.0, 0.98, 0.95, 0.90, 0.75, 0.50, 0];
        const lastThreshold = thresholds.find(t => lastRingProbabilityRef.current >= t) || 0;
        const currentThreshold = thresholds.find(t => speechProbability >= t) || 0;

        // Only create ring if we crossed a threshold or it's been 5 frames
        if (currentThreshold !== lastThreshold || speechProbability > 0.5) {
          const inactive = rings.find(r => !r.active);
          if (inactive) {
            inactive.active = true;
            inactive.radius = 0;
            // Use sqrt for more visible rings at lower probabilities
            inactive.opacity = Math.sqrt(speechProbability);
          }
          lastRingProbabilityRef.current = speechProbability;
        }
      }

      // Update all rings
      rings.forEach((ring, i) => {
        if (!ring.active) return;

        // Update physics
        ring.radius += RING_RADIUS_INCREMENT;
        ring.opacity *= RING_OPACITY_DECAY;

        // Update DOM
        const circleRef = ringRefs.current[i];
        if (circleRef) {
          circleRef.setAttribute('r', String(ring.radius));
          circleRef.setAttribute('opacity', String(ring.opacity));
        }

        // Deactivate if invisible or too large
        if (ring.opacity < 0.01 || ring.radius > RING_MAX_RADIUS) {
          ring.active = false;
        }
      });

      animationFrameRef.current = requestAnimationFrame(animate);
    };

    animationFrameRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [isActive, isSending, speechProbability]);

  // Determine button state classes
  const getButtonClasses = () => {
    if (disabled) {
      return 'bg-sunken cursor-not-allowed';
    }
    if (isError) {
      return 'bg-error-subtle hover:bg-error border-2 border-error';
    }
    if (isLoading) {
      return 'bg-accent-subtle animate-pulse cursor-wait';
    }
    if (isSending) {
      return 'bg-accent-subtle animate-pulse';
    }
    if (isActive) {
      if (isSpeaking) {
        return 'bg-success-subtle hover:bg-success';
      }
      return 'bg-accent-subtle hover:bg-accent';
    }
    if (isReady) {
      return 'bg-accent-subtle hover:bg-accent ring-2 ring-accent/30';
    }
    // Inactive
    return 'bg-sunken hover:bg-surface';
  };

  // Determine icon color
  const getIconColor = () => {
    if (disabled) return 'text-muted';
    if (isError) return 'text-error';
    if (isLoading) return 'text-accent';
    if (isSending) return 'text-accent';
    if (!isActive && !isReady) return 'text-muted';
    if (isSpeaking) return 'text-success';
    return 'text-accent';
  };

  // Get aria label based on state
  const getAriaLabel = () => {
    if (disabled) return 'Microphone disabled';
    if (isError) return 'Microphone error';
    if (isLoading) return 'Loading microphone...';
    if (isSending) return 'Sending speech...';
    if (isActive) return 'Stop recording';
    if (isReady) return 'Start recording (ready)';
    return 'Start recording';
  };

  return (
    <button
      onClick={handleClick}
      disabled={disabled || isLoading}
      className={cls(
        // Base button styles matching send button
        'w-10 h-10',
        'rounded-full',
        'transition-all duration-200',
        'flex items-center justify-center',
        'relative overflow-hidden',
        'focus:outline-none',
        'active:scale-95',
        // State-based styles
        getButtonClasses(),
        // Custom classes
        className
      )}
      aria-label={getAriaLabel()}
    >
      {/* Growing rings animation */}
      <svg
        width="40"
        height="40"
        viewBox="0 0 40 40"
        className="absolute inset-0"
      >
        {/* Pre-allocated rings for zero-allocation animation */}
        {Array.from({ length: 10 }, (_, i) => (
          <circle
            key={i}
            ref={el => { ringRefs.current[i] = el; }}
            cx="20"
            cy="20"
            r="0"
            fill="none"
            stroke={isSpeaking ? 'var(--color-success)' : 'var(--color-accent)'}
            strokeWidth="1.5"
            opacity="0"
          />
        ))}
      </svg>

      {/* Loading spinner overlay */}
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center">
          <svg className="animate-spin h-5 w-5 text-accent" viewBox="0 0 24 24">
            <circle
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="3"
              fill="none"
              opacity="0.25"
            />
            <path
              d="M12 2a10 10 0 0 1 10 10"
              stroke="currentColor"
              strokeWidth="3"
              fill="none"
              strokeLinecap="round"
            />
          </svg>
        </div>
      )}

      {/* Sending indicator overlay */}
      {isSending && (
        <div className="absolute inset-0 flex items-center justify-center">
          <svg className="h-3 w-3 text-accent animate-bounce" viewBox="0 0 24 24" fill="currentColor">
            <path d="M4 12l1.41 1.41L11 7.83V20h2V7.83l5.59 5.58L20 12l-8-8-8 8z"/>
          </svg>
        </div>
      )}

      {/* Microphone icon */}
      <svg
        className={cls(
          'w-5 h-5 relative z-10 transition-opacity',
          getIconColor(),
          (isLoading || isSending) && 'opacity-50'
        )}
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"
        />
      </svg>

    </button>
  );
};

export default React.memo(MicrophoneVAD);
