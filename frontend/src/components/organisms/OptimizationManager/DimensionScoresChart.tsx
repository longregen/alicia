import { cls } from '../../../utils/cls';

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
  successRate: 'var(--chart-success)',
  quality: 'var(--chart-primary)',
  efficiency: 'var(--chart-warning)',
  robustness: 'var(--chart-purple)',
  generalization: 'var(--chart-orange)',
  diversity: 'var(--chart-cyan)',
  innovation: 'var(--chart-pink)',
};

export function DimensionScoresChart({ scores, weights, compact }: DimensionScoresChartProps) {
  const dimensions = Object.keys(scores).sort();

  if (compact) {
    return (
      <div className="flex gap-0.5 h-5">
        {dimensions.map((dim) => {
          const widthPercent = scores[dim] * 100;
          const bgColor = DIMENSION_COLORS[dim] || 'var(--chart-primary)';
          return (
            <div key={dim} className="flex-1 bg-sunken rounded-sm overflow-hidden">
              <div
                className="h-full transition-all"
                style={{
                  width: `${widthPercent}%`,
                  backgroundColor: bgColor,
                }}
              />
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3">
        {dimensions.map((dim) => {
          const score = scores[dim];
          const weight = weights?.[dim];
          const label = DIMENSION_LABELS[dim] || dim;
          const color = DIMENSION_COLORS[dim] || 'var(--chart-primary)';
          const widthPercent = score * 100;

          return (
            <div key={dim} className="flex gap-4 items-center">
              <div className="min-w-[150px] text-sm">
                <span>{label}</span>
                {weight !== undefined && (
                  <span className="text-xs text-muted ml-2">(weight: {(weight * 100).toFixed(0)}%)</span>
                )}
              </div>
              <div className="flex-1 relative h-6 bg-sunken rounded overflow-hidden">
                <div
                  className="h-full transition-all"
                  style={{
                    width: `${widthPercent}%`,
                    backgroundColor: color,
                  }}
                />
                <span className={cls('absolute right-2 top-1/2 -translate-y-1/2 text-xs font-semibold text-default')}>{score.toFixed(3)}</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Radar chart visualization would go here in a production implementation */}
      <div className="text-center py-10 bg-surface rounded text-muted">
        <p>Radar chart visualization (requires chart library like recharts or d3)</p>
      </div>
    </div>
  );
}
