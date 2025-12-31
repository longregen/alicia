import React, { useEffect, useState, useRef } from 'react';
import { cls } from '../../utils/cls';
import { MicrophoneStatus } from '../../types/streaming';
import { SileroVADManager } from '../../utils/sileroVAD';

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
  /** Callback when speech segment is detected */
  onSpeechSegment?: (audioData: Float32Array) => void;
  /** Enable Silero VAD integration */
  useSileroVAD?: boolean;
  /** Silero VAD manager instance (passed from parent) */
  vadManager?: SileroVADManager;
  /** Callback when speech probability changes */
  onSpeechProbabilityChange?: (probability: number, isSpeaking: boolean) => void;
}

const MicrophoneVAD: React.FC<MicrophoneVADProps> = ({
  microphoneStatus: externalMicrophoneStatus,
  isSpeaking: externalIsSpeaking = false,
  speechProbability: externalSpeechProbability = 0,
  onClick,
  disabled = false,
  className = '',
  onSpeechSegment,
  useSileroVAD = false,
  vadManager,
  onSpeechProbabilityChange
}) => {
  // Internal state for Silero VAD
  const [internalMicrophoneStatus, setInternalMicrophoneStatus] = useState<MicrophoneStatus>(MicrophoneStatus.Inactive);
  const [internalIsSpeaking, setInternalIsSpeaking] = useState(false);
  const [internalSpeechProbability, setInternalSpeechProbability] = useState(0);
  const [isRecording, setIsRecording] = useState(false);

  // Use external props if provided, otherwise use internal state
  const microphoneStatus = externalMicrophoneStatus ?? internalMicrophoneStatus;
  const isSpeaking = externalIsSpeaking || internalIsSpeaking;
  const speechProbability = externalSpeechProbability || internalSpeechProbability;
  const isActive = microphoneStatus === MicrophoneStatus.Recording;

  // Create refs for direct DOM manipulation
  const ringRefs = useRef<(SVGCircleElement | null)[]>([]);
  const animationFrameRef = useRef<number>(0);
  const ringStatesRef = useRef<Array<{ active: boolean; radius: number; opacity: number }>>(
    Array.from({ length: 10 }, () => ({ active: false, radius: 0, opacity: 0 }))
  );
  const frameCountRef = useRef<number>(0);
  const lastRingProbabilityRef = useRef<number>(0);

  // Setup VAD callbacks when vadManager is provided
  useEffect(() => {
    if (!vadManager || !useSileroVAD) return;

    // Update the callbacks to use our internal state
    vadManager.updateCallbacks({
      onStatusChange: setInternalMicrophoneStatus,
      onSpeechProbability: (probability, speaking) => {
        setInternalSpeechProbability(probability);
        setInternalIsSpeaking(speaking);
        // Note: onSpeechProbabilityChange is intentionally not called here to avoid loops
      },
      onSpeechStart: () => {
        console.log('Speech detected - start');
      },
      onSpeechEnd: (audioData) => {
        console.log('Speech detected - end', audioData.length, 'samples');
        onSpeechSegment?.(audioData);
      },
      onError: (error) => {
        console.error('VAD Error:', error);
      }
    });
  }, [vadManager, useSileroVAD, onSpeechSegment]);

  // Store speech probability in refs to avoid re-renders
  const speechProbabilityRef = useRef(internalSpeechProbability);
  const isSpeakingRef = useRef(internalIsSpeaking);

  useEffect(() => {
    speechProbabilityRef.current = internalSpeechProbability;
    isSpeakingRef.current = internalIsSpeaking;
  }, [internalSpeechProbability, internalIsSpeaking]);

  // Use requestAnimationFrame to update speech probability to avoid React re-renders
  useEffect(() => {
    if (!onSpeechProbabilityChange) return;

    let rafId: number;
    let running = true;

    const updateCallback = () => {
      if (!running) return;

      // Call the callback with current ref values
      onSpeechProbabilityChange(speechProbabilityRef.current, isSpeakingRef.current);
      rafId = requestAnimationFrame(updateCallback);
    };

    rafId = requestAnimationFrame(updateCallback);

    return () => {
      running = false;
      if (rafId) {
        cancelAnimationFrame(rafId);
      }
    };
  }, [onSpeechProbabilityChange]); // Only depend on the callback function

  // Handle click with VAD integration
  const handleClick = async () => {
    if (useSileroVAD) {
      if (!vadManager) {
        console.warn('VAD is enabled but no vadManager provided');
        return;
      }

      try {
        if (isRecording) {
          vadManager.stop();
          setIsRecording(false);
        } else {
          // Initialize if not already done
          if (vadManager.getStatus() === MicrophoneStatus.Inactive) {
            await vadManager.initialize();
          }
          await vadManager.start();
          setIsRecording(true);
        }
      } catch (error) {
        console.error('Failed to toggle VAD recording:', error);
      }
    } else {
      onClick?.();
    }
  };

  useEffect(() => {
    if (!isActive) {
      // Reset all rings
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
  }, [isActive, speechProbability]); // Removed animation params to prevent restarts

  // Determine button state classes
  const getButtonClasses = () => {
    if (disabled) {
      return 'bg-sunken cursor-not-allowed';
    }
    if (microphoneStatus === MicrophoneStatus.Error) {
      return 'bg-error-subtle hover:bg-error border-2 border-error';
    }
    if (isActive) {
      if (isSpeaking) {
        return 'bg-success-subtle hover:bg-success';
      }
      return 'bg-accent-subtle hover:bg-accent';
    }
    return 'bg-sunken hover:bg-surface';
  };

  // Determine icon color
  const getIconColor = () => {
    if (disabled) return 'text-muted';
    if (microphoneStatus === MicrophoneStatus.Error) return 'text-error';
    if (!isActive) return 'text-muted';
    if (isSpeaking) return 'text-success';
    return 'text-accent';
  };

  return (
    <button
      onClick={handleClick}
      disabled={disabled}
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
      aria-label={isActive ? 'Stop recording' : 'Start recording'}
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

      {/* Microphone icon */}
      <svg
        className={cls('w-5 h-5 relative z-10', getIconColor())}
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
