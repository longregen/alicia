import React, { useState } from 'react';
import MessageList from './MessageList';
import InputArea from './InputArea';
import ResponseControls from './ResponseControls';
import VoiceVisualizer, { type VoiceState } from '../atoms/VoiceVisualizer';
import { useConnectionStore, ConnectionStatus } from '../../stores/connectionStore';
import { useLiveKit } from '../../hooks/useLiveKit';
import { useConfig } from '../../contexts/ConfigContext';
import { cls } from '../../utils/cls';

/**
 * ChatWindow organism component.
 *
 * Main chat container combining:
 * - Chat header with title, conversation ID, and audio controls
 * - MessageList for conversation history
 * - InputArea for user input with VAD support
 * - ResponseControls for stop/regenerate
 * - VoiceVisualizer for voice state feedback
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
  /** Conversation title for display */
  conversationTitle?: string;
  /** Whether to use Silero VAD for voice input */
  useSileroVAD?: boolean;
  /** Whether to show response controls */
  showControls?: boolean;
  /** Whether audio output is enabled */
  audioOutputEnabled?: boolean;
  /** Callback when audio output toggle is clicked */
  onAudioOutputToggle?: () => void;
  className?: string;
}

const ChatWindow: React.FC<ChatWindowProps> = ({
  onSendMessage,
  onStopStreaming,
  onRegenerateResponse,
  conversationId = null,
  conversationTitle = 'Conversation',
  useSileroVAD = false,
  showControls = true,
  audioOutputEnabled = true,
  onAudioOutputToggle,
  className = '',
}) => {
  const { config } = useConfig();
  const [voiceModeActive, setVoiceModeActive] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const [voiceSelectorOpen, setVoiceSelectorOpen] = useState(false);
  const [voiceState, setVoiceState] = useState<VoiceState>('idle');
  const [selectedVoice, setSelectedVoice] = useState(config?.tts?.default_voice || 'af_sarah');
  const [speed, setSpeed] = useState(config?.tts?.default_speed || 1.0);
  const [isPreviewPlaying, setIsPreviewPlaying] = useState(false);
  const connectionStatus = useConnectionStore((state) => state.status);
  const isConnected = connectionStatus === ConnectionStatus.Connected;

  // Initialize LiveKit when voice mode is active or using Silero VAD
  const {
    connected: liveKitConnected,
    error: liveKitError,
    publishAudioTrack,
    sendStop,
  } = useLiveKit((voiceModeActive || useSileroVAD) ? conversationId : null);

  const toggleVoiceMode = () => {
    setVoiceModeActive(!voiceModeActive);
    if (voiceModeActive) {
      setIsRecording(false);
      setVoiceState('idle');
    }
  };

  const toggleRecording = () => {
    if (isRecording) {
      setIsRecording(false);
      setVoiceState('processing');
      setTimeout(() => setVoiceState('idle'), 2000);
    } else {
      setIsRecording(true);
      setVoiceState('listening');
    }
  };

  const handleAudioToggle = () => {
    if (onAudioOutputToggle) {
      onAudioOutputToggle();
    }
  };

  const toggleVoiceSelector = () => {
    setVoiceSelectorOpen(!voiceSelectorOpen);
  };

  const handlePreviewVoice = async () => {
    if (isPreviewPlaying) return;
    setIsPreviewPlaying(true);
    try {
      // Create audio element to play a sample text with the selected voice
      const previewText = "Hello, this is a preview of the selected voice.";
      // For now, log - actual TTS implementation would go here
      console.log(`Preview voice: ${selectedVoice} at ${speed}x speed with text: "${previewText}"`);
      // Simulate preview duration
      await new Promise(resolve => setTimeout(resolve, 2000));
    } finally {
      setIsPreviewPlaying(false);
    }
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

  const handleStop = async () => {
    if (onStopStreaming) {
      onStopStreaming();
    }
    // Also send stop via LiveKit if connected
    if (sendStop && conversationId) {
      try {
        await sendStop();
      } catch (error) {
        console.error('Failed to send stop command:', error);
      }
    }
  };

  const handleRegenerate = async () => {
    if (onRegenerateResponse) {
      onRegenerateResponse();
    }
    // Regeneration is handled by parent component callback
  };

  // Determine voice connection status text
  const getVoiceConnectionText = () => {
    if (liveKitConnected) return 'Connected';
    if (liveKitError) return 'Error';
    if (voiceModeActive) return 'Connecting';
    return 'Disconnected';
  };

  return (
    <div className={cls('layout-stack h-full bg-background', className)}>
      {/* Chat Header */}
      <header className="h-14 border-b border-border layout-between px-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="font-medium text-foreground">{conversationTitle}</h2>
          <span className="text-xs text-muted-foreground px-2 py-1 bg-muted rounded">
            {conversationId || 'No ID'}
          </span>
        </div>
        <div className="layout-center-gap">
          <button
            onClick={handleAudioToggle}
            className={cls(
              'p-2 rounded-lg transition-colors',
              audioOutputEnabled
                ? 'text-accent bg-accent/10 hover:bg-accent hover:text-accent-foreground'
                : 'text-muted-foreground bg-muted hover:bg-muted/80'
            )}
            aria-label={audioOutputEnabled ? 'Disable audio output' : 'Enable audio output'}
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              {audioOutputEnabled ? (
                <path
                  fillRule="evenodd"
                  d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414zm-2.829 2.828a1 1 0 011.415 0A5.983 5.983 0 0115 10a5.984 5.984 0 01-1.757 4.243 1 1 0 01-1.415-1.415A3.984 3.984 0 0013 10a3.983 3.983 0 00-1.172-2.828 1 1 0 010-1.415z"
                  clipRule="evenodd"
                />
              ) : (
                <path
                  fillRule="evenodd"
                  d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM12.293 7.293a1 1 0 011.414 0L15 8.586l1.293-1.293a1 1 0 111.414 1.414L16.414 10l1.293 1.293a1 1 0 01-1.414 1.414L15 11.414l-1.293 1.293a1 1 0 01-1.414-1.414L13.586 10l-1.293-1.293a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              )}
            </svg>
          </button>
        </div>
      </header>

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
      {/* TODO: Voice mode UI is placeholder - actual recording uses Silero VAD in InputArea */}
      {voiceModeActive && (
        <div className="voice-controls layout-vcenter-gap p-4 border-t border-primary-blue-glow">
          {/* VoiceVisualizer */}
          <VoiceVisualizer state={voiceState} />

          {/* Control buttons row */}
          <div className="flex items-center justify-center gap-4">
            {/* Audio input indicator */}
            <div className="audio-input layout-center-gap text-sm text-muted-text">
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

            {/* Audio output indicator */}
            <div className="audio-output layout-center-gap text-sm text-muted-text">
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414zm-2.829 2.828a1 1 0 011.415 0A5.983 5.983 0 0115 10a5.984 5.984 0 01-1.757 4.243 1 1 0 01-1.415-1.415A3.984 3.984 0 0013 10a3.983 3.983 0 00-1.172-2.828 1 1 0 010-1.415z" clipRule="evenodd" />
              </svg>
              <span>Voice Output</span>
            </div>
          </div>
        </div>
      )}

      {/* Voice mode toggle button and voice selector */}
      <div className="layout-between p-2 border-t border-primary-blue-glow">
        <button
          className={cls(
            'voice-mode-toggle layout-center-gap px-3 py-2 rounded-lg transition-all duration-200',
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

        {/* Voice selector toggle - always accessible */}
        <div className="relative">
          <button
            className="voice-selector-toggle layout-center-gap px-3 py-2 text-sm text-muted-text hover:text-primary-text transition-colors rounded-lg hover:bg-surface-700"
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
            <div className="voice-selector-panel absolute bottom-full right-0 mb-2 w-80 bg-surface-800 border border-primary-blue-glow rounded-lg shadow-lg">
              {/* Header with close button */}
              <div className="voice-selector-header layout-between p-3 border-b border-primary-blue-glow">
                <h3 className="text-sm font-semibold text-primary-text">Voice Settings</h3>
                <button
                  className="voice-selector-close w-6 h-6 flex items-center justify-center rounded hover:bg-surface-700 text-muted-text hover:text-primary-text transition-colors"
                  onClick={toggleVoiceSelector}
                  aria-label="Close voice selector"
                >
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                  </svg>
                </button>
              </div>

              {/* Content */}
              <div className="voice-selector-content p-4 space-y-4">
                {/* Voice selection */}
                <div className="voice-option-group">
                  <label className="voice-label text-xs text-muted-text mb-2 block">Voice</label>
                  <div className="layout-center-gap">
                    <select
                      className="voice-select flex-1 px-3 py-2 bg-surface-700 border border-primary-blue-glow rounded text-sm text-primary-text focus:outline-none focus:border-primary-blue"
                      value={selectedVoice}
                      onChange={(e) => setSelectedVoice(e.target.value)}
                    >
                      {config?.tts?.voices?.map((voice) => (
                        <option key={voice.id} value={voice.id}>
                          {voice.name} ({voice.category})
                        </option>
                      )) || (
                        <option value="af_sarah">Sarah (American Female)</option>
                      )}
                    </select>
                    <button
                      onClick={handlePreviewVoice}
                      disabled={isPreviewPlaying}
                      className="ml-2 p-2 rounded-lg bg-accent/10 hover:bg-accent/20 text-accent disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                      aria-label="Preview voice"
                    >
                      {isPreviewPlaying ? (
                        <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
                        </svg>
                      )}
                    </button>
                  </div>
                </div>

                {/* Speed control */}
                <div className="voice-option-group">
                  <label className="voice-label text-xs text-muted-text mb-2 block">
                    Speed: {speed.toFixed(1)}x
                  </label>
                  <input
                    type="range"
                    className="speed-slider w-full"
                    min={config?.tts?.speed_min || 0.5}
                    max={config?.tts?.speed_max || 2.0}
                    step={config?.tts?.speed_step || 0.1}
                    value={speed}
                    onChange={(e) => setSpeed(parseFloat(e.target.value))}
                  />
                  <div className="speed-markers flex justify-between text-xs text-muted-text mt-1">
                    <span>{config?.tts?.speed_min || 0.5}x</span>
                    <span>{config?.tts?.speed_max || 2.0}x</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Input area */}
      <InputArea
        onSend={handleSendMessage}
        onPublishAudioTrack={(useSileroVAD || voiceModeActive) ? publishAudioTrack : undefined}
        disabled={!isConnected}
        placeholder={isConnected ? 'Type a message...' : 'Connecting...'}
        useSileroVAD={useSileroVAD || voiceModeActive}
      />
    </div>
  );
};

export default ChatWindow;
