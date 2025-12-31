import { useState, useEffect, useCallback } from 'react';
import { api } from '../../../services/api';
import { cls } from '../../../utils/cls';

interface OptimizationRun {
  id: string;
  name: string;
  status: string;
  prompt_type: string;
  best_score: number;
  iterations: number;
  max_iterations: number;
  started_at: string;
  completed_at?: string;
}

interface OptimizationRunListProps {
  onSelectRun: (runId: string) => void;
}

export function OptimizationRunList({ onSelectRun }: OptimizationRunListProps) {
  const [runs, setRuns] = useState<OptimizationRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<string>('all');

  const loadRuns = useCallback(async () => {
    try {
      setLoading(true);
      const params: Record<string, string> = {};
      if (filter !== 'all') {
        params.status = filter;
      }
      const data = await api.listOptimizationRuns(params);
      setRuns(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load optimization runs');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    loadRuns();
  }, [loadRuns]);

  const getStatusBadge = (status: string) => {
    const statusLower = status.toLowerCase();
    let badgeClass = 'badge ';
    if (statusLower === 'completed') badgeClass += 'badge-success';
    else if (statusLower === 'running') badgeClass += 'badge-warning';
    else if (statusLower === 'failed') badgeClass += 'badge-error';
    return <span className={badgeClass}>{status}</span>;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  if (loading) {
    return <div className="text-center py-10 text-muted">Loading optimization runs...</div>;
  }

  if (error) {
    return (
      <div className="text-center py-10 text-muted">
        <p>Error: {error}</p>
        <button className="mt-2.5 btn btn-primary" onClick={loadRuns}>Retry</button>
      </div>
    );
  }

  return (
    <div className="p-5">
      <div className="layout-between mb-5">
        <h2 className="m-0 text-2xl">Optimization Runs</h2>
        <div className="flex gap-2.5 items-center">
          <label className="font-medium">Filter:</label>
          <select className="input" value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">All</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>
      </div>

      {runs.length === 0 ? (
        <div className="text-center py-10 text-muted">
          <p>No optimization runs found.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse card overflow-hidden">
            <thead className="bg-surface">
              <tr>
                <th className="p-3 text-left border-b border font-semibold text-default">Name</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Type</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Status</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Progress</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Best Score</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Started</th>
                <th className="p-3 text-left border-b border font-semibold text-default">Actions</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => {
                const progressPercent = (run.iterations / run.max_iterations) * 100;
                return (
                  <tr key={run.id} onClick={() => onSelectRun(run.id)} className="cursor-pointer transition-colors hover:bg-surface">
                    <td className="p-3 text-left border-b border">{run.name}</td>
                    <td className="p-3 text-left border-b border">{run.prompt_type}</td>
                    <td className="p-3 text-left border-b border">{getStatusBadge(run.status)}</td>
                    <td className="p-3 text-left border-b border">
                      <div className="flex flex-col gap-1">
                        <span>{run.iterations} / {run.max_iterations}</span>
                        <div className="progress-track w-24">
                          <div
                            className={cls('progress-fill transition-all')}
                            style={{ width: `${progressPercent}%` }}
                          />
                        </div>
                      </div>
                    </td>
                    <td className="p-3 text-left border-b border">{run.best_score.toFixed(4)}</td>
                    <td className="p-3 text-left border-b border">{formatDate(run.started_at)}</td>
                    <td className="p-3 text-left border-b border">
                      <button
                        className="btn btn-primary"
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectRun(run.id);
                        }}
                      >
                        View
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
