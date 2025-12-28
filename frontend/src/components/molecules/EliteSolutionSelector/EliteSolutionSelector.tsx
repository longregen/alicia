import React, { useCallback } from 'react';
import {
  useDimensionStore,
  calculateWeightedScore,
} from '../../../stores/dimensionStore';
import type { DimensionScores, EliteSummary } from '../../../types/protocol';
import './EliteSolutionSelector.css';

// Dimension display configuration matching PivotModeSelector
const DIMENSION_DISPLAY: Array<{
  key: keyof DimensionScores;
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

interface DimensionScoreBarProps {
  dimension: keyof DimensionScores;
  score: number;
  icon: string;
  compact?: boolean;
}

const DimensionScoreBar: React.FC<DimensionScoreBarProps> = ({
  dimension,
  score,
  icon,
  compact = false,
}) => {
  // Convert 0-1 score to percentage
  const percentage = Math.round(score * 100);

  // Color coding based on score
  const getBarColor = (pct: number): string => {
    if (pct >= 80) return 'var(--color-success, #22c55e)';
    if (pct >= 60) return 'var(--color-warning, #eab308)';
    return 'var(--color-muted, #6b7280)';
  };

  if (compact) {
    return (
      <span
        className="dimension-score-compact"
        title={`${dimension}: ${percentage}%`}
      >
        <span className="compact-icon">{icon}</span>
        <span className="compact-value">{percentage}%</span>
      </span>
    );
  }

  return (
    <div className="dimension-score-bar">
      <span className="score-icon">{icon}</span>
      <div className="score-track">
        <div
          className="score-fill"
          style={{
            width: `${percentage}%`,
            backgroundColor: getBarColor(percentage),
          }}
        />
      </div>
      <span className="score-value">{percentage}%</span>
    </div>
  );
};

interface EliteCardProps {
  elite: EliteSummary;
  isActive: boolean;
  weightedScore: number;
  onSelect: (id: string) => void;
  disabled?: boolean;
}

const EliteCard: React.FC<EliteCardProps> = ({
  elite,
  isActive,
  weightedScore,
  onSelect,
  disabled = false,
}) => {
  return (
    <div className={`elite-card ${isActive ? 'active' : ''}`}>
      <div className="elite-header">
        <span className="elite-label">
          {isActive && <span className="active-indicator">⭐</span>}
          {elite.label}
        </span>
        {!isActive && (
          <button
            className="elite-use-button"
            onClick={() => onSelect(elite.id)}
            disabled={disabled}
          >
            Use
          </button>
        )}
        {isActive && <span className="current-badge">Current</span>}
      </div>

      <div className="elite-scores">
        {DIMENSION_DISPLAY.slice(0, 4).map(({ key, icon }) => (
          <DimensionScoreBar
            key={key}
            dimension={key}
            score={elite.scores[key]}
            icon={icon}
            compact
          />
        ))}
        {elite.scores.diversity > 0.5 && (
          <DimensionScoreBar
            dimension="diversity"
            score={elite.scores.diversity}
            icon="🎨"
            compact
          />
        )}
        {elite.scores.innovation > 0.5 && (
          <DimensionScoreBar
            dimension="innovation"
            score={elite.scores.innovation}
            icon="💡"
            compact
          />
        )}
      </div>

      <div className="elite-description">{elite.description}</div>

      {elite.bestFor && (
        <div className="elite-best-for">
          <span className="best-for-label">Best for:</span> {elite.bestFor}
        </div>
      )}

      <div className="elite-weighted-score">
        Weighted score: {Math.round(weightedScore * 100)}%
      </div>
    </div>
  );
};

interface EliteSolutionSelectorProps {
  onSelectElite?: (eliteId: string) => void;
  disabled?: boolean;
}

export const EliteSolutionSelector: React.FC<EliteSolutionSelectorProps> = ({
  onSelectElite,
  disabled = false,
}) => {
  const elites = useDimensionStore((s) => s.elites);
  const currentEliteId = useDimensionStore((s) => s.currentEliteId);
  const weights = useDimensionStore((s) => s.weights);
  const selectElite = useDimensionStore((s) => s.selectElite);
  const isLoading = useDimensionStore((s) => s.isLoading);

  const handleSelectElite = useCallback(
    (eliteId: string) => {
      if (disabled) return;
      selectElite(eliteId);
      onSelectElite?.(eliteId);
    },
    [disabled, selectElite, onSelectElite]
  );

  // Sort elites by weighted score (descending)
  const sortedElites = [...elites].sort((a, b) => {
    const scoreA = calculateWeightedScore(a.scores, weights);
    const scoreB = calculateWeightedScore(b.scores, weights);
    return scoreB - scoreA;
  });

  if (isLoading) {
    return (
      <div className="elite-selector loading">
        <div className="elite-header-section">
          <span className="elite-icon">🏆</span>
          <span className="elite-title">Elite Solutions</span>
        </div>
        <div className="elite-loading">Loading elite solutions...</div>
      </div>
    );
  }

  if (elites.length === 0) {
    return (
      <div className="elite-selector empty">
        <div className="elite-header-section">
          <span className="elite-icon">🏆</span>
          <span className="elite-title">Elite Solutions</span>
        </div>
        <div className="elite-empty">
          <p>No elite solutions available yet.</p>
          <p className="elite-empty-hint">
            Elite solutions are generated through prompt optimization.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`elite-selector ${disabled ? 'disabled' : ''}`}>
      <div className="elite-header-section">
        <span className="elite-icon">🏆</span>
        <span className="elite-title">Elite Solutions</span>
      </div>

      <p className="elite-intro">
        Available optimized configurations:
      </p>

      <div className="elite-list">
        {sortedElites.map((elite) => (
          <EliteCard
            key={elite.id}
            elite={elite}
            isActive={elite.id === currentEliteId}
            weightedScore={calculateWeightedScore(elite.scores, weights)}
            onSelect={handleSelectElite}
            disabled={disabled}
          />
        ))}
      </div>
    </div>
  );
};

export default EliteSolutionSelector;
