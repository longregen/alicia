import { useState, useEffect, useCallback, useMemo } from 'react';
import { api, OptimizationRun } from '../../../services/api';
import { useOptimizationProgressStore } from '../../../stores/optimizationProgressStore';
import { CreateOptimizationForm, CreateOptimizationRequest } from '../../molecules/CreateOptimizationForm/CreateOptimizationForm';
import { OptimizationProgressCard } from '../../molecules/OptimizationProgressCard/OptimizationProgressCard';

export interface GEPAControlsProps {
  conversationId?: string | null;
}

export function GEPAControls({ conversationId: _conversationId }: GEPAControlsProps) {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [recentRuns, setRecentRuns] = useState<OptimizationRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  // Subscribe to optimization progress store for active runs
  // Use progressByRunId directly and filter in useMemo to avoid infinite re-render loops
  const progressByRunId = useOptimizationProgressStore((state) => state.progressByRunId);
  const runningOptimizations = useMemo(
    () => Object.values(progressByRunId).filter((p) => p.status === 'running'),
    [progressByRunId]
  );

  // Auto-dismiss toast after 3 seconds
  useEffect(() => {
    if (toast) {
      const timer = setTimeout(() => setToast(null), 3000);
      return () => clearTimeout(timer);
    }
  }, [toast]);

  // Load recent runs on mount
  const loadRecentRuns = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.listOptimizationRuns();
      // Filter for completed/failed runs (not running)
      const completedRuns = data.filter(
        (run) => run.status.toLowerCase() !== 'running' && run.status.toLowerCase() !== 'pending'
      );
      // Sort by created_at descending and take recent ones
      const sortedRuns = completedRuns.sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );
      setRecentRuns(sortedRuns.slice(0, 10)); // Keep 10 most recent
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load optimization runs');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadRecentRuns();
  }, [loadRecentRuns]);

  // Handle creating a new optimization run
  const handleCreateOptimization = async (data: CreateOptimizationRequest) => {
    try {
      setSubmitting(true);
      await api.createOptimizationRun(data);
      setShowCreateForm(false);
      setToast({ message: 'Optimization run started successfully', type: 'success' });
      // Reload runs to get the new one
      await loadRecentRuns();
    } catch (err) {
      setToast({
        message: err instanceof Error ? err.message : 'Failed to create optimization run',
        type: 'error',
      });
      throw err; // Re-throw so form can handle it
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelCreate = () => {
    setShowCreateForm(false);
  };

  const getStatusBadge = (status: string) => {
    const statusLower = status.toLowerCase();
    let badgeClass = 'badge ';
    if (statusLower === 'completed') badgeClass += 'badge-success';
    else if (statusLower === 'failed') badgeClass += 'badge-destructive';
    else badgeClass += 'badge-secondary';
    return <span className={badgeClass}>{status}</span>;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  return (
    <div className="gepa-controls p-5 max-w-4xl mx-auto">
      {/* Toast notification */}
      {toast && (
        <div
          className={`fixed top-5 right-5 px-5 py-3 rounded-md text-sm font-medium shadow-lg z-[1000] animate-[slideIn_0.3s_ease] ${
            toast.type === 'success'
              ? 'bg-success text-success-foreground'
              : 'bg-destructive text-destructive-foreground'
          }`}
        >
          {toast.message}
        </div>
      )}

      {/* Header with action button */}
      <div className="layout-between mb-6">
        <h2 className="text-2xl font-semibold text-foreground m-0">GEPA Optimizer Controls</h2>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          {showCreateForm ? 'Cancel' : '+ Start New Optimization'}
        </button>
      </div>

      {/* Create optimization form */}
      {showCreateForm && (
        <div className="mb-6">
          <CreateOptimizationForm
            onSubmit={handleCreateOptimization}
            onCancel={handleCancelCreate}
            submitting={submitting}
          />
        </div>
      )}

      {/* Active runs section */}
      {runningOptimizations.length > 0 && (
        <div className="mb-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Active Runs</h3>
          <div className="grid gap-4 md:grid-cols-2">
            {runningOptimizations.map((progress) => (
              <OptimizationProgressCard
                key={progress.runId}
                runId={progress.runId}
                runName={progress.runName}
              />
            ))}
          </div>
        </div>
      )}

      {/* Recent runs section */}
      <div>
        <h3 className="text-lg font-semibold text-foreground mb-4">Recent Runs</h3>

        {loading && (
          <div className="text-center text-muted py-10 px-5 text-sm">
            Loading optimization runs...
          </div>
        )}

        {error && (
          <div className="bg-destructive/10 text-destructive px-4 py-3 rounded-md mb-4 text-sm">
            {error}
            <button className="ml-4 underline" onClick={loadRecentRuns}>
              Retry
            </button>
          </div>
        )}

        {!loading && !error && recentRuns.length === 0 && (
          <div className="text-center text-muted py-10 px-5 text-sm">
            No recent optimization runs. Click "Start New Optimization" to begin.
          </div>
        )}

        {!loading && !error && recentRuns.length > 0 && (
          <div className="layout-stack-gap-4">
            {recentRuns.map((run) => (
              <div
                key={run.id}
                className="card card-hover p-4 flex justify-between items-center"
              >
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    <span className="font-semibold text-foreground">{run.name}</span>
                    {getStatusBadge(run.status)}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    <span className="mr-4">Type: {run.prompt_type}</span>
                    <span className="mr-4">
                      Progress: {run.iterations}/{run.max_iterations}
                    </span>
                    <span className="mr-4">Best Score: {run.best_score.toFixed(4)}</span>
                    <span>Started: {formatDate(run.created_at)}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default GEPAControls;
