import React, { useState, useCallback } from 'react';
import {
  useDimensionStore,
  PIVOT_PRESETS,
  type PresetId,
} from '../../../stores/dimensionStore';
import type { DimensionWeights } from '../../../types/protocol';
import './PivotModeSelector.css';

// Dimension display configuration
const DIMENSION_CONFIG: Array<{
  key: keyof DimensionWeights;
  label: string;
  icon: string;
}> = [
  { key: 'successRate', label: 'Accuracy', icon: '✓' },
  { key: 'quality', label: 'Quality', icon: '★' },
  { key: 'efficiency', label: 'Speed', icon: '⚡' },
  { key: 'robustness', label: 'Reliability', icon: '🛡️' },
  { key: 'generalization', label: 'Adaptability', icon: '🔄' },
  { key: 'diversity', label: 'Creativity', icon: '🎨' },
  { key: 'innovation', label: 'Novelty', icon: '💡' },
];

interface PivotModeSelectorProps {
  showAdvanced?: boolean;
  onPresetChange?: (presetId: PresetId) => void;
  onWeightsChange?: (weights: DimensionWeights) => void;
  disabled?: boolean;
}

export const PivotModeSelector: React.FC<PivotModeSelectorProps> = ({
  showAdvanced: initialShowAdvanced = false,
  onPresetChange,
  onWeightsChange,
  disabled = false,
}) => {
  const [showAdvanced, setShowAdvanced] = useState(initialShowAdvanced);

  const weights = useDimensionStore((s) => s.weights);
  const presetId = useDimensionStore((s) => s.presetId);
  const setPreset = useDimensionStore((s) => s.setPreset);
  const setDimensionWeight = useDimensionStore((s) => s.setDimensionWeight);
  const resetToBalanced = useDimensionStore((s) => s.resetToBalanced);

  const handlePresetClick = useCallback(
    (id: PresetId) => {
      if (disabled) return;
      setPreset(id);
      onPresetChange?.(id);
    },
    [disabled, setPreset, onPresetChange]
  );

  const handleSliderChange = useCallback(
    (dimension: keyof DimensionWeights, value: number) => {
      if (disabled) return;
      setDimensionWeight(dimension, value);
      onWeightsChange?.(useDimensionStore.getState().weights);
    },
    [disabled, setDimensionWeight, onWeightsChange]
  );

  const handleReset = useCallback(() => {
    if (disabled) return;
    resetToBalanced();
    onPresetChange?.('balanced');
  }, [disabled, resetToBalanced, onPresetChange]);

  return (
    <div className={`pivot-mode-selector ${disabled ? 'disabled' : ''}`}>
      <div className="pivot-header">
        <span className="pivot-icon">⚙️</span>
        <span className="pivot-title">Response Style</span>
      </div>

      <div className="pivot-presets">
        {PIVOT_PRESETS.map((preset) => (
          <button
            key={preset.id}
            className={`pivot-preset-button ${
              presetId === preset.id ? 'active' : ''
            }`}
            onClick={() => handlePresetClick(preset.id)}
            disabled={disabled}
            title={preset.description}
            aria-pressed={presetId === preset.id}
          >
            <span className="preset-icon">{preset.icon}</span>
            <span className="preset-label">{preset.label}</span>
          </button>
        ))}
      </div>

      <button
        className="pivot-advanced-toggle"
        onClick={() => setShowAdvanced(!showAdvanced)}
        aria-expanded={showAdvanced}
      >
        Custom weights {showAdvanced ? '▲' : '▼'}
      </button>

      {showAdvanced && (
        <div className="pivot-advanced">
          <div className="dimension-sliders">
            {DIMENSION_CONFIG.map(({ key, label, icon }) => (
              <div key={key} className="dimension-slider">
                <label className="dimension-label">
                  <span className="dimension-icon">{icon}</span>
                  <span className="dimension-name">{label}</span>
                </label>
                <input
                  type="range"
                  min="0"
                  max="100"
                  value={Math.round(weights[key] * 100)}
                  onChange={(e) =>
                    handleSliderChange(key, parseInt(e.target.value, 10) / 100)
                  }
                  disabled={disabled}
                  className="dimension-range"
                  aria-label={`${label} weight`}
                />
                <span className="dimension-value">
                  {Math.round(weights[key] * 100)}%
                </span>
              </div>
            ))}
          </div>

          <div className="pivot-actions">
            <button
              className="pivot-reset-button"
              onClick={handleReset}
              disabled={disabled || presetId === 'balanced'}
            >
              Reset to Balanced
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default PivotModeSelector;
