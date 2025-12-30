import { useState, useEffect } from 'react';
import { api } from '../../../services/api';
import './OptimizationManager.css';

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

  useEffect(() => {
    loadRuns();
  }, [filter]);

  const loadRuns = async () => {
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
  };

  const getStatusBadge = (status: string) => {
    const statusClass = `status-badge status-${status.toLowerCase()}`;
    return <span className={statusClass}>{status}</span>;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  if (loading) {
    return <div className="optimization-loading">Loading optimization runs...</div>;
  }

  if (error) {
    return (
      <div className="optimization-error">
        <p>Error: {error}</p>
        <button onClick={loadRuns}>Retry</button>
      </div>
    );
  }

  return (
    <div className="optimization-run-list">
      <div className="optimization-header">
        <h2>Optimization Runs</h2>
        <div className="filter-controls">
          <label>Filter:</label>
          <select value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">All</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>
      </div>

      {runs.length === 0 ? (
        <div className="optimization-empty">
          <p>No optimization runs found.</p>
        </div>
      ) : (
        <div className="optimization-table-container">
          <table className="optimization-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Status</th>
                <th>Progress</th>
                <th>Best Score</th>
                <th>Started</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((run) => (
                <tr key={run.id} onClick={() => onSelectRun(run.id)} className="clickable-row">
                  <td>{run.name}</td>
                  <td>{run.prompt_type}</td>
                  <td>{getStatusBadge(run.status)}</td>
                  <td>
                    <div className="progress-info">
                      <span>{run.iterations} / {run.max_iterations}</span>
                      <div className="progress-bar">
                        <div
                          className="progress-fill"
                          style={{
                            width: `${(run.iterations / run.max_iterations) * 100}%`
                          }}
                        />
                      </div>
                    </div>
                  </td>
                  <td>{run.best_score.toFixed(4)}</td>
                  <td>{formatDate(run.started_at)}</td>
                  <td>
                    <button
                      className="btn-view"
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectRun(run.id);
                      }}
                    >
                      View
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
