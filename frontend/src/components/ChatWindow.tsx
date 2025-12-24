import { useState, useEffect } from 'react';
import { MessageList } from './MessageList';
import { InputBar } from './InputBar';
import { AudioInput } from './AudioInput';
import { AudioOutput } from './AudioOutput';
import { ProtocolDisplay } from './ProtocolDisplay';
import { ErrorNotification } from './ErrorNotification';
import { VoiceSelector } from './VoiceSelector';
import { ResponseControls } from './ResponseControls';
import { Message, Conversation } from '../types/models';
import { useLiveKit } from '../hooks/useLiveKit';
import { useMessageContext } from '../contexts/MessageContext';
import { api } from '../services/api';
import { storage } from '../utils/storage';

// Helper function to format relative time
function formatRelativeTime(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

  if (seconds < 10) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

interface ChatWindowProps {
  messages: Message[];
  loading: boolean;
  sending: boolean;
  onSendMessage: (content: string) => void;
  conversationId: string | null;
  conversation: Conversation | null;
  onConversationUpdate?: (conversation: Conversation) => void;
  // Sync props
  isSyncing?: boolean;
  lastSyncTime?: Date | null;
  syncError?: string | null;
}

export function ChatWindow({
  messages,
  loading,
  sending,
  onSendMessage,
  conversationId,
  conversation,
  onConversationUpdate,
  isSyncing = false,
  lastSyncTime = null,
  syncError = null,
}: ChatWindowProps) {
  const [voiceMode, setVoiceMode] = useState(() => storage.getVoiceMode());

  // Get streaming state from unified message context
  const { streamingMessages, currentTranscription } = useMessageContext();

  // Persist voice mode preference
  useEffect(() => {
    storage.setVoiceMode(voiceMode);
  }, [voiceMode]);

  const {
    room,
    connected,
    connectionState,
    error: liveKitError,
    sendMessage: sendLiveKitMessage,
    sendStop,
    sendRegenerate,
    publishAudioTrack,
    unpublishAudioTrack,
  } = useLiveKit(voiceMode ? conversationId : null);

  // Convert streaming sentences map to array for display
  const streamingSentences = Array.from(streamingMessages.values()).filter(Boolean);

  const handleVoiceModeToggle = () => {
    setVoiceMode(!voiceMode);
  };

  const handleSendMessage = (content: string) => {
    if (voiceMode && connected) {
      sendLiveKitMessage(content);
    } else {
      onSendMessage(content);
    }
  };

  const handleVoiceChange = async (voice: string) => {
    if (!conversationId || !conversation) return;

    try {
      const updatedConversation = await api.updateConversation(conversationId, {
        preferences: {
          ...conversation.preferences,
          tts_voice: voice,
        },
      });

      if (onConversationUpdate) {
        onConversationUpdate(updatedConversation);
      }
    } catch (error) {
      console.error('Failed to update voice:', error);
    }
  };

  const handleSpeedChange = async (speed: number) => {
    if (!conversationId || !conversation) return;

    try {
      const updatedConversation = await api.updateConversation(conversationId, {
        preferences: {
          ...conversation.preferences,
          tts_speed: speed,
        },
      });

      if (onConversationUpdate) {
        onConversationUpdate(updatedConversation);
      }
    } catch (error) {
      console.error('Failed to update speed:', error);
    }
  };

  return (
    <div className="chat-window">
      {/* Error notifications appear as toasts */}
      <ErrorNotification />

      <div className="chat-header">
        <button
          className={`voice-mode-toggle ${voiceMode ? 'active' : ''}`}
          onClick={handleVoiceModeToggle}
          disabled={!conversationId}
          title={voiceMode ? 'Switch to text mode' : 'Switch to voice mode'}
        >
          {voiceMode ? '🎤 Voice Mode' : '💬 Text Mode'}
        </button>

        <VoiceSelector
          currentVoice={conversation?.preferences?.tts_voice}
          currentSpeed={conversation?.preferences?.tts_speed}
          onVoiceChange={handleVoiceChange}
          onSpeedChange={handleSpeedChange}
          disabled={!conversationId}
        />

        {voiceMode && (
          <div className="connection-status">
            {connectionState === 'connected' && '🟢 Connected'}
            {connectionState === 'connecting' && '🟡 Connecting...'}
            {connectionState === 'reconnecting' && '🟡 Reconnecting...'}
            {connectionState === 'disconnected' && '🔴 Disconnected'}
          </div>
        )}

        {/* Sync status indicator */}
        {conversationId && (
          <div
            className="sync-status"
            style={{
              fontSize: '12px',
              color: syncError ? '#d32f2f' : '#666',
              marginLeft: 'auto',
              padding: '4px 8px',
            }}
            title={
              syncError
                ? `Sync error: ${syncError}`
                : lastSyncTime
                ? `Last synced: ${lastSyncTime.toLocaleTimeString()}`
                : 'Not synced yet'
            }
          >
            {isSyncing && '🔄 Syncing...'}
            {!isSyncing && syncError && '⚠️ Sync error'}
            {!isSyncing && !syncError && lastSyncTime && (
              <>✓ Synced {formatRelativeTime(lastSyncTime)}</>
            )}
          </div>
        )}
      </div>

      {liveKitError && (
        <div className="livekit-error" style={{ color: 'red', padding: '8px', background: '#ffebee' }}>
          {liveKitError}
        </div>
      )}

      <MessageList messages={messages} loading={loading} />

      {/* Show protocol messages (errors, reasoning, tools, memories, commentary) */}
      <ProtocolDisplay />

      {/* Show streaming sentences in voice mode */}
      {streamingSentences.length > 0 && (
        <div className="streaming-response" style={{
          padding: '12px',
          background: '#f5f5f5',
          borderRadius: '8px',
          margin: '8px',
          fontStyle: 'italic',
        }}>
          <div style={{ fontSize: '12px', color: '#666', marginBottom: '4px' }}>
            Assistant (streaming):
          </div>
          {streamingSentences.join(' ')}
        </div>
      )}

      {/* Show transcription */}
      {currentTranscription && (
        <div className="transcription" style={{
          padding: '8px',
          background: '#e3f2fd',
          borderRadius: '4px',
          margin: '8px',
          fontSize: '14px',
        }}>
          <strong>You:</strong> {currentTranscription}
        </div>
      )}

      {/* Response controls (Stop/Regenerate) - only in voice mode */}
      {voiceMode && (
        <ResponseControls
          onStop={() => sendStop()}
          onRegenerate={(targetId) => sendRegenerate(targetId)}
          disabled={!connected}
        />
      )}

      {voiceMode && room && (
        <div className="voice-controls" style={{ padding: '8px', borderTop: '1px solid #ddd' }}>
          <AudioOutput room={room} />
          <AudioInput
            onTrackReady={publishAudioTrack}
            onTrackStop={unpublishAudioTrack}
            disabled={!connected}
          />
        </div>
      )}

      <InputBar
        onSend={handleSendMessage}
        disabled={voiceMode ? !connected : (!conversationId || sending)}
      />
    </div>
  );
}
