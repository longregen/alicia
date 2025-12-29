import React, { useState, useEffect } from 'react';
import AudioAddon from './AudioAddon';
import { AUDIO_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { flexCenterGap, uiPatterns } from '../../utils/uiPatterns';
import type { BaseComponentProps, MessageAddon, AudioState } from '../../types/components';

// Tool details interface
export interface ToolDetail {
  id: string;
  name: string;
  description: string;
  result?: string;
  status?: 'pending' | 'running' | 'completed' | 'error';
}

// Component props interface
export interface ComplexAddonsProps extends BaseComponentProps {
  /** Array of addons to display */
  addons: MessageAddon[];
  /** Tool details for expandable tools */
  toolDetails?: ToolDetail[];
  /** Message timestamp */
  timestamp: Date;
}

const ComplexAddons: React.FC<ComplexAddonsProps> = ({
  addons,
  toolDetails = [],
  timestamp,
  className = ''
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
      return 'text-red-500';
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
            'hover:scale-110 hover:bg-surface-bg/50 rounded-full',
            getAddonAnimation(addon, toolDetail),
            isExpanded ? 'scale-110 bg-primary-blue/20' : ''
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
            <div className={cls("absolute -bottom-1 -right-1 w-2 h-2 bg-primary-blue rounded-full", uiPatterns.pulseAnimation)} />
          )}
        </button>

        {/* Tooltip */}
        {isHovered && !isExpanded && (
          <div className={cls(
            'absolute z-20 transition-all duration-200',
            'bg-gray-900/95 backdrop-blur-sm text-white text-xs rounded-md p-3 min-w-[16rem] max-w-[20rem]',
            'pointer-events-none shadow-xl border border-gray-700/50',
            'top-full left-1/2 transform -translate-x-1/2 mt-2'
          )}>
            <div className="font-semibold text-sm">{toolDetail?.name || addon.tooltip}</div>
            {toolDetail?.description && (
              <div className="text-gray-300 mt-1.5 text-xs leading-relaxed">
                {toolDetail.description}
              </div>
            )}

            {/* Status indicator */}
            {toolDetail?.status && (
              <div className={cls(
                'mt-2 text-[11px] font-medium',
                toolDetail.status === 'running' ? 'text-blue-400' : '',
                toolDetail.status === 'completed' ? 'text-green-400' : '',
                toolDetail.status === 'error' ? 'text-red-400' : ''
              )}>
                {toolDetail.status === 'running' && '⚡ Currently running...'}
                {toolDetail.status === 'completed' && '✓ Completed'}
                {toolDetail.status === 'error' && '⚠️ Error occurred'}
              </div>
            )}

            {/* Tooltip arrow */}
            <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-b-gray-900/95" />
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
      <div className={cls(
        'transition-all duration-300 ease-in-out overflow-hidden',
        'bg-surface-bg/50 rounded-lg p-4 mt-3 border border-surface-700',
        'max-h-48 opacity-100'
      )}>
        <div className="text-sm font-medium text-primary-text">{toolDetail.name}</div>
        <div className="text-xs text-muted-text mt-1">{toolDetail.description}</div>

        {toolDetail.result && (
          <div className="text-xs text-green-400 mt-2">✓ {toolDetail.result}</div>
        )}

        {toolDetail.status === 'running' && (
          <div className="text-xs text-primary-blue mt-2 flex items-center gap-1">
            <div className="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin" />
            Running...
          </div>
        )}

        {toolDetail.status === 'pending' && (
          <div className="text-xs text-yellow-500 mt-2">⏳ Pending...</div>
        )}

        {toolDetail.status === 'error' && (
          <div className="text-xs text-red-500 mt-2">❌ Error occurred</div>
        )}
      </div>
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
        <div className="text-xs text-muted-text">
          {timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </div>
      </div>

      {/* Tool Details */}
      {renderToolDetails()}
    </div>
  );
};

export default ComplexAddons;
