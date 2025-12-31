import React, { useState, useEffect } from 'react';
import AudioAddon from './AudioAddon';
import FeedbackControls from './FeedbackControls';
import { AUDIO_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { flexCenterGap, uiPatterns } from '../../utils/uiPatterns';
import { useFeedback } from '../../hooks/useFeedback';
import type { BaseComponentProps, MessageAddon, AudioState } from '../../types/components';

// Tool details interface
export interface ToolDetail {
  id: string;
  name: string;
  description: string;
  result?: string;
  status?: 'pending' | 'running' | 'completed' | 'error';
}

// Separate component for tool details panel with feedback controls
// This allows us to use hooks inside the expanded panel
interface ToolDetailsPanelProps {
  toolDetail: ToolDetail;
  showFeedback?: boolean;
}

const ToolDetailsPanel: React.FC<ToolDetailsPanelProps> = ({ toolDetail, showFeedback = false }) => {
  // useFeedback hook for tool voting
  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('tool_use', toolDetail.id);

  return (
    <div className={cls(
      'transition-all duration-300 ease-in-out overflow-hidden',
      'bg-surface rounded-lg p-4 mt-3 border border-border',
      'max-h-64 opacity-100'
    )}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <div className="text-sm font-medium text-default">{toolDetail.name}</div>
          <div className="text-xs text-muted mt-1">{toolDetail.description}</div>
        </div>
      </div>

      {toolDetail.result && (
        <div className="text-xs text-success mt-2">✓ {toolDetail.result}</div>
      )}

      {toolDetail.status === 'running' && (
        <div className="text-xs text-accent mt-2 flex items-center gap-1">
          <div className="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin" />
          Running...
        </div>
      )}

      {toolDetail.status === 'pending' && (
        <div className="text-xs text-warning mt-2">⏳ Pending...</div>
      )}

      {toolDetail.status === 'error' && (
        <div className="text-xs text-error mt-2">❌ Error occurred</div>
      )}

      {/* Feedback controls for completed tools */}
      {showFeedback && toolDetail.status === 'completed' && (
        <div className="mt-3 pt-3 border-t border-border-muted">
          <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
          <FeedbackControls
            currentVote={currentVote as 'up' | 'down' | null}
            onVote={vote}
            upvotes={counts.up}
            downvotes={counts.down}
            isLoading={feedbackLoading}
            compact
          />
        </div>
      )}
    </div>
  );
};

// Component props interface
export interface ComplexAddonsProps extends BaseComponentProps {
  /** Array of addons to display */
  addons: MessageAddon[];
  /** Tool details for expandable tools */
  toolDetails?: ToolDetail[];
  /** Message timestamp */
  timestamp: Date;
  /** Whether to show feedback controls (default: false) */
  showFeedback?: boolean;
}

const ComplexAddons: React.FC<ComplexAddonsProps> = ({
  addons,
  toolDetails = [],
  timestamp,
  className = '',
  showFeedback = false,
}) => {
  const [expandedToolId, setExpandedToolId] = useState<string | null>(null);
  const [hoveredAddonId, setHoveredAddonId] = useState<string | null>(null);
  const [audioState, setAudioState] = useState<AudioState>(AUDIO_STATES.IDLE);
  const [audioCurrentTime, setAudioCurrentTime] = useState(0);

  // Don't separate audio addons - treat them like any other addon
  // const audioAddon = addons.find(addon => addon.type === 'audio');
  // const nonAudioAddons = addons.filter(addon => addon.type !== 'audio');

  // Mock audio duration - in a real app this would come from the audio file
  const audioDuration = 45; // seconds

  // Simple audio simulation for demo
  useEffect(() => {
    if (audioState === AUDIO_STATES.PLAYING) {
      const interval = setInterval(() => {
        setAudioCurrentTime(prev => {
          if (prev >= audioDuration) {
            setAudioState(AUDIO_STATES.IDLE);
            return 0;
          }
          return prev + 0.1;
        });
      }, 100);
      return () => clearInterval(interval);
    }
  }, [audioState, audioDuration]);

  const getToolDetail = (addonId: string): ToolDetail | undefined => {
    return toolDetails.find(tool => tool.id === addonId);
  };

  const getAddonAnimation = (_addon: MessageAddon, toolDetail?: ToolDetail) => {
    if (toolDetail?.status === 'running') {
      return `${uiPatterns.pulseAnimation} scale-110`;
    }
    if (toolDetail?.status === 'pending') {
      return `${uiPatterns.pulseAnimation} opacity-70`;
    }
    if (toolDetail?.status === 'error') {
      return 'text-error';
    }
    return '';
  };

  const renderAddon = (addon: MessageAddon) => {
    // Special rendering for audio addons
    if (addon.type === 'audio') {
      return (
        <AudioAddon
          key={addon.id}
          mode="compact"
          state={audioState}
          onPlay={() => setAudioState(AUDIO_STATES.PLAYING)}
          onPause={() => setAudioState(AUDIO_STATES.PAUSED)}
          onStop={() => {
            setAudioState(AUDIO_STATES.IDLE);
            setAudioCurrentTime(0);
          }}
          duration={audioDuration}
          currentTime={audioCurrentTime}
        />
      );
    }

    // Default rendering for other addon types
    const toolDetail = getToolDetail(addon.id);
    const isHovered = hoveredAddonId === addon.id;
    const isExpanded = expandedToolId === addon.id;

    return (
      <div key={addon.id} className="relative">
        <button
          className={cls(
            'relative w-6 h-6 flex items-center justify-center',
            'text-sm cursor-pointer transition-all duration-200',
            'hover:scale-110 hover:bg-surface rounded-full',
            getAddonAnimation(addon, toolDetail),
            isExpanded ? 'scale-110 bg-accent-subtle' : ''
          )}
          onMouseEnter={() => setHoveredAddonId(addon.id)}
          onMouseLeave={() => setHoveredAddonId(null)}
          onClick={() => {
            if (addon.type === 'tool' || addon.type === 'icon') {
              setExpandedToolId(expandedToolId === addon.id ? null : addon.id);
            }
          }}
          title={addon.tooltip}
        >
          {addon.emoji}

          {/* Status indicator for running tools */}
          {toolDetail?.status === 'running' && (
            <div className={cls("absolute -bottom-1 -right-1 w-2 h-2 bg-accent rounded-full", uiPatterns.pulseAnimation)} />
          )}
        </button>

        {/* Tooltip */}
        {isHovered && !isExpanded && (
          <div className={cls(
            'absolute z-20 transition-all duration-200',
            'bg-overlay backdrop-blur-sm text-on-emphasis text-xs rounded-md p-3 min-w-[16rem] max-w-[20rem]',
            'pointer-events-none shadow-xl border border-border-emphasis',
            'top-full left-1/2 transform -translate-x-1/2 mt-2'
          )}>
            <div className="font-semibold text-sm">{toolDetail?.name || addon.tooltip}</div>
            {toolDetail?.description && (
              <div className="text-subtle mt-1.5 text-xs leading-relaxed">
                {toolDetail.description}
              </div>
            )}

            {/* Status indicator */}
            {toolDetail?.status && (
              <div className={cls(
                'mt-2 text-[11px] font-medium',
                toolDetail.status === 'running' ? 'text-accent' : '',
                toolDetail.status === 'completed' ? 'text-success' : '',
                toolDetail.status === 'error' ? 'text-error' : ''
              )}>
                {toolDetail.status === 'running' && '⚡ Currently running...'}
                {toolDetail.status === 'completed' && '✓ Completed'}
                {toolDetail.status === 'error' && '⚠️ Error occurred'}
              </div>
            )}

            {/* Tooltip arrow */}
            <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-b-overlay" />
          </div>
        )}
      </div>
    );
  };

  const renderToolDetails = () => {
    if (!expandedToolId) return null;

    const toolDetail = getToolDetail(expandedToolId);
    if (!toolDetail) return null;

    return (
      <ToolDetailsPanel
        toolDetail={toolDetail}
        showFeedback={showFeedback}
      />
    );
  };

  return (
    <div className={cls('space-y-2 w-full', className)}>
      {/* Main addon row */}
      <div className="flex items-center justify-between w-full">
        {/* Left: All addons inline */}
        <div className={flexCenterGap(2)}>
          {/* All addons */}
          {addons.map(renderAddon)}
        </div>

        {/* Right: Timestamp */}
        <div className="text-xs text-muted">
          {timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </div>
      </div>

      {/* Tool Details */}
      {renderToolDetails()}
    </div>
  );
};

export default ComplexAddons;
