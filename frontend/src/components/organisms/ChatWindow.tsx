import React, { useState } from 'react';
import MessageList from './MessageList';
import InputArea from './InputArea';
import ResponseControls from './ResponseControls';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { useLiveKit } from '../../hooks/useLiveKit';
import { cls } from '../../utils/cls';
import { useConfig } from '../../contexts/ConfigContext';

/**
 * ChatWindow organism component.
 *
 * Main chat container combining:
 * - MessageList for conversation history
 * - InputArea for user input with VAD support
 * - ResponseControls for stop/regenerate
 * - LiveKit integration for voice streaming
 * - Overall layout structure
 */

export interface ChatWindowProps {
  /** Callback when user sends a message */
  onSendMessage?: (message: string, isVoice: boolean) => void;
  /** Callback when user stops streaming */
  onStopStreaming?: () => void;
  /** Callback when user regenerates response */
  onRegenerateResponse?: () => void;
  /** Conversation ID for LiveKit connection (when using VAD) */
  conversationId?: string | null;
  /** Whether to use Silero VAD for voice input */
  useSileroVAD?: boolean;
  /** Whether to show response controls */
  showControls?: boolean;
  className?: string;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  onSendMessage,
  onStopStreaming,
  onRegenerateResponse,
  conversationId = null,
  useSileroVAD = false,
  showControls = true,
  className = '',
}) => {
  const [voiceModeActive, setVoiceModeActive] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const [voiceSelectorOpen, setVoiceSelectorOpen] = useState(false);
  const [selectedVoice, setSelectedVoice] = useState('af_sarah');
  const [speed, setSpeed] = useState(1.0);
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;
  const { config } = useConfig();

  // Initialize LiveKit when voice mode is active or using Silero VAD
  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
  } = useLiveKit((voiceModeActive || useSileroVAD) ? conversationId : null);

  const toggleVoiceMode = () => {
    setVoiceModeActive(!voiceModeActive);
    if (voiceModeActive) {
      setIsRecording(false);
    }
  };

  const toggleRecording = () => {
    setIsRecording(!isRecording);
  };

  const toggleVoiceSelector = () => {
    setVoiceSelectorOpen(!voiceSelectorOpen);
  };

  // Handle keyboard events for closing voice selector
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && voiceSelectorOpen) {
        setVoiceSelectorOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [voiceSelectorOpen]);

  const handleSendMessage = (message: string, isVoice: boolean) => {
    if (onSendMessage) {
      onSendMessage(message, isVoice);
    }
  };

  const handleStop = () => {
    if (onStopStreaming) {
      onStopStreaming();
    }
  };

  const handleRegenerate = () => {
    if (onRegenerateResponse) {
      onRegenerateResponse();
    }
  };

  // Determine voice connection status text
  const getVoiceConnectionText = () => {
    if (liveKitConnected) return 'Connected';
    if (liveKitError) return 'Error';
    if (voiceModeActive) return 'Connecting';
    return 'Disconnected';
  };

  return (
    <div className={cls('chat-window flex flex-col h-full bg-main-bg', className)}>
      {/* Connection status indicator - always visible when voice mode is active */}
      <div className={cls(
        'connection-status flex items-center justify-center gap-2 px-4 py-2 border-b',
        voiceModeActive
          ? (liveKitConnected ? 'bg-green-500/20 text-green-500 border-green-500/50' : 'bg-warning/20 text-warning border-warning/50')
          : (!isConnected ? 'bg-warning/20 text-warning border-warning/50' : 'hidden')
      )}>
        <svg className="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
        <span className="text-sm font-medium">
          {voiceModeActive ? getVoiceConnectionText() : (
            <>
              {connectionStatus === ConnectionStatus.Connecting && 'Connecting'}
              {connectionStatus === ConnectionStatus.Reconnecting && 'Reconnecting'}
              {connectionStatus === ConnectionStatus.Disconnected && 'Disconnected'}
              {connectionStatus === ConnectionStatus.Connected && 'Connected'}
              {connectionStatus === ConnectionStatus.Error && 'Connection error'}
            </>
          )}
        </span>
      </div>

      {/* LiveKit error indicator (when using VAD) */}
      {(useSileroVAD || voiceModeActive) && liveKitError && (
        <div className="flex items-center justify-center gap-2 px-4 py-2 bg-error/20 text-error border-b border-error/50">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <span className="text-sm font-medium">{liveKitError}</span>
        </div>
      )}

      {/* Message list */}
      <div className="flex-1 overflow-hidden">
        <MessageList />
      </div>

      {/* Response controls */}
      {showControls && (
        <ResponseControls
          onStop={handleStop}
          onRegenerate={handleRegenerate}
          show={isConnected}
        />
      )}

      {/* Voice controls - visible when voice mode is active */}
      {voiceModeActive && (
        <div className="voice-controls flex items-center justify-center gap-4 p-4 border-t border-primary-blue-glow">
          {/* Audio input indicator */}
          <div className="audio-input flex items-center gap-2 text-sm text-muted-text">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
            </svg>
            <span>Voice Input</span>
          </div>

          {/* Record button */}
          <button
            className={cls(
              'record-btn w-12 h-12 rounded-full flex items-center justify-center transition-all duration-200',
              isRecording
                ? 'recording bg-red-500 hover:bg-red-600 text-white animate-pulse'
                : 'bg-primary-blue hover:bg-primary-blue-hover text-white'
            )}
            onClick={toggleRecording}
            aria-label={isRecording ? 'Stop recording' : 'Start recording'}
          >
            {isRecording ? (
              <div className="w-4 h-4 bg-white rounded-sm" />
            ) : (
              <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
              </svg>
            )}
          </button>

          {/* Voice selector toggle */}
          <div className="relative">
            <button
              className="voice-selector-toggle flex items-center gap-2 px-3 py-2 text-sm text-muted-text hover:text-primary-text transition-colors"
              aria-label="Select voice"
              onClick={toggleVoiceSelector}
              aria-expanded={voiceSelectorOpen}
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
              <span>Voice</span>
            </button>

            {/* Voice selector panel */}
            {voiceSelectorOpen && (
              <div className="voice-selector-panel absolute bottom-full left-0 mb-2 w-64 bg-surface-800 border border-primary-blue-glow rounded-lg shadow-lg p-3">
                <div className="voice-selector-header flex items-center justify-between mb-3">
                  <h3 className="text-sm font-medium text-primary-text">Voice Settings</h3>
                  <button
                    className="voice-selector-close p-1 text-muted-text hover:text-primary-text rounded transition-colors"
                    onClick={() => setVoiceSelectorOpen(false)}
                    aria-label="Close voice selector"
                  >
                    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </button>
                </div>
                <div className="voice-selector-content space-y-3">
                  <div>
                    <label className="text-xs text-muted-text mb-1 block">Voice</label>
                    <select
                      className="voice-select w-full px-2 py-1.5 text-sm bg-surface-700 border border-primary-blue-glow rounded text-primary-text"
                      value={selectedVoice}
                      onChange={(e) => setSelectedVoice(e.target.value)}
                    >
                      {config?.tts?.voices ? (
                        config.tts.voices.map((voice) => (
                          <option key={voice.id} value={voice.id}>
                            {voice.name}
                          </option>
                        ))
                      ) : (
                        <>
                          <option value="af_sarah">Sarah</option>
                          <option value="am_adam">Adam</option>
                          <option value="af_nicole">Nicole</option>
                          <option value="am_michael">Michael</option>
                          <option value="bf_emma">Emma</option>
                          <option value="bm_george">George</option>
                        </>
                      )}
                    </select>
                  </div>
                  <div>
                    <label className="text-xs text-muted-text mb-1 block">Speed: {speed.toFixed(1)}x</label>
                    <input
                      type="range"
                      className="speed-slider w-full"
                      min={config?.tts?.speed_min ?? 0.5}
                      max={config?.tts?.speed_max ?? 2.0}
                      step={config?.tts?.speed_step ?? 0.1}
                      value={speed}
                      onChange={(e) => setSpeed(parseFloat(e.target.value))}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Voice mode toggle button */}
      <div className="flex items-center justify-between p-2 border-t border-primary-blue-glow">
        <button
          className={cls(
            'voice-mode-toggle flex items-center gap-2 px-3 py-2 rounded-lg transition-all duration-200',
            voiceModeActive
              ? 'active bg-primary-blue text-white'
              : 'bg-surface-800 text-muted-text hover:bg-surface-700'
          )}
          onClick={toggleVoiceMode}
          aria-label={voiceModeActive ? 'Disable voice mode' : 'Enable voice mode'}
          aria-pressed={voiceModeActive}
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M7 4a3 3 0 016 0v4a3 3 0 11-6 0V4zm4 10.93A7.001 7.001 0 0017 8a1 1 0 10-2 0A5 5 0 015 8a1 1 0 00-2 0 7.001 7.001 0 006 6.93V17H6a1 1 0 100 2h8a1 1 0 100-2h-3v-2.07z" clipRule="evenodd" />
          </svg>
          <span className="text-sm font-medium">Voice Mode</span>
        </button>
      </div>

      {/* Input area */}
      <InputArea
        onSend={handleSendMessage}
        onPublishAudioTrack={(useSileroVAD || voiceModeActive) ? publishAudioTrack : undefined}
        disabled={!isConnected || ((useSileroVAD || voiceModeActive) && !liveKitConnected)}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        useSileroVAD={useSileroVAD || voiceModeActive}
      />
    </div>
  );
};

export default ChatWindow;
