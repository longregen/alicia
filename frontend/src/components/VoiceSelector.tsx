import { useState, useEffect } from 'react';
import type { TTSRequest } from '../types/models';

interface VoiceSelectorProps {
  currentVoice?: string;
  currentSpeed?: number;
  onVoiceChange: (voice: string) => void;
  onSpeedChange: (speed: number) => void;
  disabled?: boolean;
}

interface VoiceOption {
  id: string;
  name: string;
  category: string;
}

// Kokoro TTS voice list organized by category
const VOICES: VoiceOption[] = [
  // American Female
  { id: 'af_sarah', name: 'Sarah', category: 'American Female' },
  { id: 'af_nicole', name: 'Nicole', category: 'American Female' },
  { id: 'af_sky', name: 'Sky', category: 'American Female' },
  { id: 'af_bella', name: 'Bella', category: 'American Female' },

  // American Male
  { id: 'am_adam', name: 'Adam', category: 'American Male' },
  { id: 'am_michael', name: 'Michael', category: 'American Male' },
  { id: 'am_jack', name: 'Jack', category: 'American Male' },
  { id: 'am_ryan', name: 'Ryan', category: 'American Male' },

  // British Female
  { id: 'bf_emma', name: 'Emma', category: 'British Female' },
  { id: 'bf_isabella', name: 'Isabella', category: 'British Female' },
  { id: 'bf_lily', name: 'Lily', category: 'British Female' },
  { id: 'bf_alice', name: 'Alice', category: 'British Female' },

  // British Male
  { id: 'bm_george', name: 'George', category: 'British Male' },
  { id: 'bm_lewis', name: 'Lewis', category: 'British Male' },
  { id: 'bm_charlie', name: 'Charlie', category: 'British Male' },
  { id: 'bm_oliver', name: 'Oliver', category: 'British Male' },
];

export function VoiceSelector({
  currentVoice = 'af_sarah',
  currentSpeed = 1.0,
  onVoiceChange,
  onSpeedChange,
  disabled = false,
}: VoiceSelectorProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [selectedVoice, setSelectedVoice] = useState(currentVoice);
  const [speed, setSpeed] = useState(currentSpeed);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  useEffect(() => {
    setSelectedVoice(currentVoice);
  }, [currentVoice]);

  useEffect(() => {
    setSpeed(currentSpeed);
  }, [currentSpeed]);

  const handleVoiceChange = (voiceId: string) => {
    setSelectedVoice(voiceId);
    onVoiceChange(voiceId);
  };

  const handleSpeedChange = (newSpeed: number) => {
    setSpeed(newSpeed);
    onSpeedChange(newSpeed);
  };

  const handlePreview = async () => {
    setIsPreviewLoading(true);
    setPreviewError(null);

    try {
      const ttsRequest: TTSRequest = {
        model: 'kokoro',
        input: "Hello, I'm Alicia. This is a preview of my voice.",
        voice: selectedVoice,
        response_format: 'mp3',
        speed: speed,
      };

      const response = await fetch('/v1/audio/speech', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(ttsRequest),
      });

      if (!response.ok) {
        throw new Error(`TTS request failed: ${response.status} ${response.statusText}`);
      }

      const audioBlob = await response.blob();
      const audioUrl = URL.createObjectURL(audioBlob);
      const audio = new Audio(audioUrl);

      audio.onended = () => {
        URL.revokeObjectURL(audioUrl);
      };

      audio.onerror = () => {
        URL.revokeObjectURL(audioUrl);
        setPreviewError('Failed to play audio');
        setIsPreviewLoading(false);
      };

      await audio.play();
      setIsPreviewLoading(false);
    } catch (error) {
      console.error('Voice preview error:', error);
      setPreviewError(error instanceof Error ? error.message : 'Failed to preview voice');
      setIsPreviewLoading(false);
    }
  };

  const selectedVoiceObj = VOICES.find(v => v.id === selectedVoice);
  const voicesByCategory = VOICES.reduce((acc, voice) => {
    if (!acc[voice.category]) {
      acc[voice.category] = [];
    }
    acc[voice.category].push(voice);
    return acc;
  }, {} as Record<string, VoiceOption[]>);

  return (
    <div className="voice-selector">
      <button
        className="voice-selector-toggle"
        onClick={() => setIsExpanded(!isExpanded)}
        disabled={disabled}
        title="Voice settings"
      >
        🎙️ Voice
      </button>

      {isExpanded && (
        <div className="voice-selector-panel">
          <div className="voice-selector-header">
            <h3>Voice Settings</h3>
            <button
              className="voice-selector-close"
              onClick={() => setIsExpanded(false)}
            >
              ✕
            </button>
          </div>

          <div className="voice-selector-content">
            {/* Voice Selection */}
            <div className="voice-option-group">
              <label className="voice-label">Voice:</label>
              <select
                value={selectedVoice}
                onChange={(e) => handleVoiceChange(e.target.value)}
                disabled={disabled}
                className="voice-select"
              >
                {Object.entries(voicesByCategory).map(([category, voices]) => (
                  <optgroup key={category} label={category}>
                    {voices.map((voice) => (
                      <option key={voice.id} value={voice.id}>
                        {voice.name}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </select>
            </div>

            {/* Speed Adjustment */}
            <div className="voice-option-group">
              <label className="voice-label">
                Speed: {speed.toFixed(2)}x
              </label>
              <input
                type="range"
                min="0.5"
                max="2.0"
                step="0.1"
                value={speed}
                onChange={(e) => handleSpeedChange(parseFloat(e.target.value))}
                disabled={disabled}
                className="speed-slider"
              />
              <div className="speed-markers">
                <span>0.5x</span>
                <span>1.0x</span>
                <span>2.0x</span>
              </div>
            </div>

            {/* Preview Button */}
            <div className="voice-option-group">
              <button
                onClick={handlePreview}
                disabled={disabled || isPreviewLoading}
                className="preview-btn"
              >
                {isPreviewLoading ? '⏳ Loading...' : '🔊 Preview Voice'}
              </button>
              {previewError && (
                <div className="preview-error" style={{ color: 'red', fontSize: '0.875rem', marginTop: '0.5rem' }}>
                  {previewError}
                </div>
              )}
            </div>

            {/* Current Selection Display */}
            <div className="current-selection">
              <small>
                Current: {selectedVoiceObj?.name || selectedVoice} ({selectedVoiceObj?.category}) at {speed}x speed
              </small>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
