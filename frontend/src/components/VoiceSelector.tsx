import { useState, useEffect } from 'react';
import { useConfig } from '../contexts/ConfigContext';
import type { TTSRequest } from '../types/models';
import type { Voice } from '../services/api';

interface VoiceSelectorProps {
  currentVoice?: string;
  currentSpeed?: number;
  onVoiceChange: (voice: string) => void;
  onSpeedChange: (speed: number) => void;
  disabled?: boolean;
}

export function VoiceSelector({
  currentVoice,
  currentSpeed,
  onVoiceChange,
  onSpeedChange,
  disabled = false,
}: VoiceSelectorProps) {
  const { config, loading: configLoading } = useConfig();
  const [isExpanded, setIsExpanded] = useState(false);
  const [selectedVoice, setSelectedVoice] = useState(currentVoice || config?.tts?.default_voice || 'af_sarah');
  const [speed, setSpeed] = useState(currentSpeed || config?.tts?.default_speed || 1.0);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  // Sync with config defaults when config loads
  useEffect(() => {
    if (config?.tts) {
      if (!currentVoice) {
        setSelectedVoice(config.tts.default_voice);
      }
      if (!currentSpeed) {
        setSpeed(config.tts.default_speed);
      }
    }
  }, [config, currentVoice, currentSpeed]);

  useEffect(() => {
    if (currentVoice) {
      setSelectedVoice(currentVoice);
    }
  }, [currentVoice]);

  useEffect(() => {
    if (currentSpeed !== undefined) {
      setSpeed(currentSpeed);
    }
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
    if (!config?.tts) {
      setPreviewError('TTS is not configured');
      return;
    }

    setIsPreviewLoading(true);
    setPreviewError(null);

    try {
      const ttsRequest: TTSRequest = {
        model: config.tts.model,
        input: "Hello, I'm Alicia. This is a preview of my voice.",
        voice: selectedVoice,
        response_format: 'mp3',
        speed: speed,
      };

      const response = await fetch(config.tts.endpoint, {
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

  // Get voices from config, fallback to empty array
  const voices = config?.tts?.voices || [];

  const selectedVoiceObj = voices.find(v => v.id === selectedVoice);
  const voicesByCategory = voices.reduce((acc, voice) => {
    if (!acc[voice.category]) {
      acc[voice.category] = [];
    }
    acc[voice.category].push(voice);
    return acc;
  }, {} as Record<string, Voice[]>);

  // Show loading state while config is loading
  if (configLoading) {
    return (
      <div className="voice-selector">
        <button className="voice-selector-toggle" disabled title="Loading...">
          🎙️ Voice
        </button>
      </div>
    );
  }

  // Show disabled state if TTS is not enabled
  if (!config?.tts_enabled || !config?.tts) {
    return (
      <div className="voice-selector">
        <button className="voice-selector-toggle" disabled title="TTS not available">
          🎙️ Voice (Unavailable)
        </button>
      </div>
    );
  }

  const speedMin = config.tts.speed_min;
  const speedMax = config.tts.speed_max;
  const speedStep = config.tts.speed_step;

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
                min={speedMin}
                max={speedMax}
                step={speedStep}
                value={speed}
                onChange={(e) => handleSpeedChange(parseFloat(e.target.value))}
                disabled={disabled}
                className="speed-slider"
              />
              <div className="speed-markers">
                <span>{speedMin}x</span>
                <span>{((speedMin + speedMax) / 2).toFixed(1)}x</span>
                <span>{speedMax}x</span>
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
