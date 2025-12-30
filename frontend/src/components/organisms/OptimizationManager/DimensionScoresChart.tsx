import './OptimizationManager.css';

interface DimensionScoresChartProps {
  scores: Record<string, number>;
  weights?: Record<string, number>;
  compact?: boolean;
}

const DIMENSION_LABELS: Record<string, string> = {
  successRate: 'Success Rate',
  quality: 'Quality',
  efficiency: 'Efficiency',
  robustness: 'Robustness',
  generalization: 'Generalization',
  diversity: 'Diversity',
  innovation: 'Innovation',
};

const DIMENSION_COLORS: Record<string, string> = {
  successRate: '#4CAF50',
  quality: '#2196F3',
  efficiency: '#FFC107',
  robustness: '#9C27B0',
  generalization: '#FF5722',
  diversity: '#00BCD4',
  innovation: '#E91E63',
};

export function DimensionScoresChart({ scores, weights, compact }: DimensionScoresChartProps) {
  const dimensions = Object.keys(scores).sort();

  if (compact) {
    return (
      <div className="dimension-chart-compact">
        {dimensions.map((dim) => (
          <div key={dim} className="dimension-bar-compact">
            <div
              className="dimension-fill-compact"
              style={{
                width: `${scores[dim] * 100}%`,
                backgroundColor: DIMENSION_COLORS[dim] || '#999',
              }}
            />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="dimension-scores-chart">
      <div className="dimension-bars">
        {dimensions.map((dim) => {
          const score = scores[dim];
          const weight = weights?.[dim];
          const label = DIMENSION_LABELS[dim] || dim;
          const color = DIMENSION_COLORS[dim] || '#999';

          return (
            <div key={dim} className="dimension-row">
              <div className="dimension-label">
                <span>{label}</span>
                {weight !== undefined && (
                  <span className="dimension-weight">(weight: {(weight * 100).toFixed(0)}%)</span>
                )}
              </div>
              <div className="dimension-bar-container">
                <div
                  className="dimension-bar-fill"
                  style={{
                    width: `${score * 100}%`,
                    backgroundColor: color,
                  }}
                />
                <span className="dimension-value">{score.toFixed(3)}</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Radar chart visualization would go here in a production implementation */}
      <div className="radar-chart-placeholder">
        <p>Radar chart visualization (requires chart library like recharts or d3)</p>
      </div>
    </div>
  );
}
