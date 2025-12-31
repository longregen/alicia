import React, { useState, useCallback } from 'react';
import {
  useDimensionStore,
  PIVOT_PRESETS,
  type PresetId,
} from '../../../stores/dimensionStore';
import type { DimensionWeights } from '../../../types/protocol';
import { cls } from '../../../utils/cls';

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

export interface PivotModeSelectorProps {
  showAdvanced?: boolean;
  onPresetChange?: (presetId: PresetId) => void;
  onWeightsChange?: (weights: DimensionWeights) => void;
  disabled?: boolean;
}

export const PivotModeSelector: React.FC<PivotModeSelectorProps> = ({
  showAdvanced: initialShowAdvanced = false, // Renamed from showAdvanced for clarity
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
    <div className={cls('bg-surface border border-default rounded-lg p-4', disabled ? 'opacity-60 pointer-events-none' : '')}>
      <div className="layout-center-gap mb-3">
        <span className="text-base">⚙️</span>
        <span className="font-semibold text-sm text-foreground">Response Style</span>
      </div>

      <div className="flex flex-wrap gap-2 mb-3">
        {PIVOT_PRESETS.map((preset) => (
          <button
            key={preset.id}
            className={cls(
              'flex items-center gap-1 px-3 py-2 rounded-md cursor-pointer text-sm transition-all',
              presetId === preset.id
                ? 'bg-accent/20 text-accent font-medium'
                : 'bg-secondary text-muted-foreground hover:bg-accent/10 hover:text-accent disabled:opacity-50 disabled:cursor-not-allowed'
            )}
            onClick={() => handlePresetClick(preset.id)}
            disabled={disabled}
            title={preset.description}
            aria-pressed={presetId === preset.id}
          >
            <span className="text-sm">{preset.icon}</span>
            <span className="text-xs">{preset.label}</span>
          </button>
        ))}
      </div>

      <button
        className="btn-ghost w-full text-xs text-center"
        onClick={() => setShowAdvanced(!showAdvanced)}
        aria-expanded={showAdvanced}
      >
        Custom weights {showAdvanced ? '▲' : '▼'}
      </button>

      {showAdvanced && (
        <div className="mt-3 pt-3 border-t border-border/50">
          <div className="flex flex-col gap-3">
            {DIMENSION_CONFIG.map(({ key, label, icon }) => (
              <div key={key} className="flex items-center gap-3">
                <label className="flex items-center gap-1 min-w-[110px] text-xs text-muted">
                  <span className="text-sm">{icon}</span>
                  <span className="flex-1">{label}</span>
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
                  className="flex-1 h-1 rounded bg-sunken appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3.5 [&::-webkit-slider-thumb]:h-3.5 [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-accent [&::-webkit-slider-thumb]:cursor-pointer [&::-webkit-slider-thumb]:transition-transform [&::-webkit-slider-thumb]:hover:scale-110 [&::-moz-range-thumb]:w-3.5 [&::-moz-range-thumb]:h-3.5 [&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-accent [&::-moz-range-thumb]:cursor-pointer"
                  aria-label={`${label} weight`}
                />
                <span className="min-w-[40px] text-right text-xs font-medium text-default">
                  {Math.round(weights[key] * 100)}%
                </span>
              </div>
            ))}
          </div>

          <div className="flex justify-end gap-2 mt-4">
            <button
              className="btn btn-secondary text-xs"
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
