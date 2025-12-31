import React, { useCallback } from 'react';
import {
  useDimensionStore,
  calculateWeightedScore,
} from '../../../stores/dimensionStore';
import type { DimensionScores, EliteSummary } from '../../../types/protocol';
import { cls } from '../../../utils/cls';

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

  // Get color class based on score
  const getBarColorClass = (pct: number): string => {
    if (pct >= 80) return 'bg-success';
    if (pct >= 60) return 'bg-warning';
    return 'bg-muted';
  };

  if (compact) {
    return (
      <span
        className="inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-sunken rounded text-xs"
        title={`${dimension}: ${percentage}%`}
      >
        <span className="text-xs">{icon}</span>
        <span className="text-muted font-medium">{percentage}%</span>
      </span>
    );
  }

  return (
    <div className="layout-center-gap w-full">
      <span className="text-sm min-w-[18px]">{icon}</span>
      <div className="progress-track h-1.5 flex-1">
        <div
          className={cls('progress-fill h-full transition-all duration-300', getBarColorClass(percentage))}
          style={{ width: `${percentage}%` }}
        />
      </div>
      <span className="min-w-[36px] text-right text-xs font-medium text-muted">{percentage}%</span>
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
    <div className={`card p-3 transition-colors ${isActive ? 'border-accent bg-accent-subtle' : 'hover:border-accent'}`}>
      <div className="layout-between mb-2">
        <span className="font-semibold text-[13px] text-default flex items-center gap-1">
          {isActive && <span className="text-sm">⭐</span>}
          {elite.label}
        </span>
        {!isActive && (
          <button
            className="btn btn-secondary text-xs px-2.5 py-1 hover:btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            onClick={() => onSelect(elite.id)}
            disabled={disabled}
          >
            Use
          </button>
        )}
        {isActive && <span className="badge bg-accent text-white text-[10px]">Current</span>}
      </div>

      <div className="flex flex-wrap gap-2 mb-2">
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

      <div className="text-xs text-muted leading-snug mb-1.5">{elite.description}</div>

      {elite.bestFor && (
        <div className="text-xs text-muted mb-1.5">
          <span className="font-medium">Best for:</span> {elite.bestFor}
        </div>
      )}

      <div className="text-[10px] text-muted text-right italic">
        Weighted score: {Math.round(weightedScore * 100)}%
      </div>
    </div>
  );
};

export interface EliteSolutionSelectorProps {
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
      <div className="bg-surface border border-default rounded-lg p-4">
        <div className="layout-center-gap mb-3">
          <span className="text-lg">🏆</span>
          <span className="font-semibold text-sm text-foreground">Elite Solutions</span>
        </div>
        <div className="py-6 text-center text-muted-foreground text-[13px]">Loading elite solutions...</div>
      </div>
    );
  }

  if (elites.length === 0) {
    return (
      <div className="bg-surface border border-default rounded-lg p-4">
        <div className="layout-center-gap mb-3">
          <span className="text-lg">🏆</span>
          <span className="font-semibold text-sm text-foreground">Elite Solutions</span>
        </div>
        <div className="py-6 text-center text-muted-foreground text-[13px]">
          <p>No elite solutions available yet.</p>
          <p className="text-xs text-muted-foreground mt-2">
            Elite solutions are generated through prompt optimization.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-surface border border-default rounded-lg p-4 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
      <div className="layout-center-gap mb-3">
        <span className="text-lg">🏆</span>
        <span className="font-semibold text-sm text-foreground">Elite Solutions</span>
      </div>

      <p className="text-xs text-muted mb-3">
        Available optimized configurations:
      </p>

      <div className="flex flex-col gap-3">
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
