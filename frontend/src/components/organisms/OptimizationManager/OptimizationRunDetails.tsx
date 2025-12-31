import { useState, useEffect, useCallback } from 'react';
import { api } from '../../../services/api';
import { cls } from '../../../utils/cls';
import { DimensionScoresChart } from './DimensionScoresChart';
import { ParetoArchiveViewer } from './ParetoArchiveViewer';

interface OptimizationRun {
  id: string;
  name: string;
  description?: string;
  status: string;
  prompt_type: string;
  baseline_score?: number;
  best_score: number;
  iterations: number;
  max_iterations: number;
  dimension_weights?: Record<string, number>;
  best_dim_scores?: Record<string, number>;
  config?: Record<string, unknown>;
  started_at: string;
  completed_at?: string;
}

interface PromptCandidate {
  id: string;
  iteration: number;
  prompt_text: string;
  score: number;
  dimension_scores?: Record<string, number>;
  evaluation_count: number;
  success_count: number;
  created_at: string;
}

interface OptimizationRunDetailsProps {
  runId: string;
  onBack: () => void;
}

export function OptimizationRunDetails({ runId, onBack }: OptimizationRunDetailsProps) {
  const [run, setRun] = useState<OptimizationRun | null>(null);
  const [candidates, setCandidates] = useState<PromptCandidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTab, setSelectedTab] = useState<'overview' | 'candidates' | 'pareto'>('overview');

  const loadRunDetails = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api.getOptimizationRun(runId);
      setRun(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load run details');
    } finally {
      setLoading(false);
    }
  }, [runId]);

  const loadCandidates = useCallback(async () => {
    try {
      const data = await api.getOptimizationCandidates(runId);
      setCandidates(data);
    } catch (err) {
      console.error('Failed to load candidates:', err);
    }
  }, [runId]);

  useEffect(() => {
    loadRunDetails();
    loadCandidates();
  }, [loadRunDetails, loadCandidates]);

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  if (loading) {
    return <div className="text-center py-10 text-muted">Loading run details...</div>;
  }

  if (error || !run) {
    return (
      <div className="text-center py-10 text-muted">
        <p>Error: {error || 'Run not found'}</p>
        <button className="btn btn-secondary" onClick={onBack}>Back to List</button>
      </div>
    );
  }

  const statusLower = run.status.toLowerCase();
  let badgeClass = 'badge ';
  if (statusLower === 'completed') badgeClass += 'badge-success';
  else if (statusLower === 'running') badgeClass += 'badge-warning';
  else if (statusLower === 'failed') badgeClass += 'badge-error';

  return (
    <div className="p-5">
      <div className="flex gap-5 items-center mb-5">
        <button className="btn btn-secondary" onClick={onBack}>← Back</button>
        <h2 className="flex-1 m-0">{run.name}</h2>
        <span className={badgeClass}>
          {run.status}
        </span>
      </div>

      <div className="tabs">
        <button
          className={selectedTab === 'overview' ? 'tab active' : 'tab'}
          onClick={() => setSelectedTab('overview')}
        >
          Overview
        </button>
        <button
          className={selectedTab === 'candidates' ? 'tab active' : 'tab'}
          onClick={() => setSelectedTab('candidates')}
        >
          Candidates ({candidates.length})
        </button>
        <button
          className={selectedTab === 'pareto' ? 'tab active' : 'tab'}
          onClick={() => setSelectedTab('pareto')}
        >
          Pareto Archive
        </button>
      </div>

      <div className="p-5 card">
        {selectedTab === 'overview' && (
          <div>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-5 mb-7">
              <div className="bg-surface p-5 rounded-md">
                <h3 className="mt-0 mb-4 text-base">Run Information</h3>
                <dl className="grid grid-cols-[auto_1fr] gap-2 m-0">
                  <dt className="font-semibold text-muted">ID:</dt>
                  <dd className="m-0">{run.id}</dd>
                  <dt className="font-semibold text-muted">Type:</dt>
                  <dd className="m-0">{run.prompt_type}</dd>
                  <dt className="font-semibold text-muted">Iterations:</dt>
                  <dd className="m-0">{run.iterations} / {run.max_iterations}</dd>
                  <dt className="font-semibold text-muted">Started:</dt>
                  <dd className="m-0">{formatDate(run.started_at)}</dd>
                  {run.completed_at && (
                    <>
                      <dt className="font-semibold text-muted">Completed:</dt>
                      <dd className="m-0">{formatDate(run.completed_at)}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="bg-surface p-5 rounded-md">
                <h3 className="mt-0 mb-4 text-base">Performance</h3>
                <dl className="grid grid-cols-[auto_1fr] gap-2 m-0">
                  <dt className="font-semibold text-muted">Best Score:</dt>
                  <dd className="m-0 text-2xl font-bold text-success">{run.best_score.toFixed(4)}</dd>
                  {run.baseline_score !== undefined && (
                    <>
                      <dt className="font-semibold text-muted">Baseline Score:</dt>
                      <dd className="m-0">{run.baseline_score.toFixed(4)}</dd>
                      <dt className="font-semibold text-muted">Improvement:</dt>
                      <dd className={cls('m-0 font-semibold', run.best_score > run.baseline_score ? 'text-success' : 'text-error')}>
                        {((run.best_score - run.baseline_score) / run.baseline_score * 100).toFixed(1)}%
                      </dd>
                    </>
                  )}
                </dl>
              </div>
            </div>

            {run.best_dim_scores && (
              <div className="mt-7">
                <h3 className="mb-4">Best Dimension Scores</h3>
                <DimensionScoresChart scores={run.best_dim_scores} weights={run.dimension_weights} />
              </div>
            )}

            {run.dimension_weights && (
              <div className="mt-7">
                <h3 className="mb-4">Dimension Weights</h3>
                <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-4">
                  {Object.entries(run.dimension_weights).map(([dim, weight]) => {
                    const widthPercent = weight * 100;
                    return (
                      <div key={dim} className="flex flex-col gap-1">
                        <span className="text-sm font-medium">{dim}:</span>
                        <span className="text-lg font-bold text-accent">{widthPercent.toFixed(0)}%</span>
                        <div className="h-2 bg-sunken rounded overflow-hidden">
                          <div className="h-full bg-accent transition-all" style={{ width: `${widthPercent}%` }} />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        {selectedTab === 'candidates' && (
          <div>
            <h3>Prompt Candidates</h3>
            {candidates.length === 0 ? (
              <p>No candidates found.</p>
            ) : (
              <div className="flex flex-col gap-4">
                {candidates.map((candidate) => (
                  <div key={candidate.id} className="bg-surface p-4 rounded-md border-l-4 border-accent">
                    <div className="flex justify-between mb-2">
                      <span className="font-semibold">Iteration {candidate.iteration}</span>
                      <span className="font-semibold text-success">Score: {candidate.score.toFixed(4)}</span>
                    </div>
                    <div className="flex gap-4 text-sm text-muted mb-2.5">
                      <span>Evaluations: {candidate.evaluation_count}</span>
                      <span>
                        Success Rate: {((candidate.success_count / candidate.evaluation_count) * 100).toFixed(1)}%
                      </span>
                    </div>
                    {candidate.dimension_scores && (
                      <div className="my-2.5">
                        <DimensionScoresChart scores={candidate.dimension_scores} compact />
                      </div>
                    )}
                    <details className="mt-2.5">
                      <summary className="cursor-pointer font-medium text-accent">View Prompt</summary>
                      <pre className="mt-2.5 p-2.5 bg-elevated rounded overflow-x-auto text-xs leading-relaxed">{candidate.prompt_text}</pre>
                    </details>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {selectedTab === 'pareto' && (
          <div>
            <ParetoArchiveViewer candidates={candidates} />
          </div>
        )}
      </div>
    </div>
  );
}
