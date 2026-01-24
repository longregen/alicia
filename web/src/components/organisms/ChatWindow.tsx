import React, { useMemo, useState } from 'react';
import { MoreVertical, Volume2, VolumeX, Archive, Trash2, Menu } from 'lucide-react';
import MessageList from './MessageList';
import InputArea from './InputArea';
import ResponseControls from './ResponseControls';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { useVoiceConnectionStore, VoiceConnectionStatus } from '../../stores/voiceConnectionStore';
import { useSidebarStore } from '../../stores/sidebarStore';
import { usePreferences } from '../../hooks/usePreferences';
import { useLiveKit } from '../../hooks/useLiveKit';
import { useWebSocket } from '../../contexts/WebSocketContext';
import { cls } from '../../utils/cls';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '../atoms/DropdownMenu';
import type { MessageId, ConversationId } from '../../types/chat';
import { createConversationId } from '../../types/chat';

export interface ChatWindowProps {
  onSendMessage?: (message: string, isVoice: boolean) => void;
  onStopStreaming?: () => void;
  onRegenerateResponse?: () => void;
  onBranchSwitch?: (targetMessageId: string) => void;
  onRetry?: (messageId: MessageId) => void;
  onArchive?: () => void;
  onDelete?: () => void;
  conversationId?: string | null;
  conversationTitle?: string;
  showControls?: boolean;
  className?: string;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  onSendMessage,
  onStopStreaming,
  onRegenerateResponse,
  onBranchSwitch,
  onRetry,
  onArchive,
  onDelete,
  conversationId = null,
  conversationTitle = 'Conversation',
  showControls = true,
  className = '',
}) => {
  const convId: ConversationId | null = useMemo(
    () => conversationId ? createConversationId(conversationId) : null,
    [conversationId]
  );

  const [voiceActive, setVoiceActive] = useState(false);
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;
  const { audio_output_enabled: audioOutputEnabled, updatePreference } = usePreferences();
  const toggleAudioOutput = () => updatePreference('audio_output_enabled', !audioOutputEnabled);
  const openSidebar = useSidebarStore((state) => state.setOpen);

  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
  } = useLiveKit(voiceActive ? conversationId : null, { audioOutputEnabled });

  const { retryVoiceJoin } = useWebSocket();
  const voiceConnectionStatus = useVoiceConnectionStore((state) => state.status);
  const voiceConnectionError = useVoiceConnectionStore((state) => state.error);
  const voiceRetryCount = useVoiceConnectionStore((state) => state.retryCount);

  const getConnectionStatusText = () => {
    if (voiceActive) {
      if (voiceConnectionStatus === VoiceConnectionStatus.Retrying) {
        return `Voice connection retrying (attempt ${voiceRetryCount}/3)...`;
      }
      if (voiceConnectionStatus === VoiceConnectionStatus.Error) {
        return voiceConnectionError || 'Voice connection failed';
      }
      if (voiceConnectionStatus === VoiceConnectionStatus.Connecting) {
        return 'Connecting voice...';
      }
      if (voiceConnectionStatus === VoiceConnectionStatus.Connected && liveKitConnected) {
        return null;
      }
      if (liveKitConnected) return null;
      if (liveKitError) return 'Voice error';
      return 'Connecting voice...';
    }
    switch (connectionStatus) {
      case ConnectionStatus.Connecting: return 'Connecting';
      case ConnectionStatus.Reconnecting: return 'Reconnecting';
      case ConnectionStatus.Disconnected: return 'Disconnected';
      case ConnectionStatus.Connected: return null;
      case ConnectionStatus.Error: return 'Connection error';
      default: return null;
    }
  };

  const connectionStatusText = getConnectionStatusText();
  const showConnectionStatus = connectionStatusText !== null;

  return (
    <div className={cls('stack h-full bg-background', className)}>
      {/* Header */}
      <header className="h-14 flex-between px-4 border-b border-border shrink-0">
        <div className="row-3">
          <button
            onClick={() => openSidebar(true)}
            className="lg:hidden p-2 -ml-2 hover:bg-elevated rounded-md transition-colors"
            aria-label="Open sidebar"
          >
            <Menu className="w-6 h-6 text-default" />
          </button>
          <h2 className="font-medium text-foreground">{conversationTitle}</h2>
          <span className="badge badge-default">{conversationId || 'No ID'}</span>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className="btn-icon"
              aria-label="Conversation options"
            >
              <MoreVertical className="w-5 h-5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuCheckboxItem
              checked={audioOutputEnabled}
              onCheckedChange={toggleAudioOutput}
            >
              {audioOutputEnabled ? (
                <Volume2 className="w-4 h-4" />
              ) : (
                <VolumeX className="w-4 h-4 text-muted-foreground" />
              )}
              {audioOutputEnabled ? 'Speak responses' : 'Responses muted'}
            </DropdownMenuCheckboxItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onArchive} disabled={!onArchive}>
              <Archive className="w-4 h-4" />
              Archive conversation
            </DropdownMenuItem>
            <DropdownMenuItem
              variant="destructive"
              onClick={onDelete}
              disabled={!onDelete}
            >
              <Trash2 className="w-4 h-4" />
              Delete conversation
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </header>

      {showConnectionStatus && (
        <div className={cls(
          'status-bar',
          (voiceActive && voiceConnectionStatus === VoiceConnectionStatus.Error) || (voiceActive && liveKitError)
            ? 'status-bar-error'
            : voiceActive && voiceConnectionStatus === VoiceConnectionStatus.Retrying
              ? 'status-bar-warning'
              : voiceActive && liveKitConnected
                ? 'status-bar-success'
                : 'status-bar-warning'
        )}>
          <svg className="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <span>{connectionStatusText}</span>
          {voiceActive && voiceConnectionStatus === VoiceConnectionStatus.Error && (
            <button
              onClick={retryVoiceJoin}
              className="ml-2 px-2 py-0.5 text-xs font-medium rounded bg-white/20 hover:bg-white/30 transition-colors"
            >
              Retry
            </button>
          )}
        </div>
      )}

      <div className="flex-1 overflow-hidden">
        <MessageList conversationId={convId} onBranchSwitch={onBranchSwitch} onRetry={onRetry} />
      </div>

      {showControls && (
        <ResponseControls
          conversationId={convId}
          onStop={onStopStreaming}
          onRegenerate={onRegenerateResponse}
          show={isConnected}
        />
      )}

      <InputArea
        onSend={onSendMessage}
        onPublishAudioTrack={voiceActive ? publishAudioTrack : undefined}
        onVoiceActiveChange={setVoiceActive}
        voiceActive={voiceActive}
        disabled={!isConnected}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        conversationId={conversationId}
      />
    </div>
  );
};

export default ChatWindow;
