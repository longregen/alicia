import React, { useState } from 'react';
import { AUDIO_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, AudioState } from '../../types/components';

// Component props interface
export interface AudioAddonProps extends BaseComponentProps {
  /** Current audio state */
  state?: AudioState;
  /** Play handler */
  onPlay?: () => void;
  /** Pause handler */
  onPause?: () => void;
  /** Stop handler (resets to beginning) */
  onStop?: () => void;
  /** Total duration in seconds */
  duration?: number;
  /** Current playback time in seconds */
  currentTime?: number;
  /** Whether the addon is disabled */
  disabled?: boolean;
  /** Display mode - compact for inline use, full for below */
  mode?: 'compact' | 'full';
}

const AudioAddon: React.FC<AudioAddonProps> = ({
  state = AUDIO_STATES.IDLE,
  onPlay,
  onPause,
  onStop,
  duration = 0,
  currentTime = 0,
  disabled = false,
  mode = 'full',
  className = ''
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Auto-expand when playing, collapse when idle (only in full mode)
  React.useEffect(() => {
    if (mode === 'full') {
      if (state === AUDIO_STATES.PLAYING || state === AUDIO_STATES.PAUSED) {
        setIsExpanded(true);
      } else if (state === AUDIO_STATES.IDLE) {
        setIsExpanded(false);
      }
    }
  }, [state, mode]);

  const handleClick = () => {
    if (disabled) return;

    switch (state) {
      case AUDIO_STATES.IDLE:
        onPlay?.();
        break;
      case AUDIO_STATES.PLAYING:
        onPause?.();
        break;
      case AUDIO_STATES.PAUSED:
        onPlay?.();
        break;
      default:
        break;
    }
  };

  const handleStop = (e: React.MouseEvent) => {
    e.stopPropagation();
    onStop?.();
  };

  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getProgressPercentage = (): number => {
    if (!duration || duration === 0) return 0;
    return (currentTime / duration) * 100;
  };

  const getIcon = (size: 'small' | 'normal' = 'normal') => {
    const sizeClass = size === 'small' ? 'w-3 h-3' : 'w-4 h-4';

    switch (state) {
      case AUDIO_STATES.PLAYING:
        return (
          <svg className={sizeClass} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
          </svg>
        );
      case AUDIO_STATES.LOADING:
        return (
          <div className={cls(sizeClass, 'border-2', 'border-current', 'border-t-transparent', 'rounded-full', 'animate-spin')} />
        );
      default:
        return (
          <svg className={sizeClass} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
          </svg>
        );
    }
  };

  const containerClasses = cls([
    'flex',
    'items-center',
    'gap-2',
    'transition-all',
    'duration-300',
    'ease-in-out',
    isExpanded ? 'w-48' : 'w-auto',
    className
  ]);

  const buttonClasses = cls([
    'w-8',
    'h-8',
    'flex',
    'items-center',
    'justify-center',
    'rounded-full',
    'bg-accent',
    'text-on-emphasis',
    'transition-colors',
    disabled ? 'opacity-50' : 'hover:bg-accent-hover',
    disabled ? 'cursor-not-allowed' : 'cursor-pointer',
  ]);

  // Compact mode: show small play/pause button with duration
  if (mode === 'compact') {
    return (
      <div className={cls(
        'flex',
        'items-center',
        'justify-center',
        'gap-1.5',
        className
      )}>
        <button
          className={cls(
            'w-6 h-6 p-1.5',
            'flex',
            'items-center',
            'justify-center',
            'rounded-full bg-accent-subtle hover:bg-accent',
            'transition-colors',
            'duration-300',
            disabled ? 'opacity-50' : '',
            disabled ? 'cursor-not-allowed' : 'cursor-pointer',
            state === AUDIO_STATES.PLAYING ? 'bg-accent' : ''
          )}
          onClick={handleClick}
          disabled={disabled || state === AUDIO_STATES.LOADING}
          aria-label={state === AUDIO_STATES.PLAYING ? 'Pause audio' : 'Play audio'}
          title={`Audio ${formatTime(duration)} - ${state === AUDIO_STATES.PLAYING ? 'Playing' : state === AUDIO_STATES.PAUSED ? 'Paused' : 'Click to play'}`}
        >
          {getIcon('small')}
        </button>
        <span className={cls(
          'text-xs text-muted',
          state === AUDIO_STATES.PLAYING ? 'text-accent' : ''
        )}>
          { state === AUDIO_STATES.PLAYING ? formatTime(currentTime) + ' / ' : ''}
          {formatTime(duration)}
        </span>
      </div>
    );
  }

  return (
    <div className={containerClasses}>
      <button
        className={buttonClasses}
        onClick={handleClick}
        disabled={disabled || state === AUDIO_STATES.LOADING}
        aria-label={state === AUDIO_STATES.PLAYING ? 'Pause' : 'Play'}
      >
        {getIcon()}
      </button>

      {isExpanded && (
        <div className={cls(
          'flex',
          'items-center',
          'gap-2',
          'flex-1',
          'overflow-hidden'
        )}>
          {/* Progress bar */}
          <div className={cls(
            'flex-1',
            'h-1',
            'bg-surface',
            'rounded-full',
            'relative',
            'overflow-hidden'
          )}>
            <div
              className={cls(
                'absolute',
                'top-0',
                'left-0',
                'h-full',
                'bg-accent',
                'transition-all',
                'duration-100'
              )}
              style={{ width: `${getProgressPercentage()}%` }}
            />
          </div>

          {/* Time display */}
          <div className={cls('text-xs', 'text-muted', 'min-w-[3rem]', 'text-right')}>
            {formatTime(currentTime)} / {formatTime(duration)}
          </div>

          {/* Stop button (only when playing or paused) */}
          {(state === AUDIO_STATES.PLAYING || state === AUDIO_STATES.PAUSED) && (
            <button
              className={cls(
                'w-6',
                'h-6',
                'flex',
                'items-center',
                'justify-center',
                'rounded-full',
                'text-muted',
                'hover:text-default',
                'hover:bg-surface',
                'transition-colors'
              )}
              onClick={handleStop}
              aria-label="Stop"
            >
              <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clipRule="evenodd" />
              </svg>
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default AudioAddon;
