import { useState, useEffect, useCallback } from 'react';
import { api } from '../../../services/api';
import { DimensionScoresChart } from './DimensionScoresChart';
import { ParetoArchiveViewer } from './ParetoArchiveViewer';
import './OptimizationManager.css';

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
    return <div className="optimization-loading">Loading run details...</div>;
  }

  if (error || !run) {
    return (
      <div className="optimization-error">
        <p>Error: {error || 'Run not found'}</p>
        <button onClick={onBack}>Back to List</button>
      </div>
    );
  }

  return (
    <div className="optimization-run-details">
      <div className="details-header">
        <button className="btn-back" onClick={onBack}>← Back</button>
        <h2>{run.name}</h2>
        <span className={`status-badge status-${run.status.toLowerCase()}`}>
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

      <div className="tab-content">
        {selectedTab === 'overview' && (
          <div className="overview-tab">
            <div className="info-grid">
              <div className="info-card">
                <h3>Run Information</h3>
                <dl>
                  <dt>ID:</dt>
                  <dd>{run.id}</dd>
                  <dt>Type:</dt>
                  <dd>{run.prompt_type}</dd>
                  <dt>Iterations:</dt>
                  <dd>{run.iterations} / {run.max_iterations}</dd>
                  <dt>Started:</dt>
                  <dd>{formatDate(run.started_at)}</dd>
                  {run.completed_at && (
                    <>
                      <dt>Completed:</dt>
                      <dd>{formatDate(run.completed_at)}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="info-card">
                <h3>Performance</h3>
                <dl>
                  <dt>Best Score:</dt>
                  <dd className="score-value">{run.best_score.toFixed(4)}</dd>
                  {run.baseline_score !== undefined && (
                    <>
                      <dt>Baseline Score:</dt>
                      <dd>{run.baseline_score.toFixed(4)}</dd>
                      <dt>Improvement:</dt>
                      <dd className={run.best_score > run.baseline_score ? 'positive' : 'negative'}>
                        {((run.best_score - run.baseline_score) / run.baseline_score * 100).toFixed(1)}%
                      </dd>
                    </>
                  )}
                </dl>
              </div>
            </div>

            {run.best_dim_scores && (
              <div className="dimension-scores-section">
                <h3>Best Dimension Scores</h3>
                <DimensionScoresChart scores={run.best_dim_scores} weights={run.dimension_weights} />
              </div>
            )}

            {run.dimension_weights && (
              <div className="dimension-weights-section">
                <h3>Dimension Weights</h3>
                <div className="weights-grid">
                  {Object.entries(run.dimension_weights).map(([dim, weight]) => (
                    <div key={dim} className="weight-item">
                      <span className="weight-label">{dim}:</span>
                      <span className="weight-value">{(weight * 100).toFixed(0)}%</span>
                      <div className="weight-bar">
                        <div className="weight-fill" style={{ width: `${weight * 100}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {selectedTab === 'candidates' && (
          <div className="candidates-tab">
            <h3>Prompt Candidates</h3>
            {candidates.length === 0 ? (
              <p>No candidates found.</p>
            ) : (
              <div className="candidates-list">
                {candidates.map((candidate) => (
                  <div key={candidate.id} className="candidate-card">
                    <div className="candidate-header">
                      <span className="candidate-iteration">Iteration {candidate.iteration}</span>
                      <span className="candidate-score">Score: {candidate.score.toFixed(4)}</span>
                    </div>
                    <div className="candidate-stats">
                      <span>Evaluations: {candidate.evaluation_count}</span>
                      <span>
                        Success Rate: {((candidate.success_count / candidate.evaluation_count) * 100).toFixed(1)}%
                      </span>
                    </div>
                    {candidate.dimension_scores && (
                      <div className="candidate-dimensions">
                        <DimensionScoresChart scores={candidate.dimension_scores} compact />
                      </div>
                    )}
                    <details className="candidate-prompt">
                      <summary>View Prompt</summary>
                      <pre>{candidate.prompt_text}</pre>
                    </details>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {selectedTab === 'pareto' && (
          <div className="pareto-tab">
            <ParetoArchiveViewer candidates={candidates} />
          </div>
        )}
      </div>
    </div>
  );
}
