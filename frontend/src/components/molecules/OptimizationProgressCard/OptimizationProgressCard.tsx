import React from 'react';
import { useOptimizationProgressStore, type OptimizationStatus } from '../../../stores/optimizationProgressStore';
import { DimensionScoresChart } from '../../organisms/OptimizationManager/DimensionScoresChart';
import Badge from '../../atoms/Badge';
import { Progress } from '../../atoms/Progress';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardFooter,
} from '../../atoms/Card';
import { cls } from '../../../utils/cls';

export interface OptimizationProgressCardProps {
  runId: string;
  runName: string;
  onViewDetails?: () => void;
}

const getStatusBadgeVariant = (status: OptimizationStatus): 'warning' | 'success' | 'destructive' | 'secondary' => {
  switch (status) {
    case 'running':
      return 'warning';
    case 'completed':
      return 'success';
    case 'failed':
      return 'destructive';
    case 'pending':
    default:
      return 'secondary';
  }
};

const getStatusLabel = (status: OptimizationStatus): string => {
  switch (status) {
    case 'running':
      return 'Running';
    case 'completed':
      return 'Completed';
    case 'failed':
      return 'Failed';
    case 'pending':
      return 'Pending';
    default:
      return status;
  }
};

export const OptimizationProgressCard: React.FC<OptimizationProgressCardProps> = ({
  runId,
  runName,
  onViewDetails,
}) => {
  // Subscribe to progress store for this specific runId
  const progress = useOptimizationProgressStore((state) => state.progressByRunId[runId]);

  // If no progress data, show loading state
  if (!progress) {
    return (
      <Card className="w-full">
        <CardHeader>
          <CardTitle className="text-sm">{runName}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="py-4 text-center text-muted-foreground text-sm">
            Loading optimization progress...
          </div>
        </CardContent>
      </Card>
    );
  }

  const {
    status,
    iteration,
    maxIterations,
    currentScore,
    bestScore,
    dimensionScores,
    dimensionWeights,
    errorMessage,
  } = progress;

  // Calculate progress percentage
  const progressPercent = maxIterations > 0 ? (iteration / maxIterations) * 100 : 0;

  return (
    <Card className="w-full">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-sm font-semibold">{runName}</CardTitle>
          <Badge
            variant={getStatusBadgeVariant(status)}
            showDot
            dotColor={cls(
              status === 'running' && 'bg-yellow-500',
              status === 'completed' && 'bg-green-500',
              status === 'failed' && 'bg-red-500',
              status === 'pending' && 'bg-gray-400'
            )}
          >
            {getStatusLabel(status)}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Progress bar with iteration info */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>Progress</span>
            <span>
              {iteration} / {maxIterations} ({progressPercent.toFixed(0)}%)
            </span>
          </div>
          <Progress value={progressPercent} className="h-2" />
        </div>

        {/* Score display */}
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-sunken rounded-md p-3">
            <div className="text-xs text-muted-foreground mb-1">Current Score</div>
            <div className="text-lg font-semibold text-foreground">
              {currentScore.toFixed(4)}
            </div>
          </div>
          <div className="bg-sunken rounded-md p-3">
            <div className="text-xs text-muted-foreground mb-1">Best Score</div>
            <div className="text-lg font-semibold text-success">
              {bestScore.toFixed(4)}
            </div>
          </div>
        </div>

        {/* Error message if failed */}
        {status === 'failed' && errorMessage && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-md p-3">
            <div className="text-xs text-destructive font-medium mb-1">Error</div>
            <div className="text-sm text-destructive">{errorMessage}</div>
          </div>
        )}

        {/* Dimension scores chart */}
        {dimensionScores && Object.keys(dimensionScores).length > 0 && (
          <div className="space-y-2">
            <div className="text-xs text-muted-foreground font-medium">Dimension Scores</div>
            <DimensionScoresChart
              scores={dimensionScores as unknown as Record<string, number>}
              weights={dimensionWeights}
              compact
            />
          </div>
        )}
      </CardContent>

      {/* View Details button */}
      {onViewDetails && (
        <CardFooter className="pt-0">
          <button
            className="btn btn-secondary w-full text-sm"
            onClick={onViewDetails}
          >
            View Details
          </button>
        </CardFooter>
      )}
    </Card>
  );
};

export default OptimizationProgressCard;
