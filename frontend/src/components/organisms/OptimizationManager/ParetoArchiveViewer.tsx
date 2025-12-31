import { useState } from 'react';
import { cls } from '../../../utils/cls';

interface PromptCandidate {
  id: string;
  iteration: number;
  prompt_text: string;
  score: number;
  dimension_scores?: Record<string, number>;
  evaluation_count: number;
  success_count: number;
}

interface ParetoArchiveViewerProps {
  candidates: PromptCandidate[];
}

export function ParetoArchiveViewer({ candidates }: ParetoArchiveViewerProps) {
  const [selectedDimensions, setSelectedDimensions] = useState<[string, string]>(['successRate', 'quality']);

  const dimensions = candidates.length > 0 && candidates[0].dimension_scores
    ? Object.keys(candidates[0].dimension_scores)
    : ['successRate', 'quality', 'efficiency', 'robustness', 'generalization', 'diversity', 'innovation'];

  const toggleDimension = (index: 0 | 1, dimension: string) => {
    const newDimensions = [...selectedDimensions] as [string, string];
    newDimensions[index] = dimension;
    setSelectedDimensions(newDimensions);
  };

  const getParetoFrontier = () => {
    if (!candidates || candidates.length === 0) return [];

    const [dim1, dim2] = selectedDimensions;
    const frontier: PromptCandidate[] = [];

    const candidatesWithScores = candidates.filter(
      (c) => c.dimension_scores && c.dimension_scores[dim1] !== undefined && c.dimension_scores[dim2] !== undefined
    );

    for (const candidate of candidatesWithScores) {
      const isDominated = candidatesWithScores.some((other) => {
        if (other.id === candidate.id) return false;
        const otherDim1 = other.dimension_scores![dim1];
        const otherDim2 = other.dimension_scores![dim2];
        const candDim1 = candidate.dimension_scores![dim1];
        const candDim2 = candidate.dimension_scores![dim2];

        return otherDim1 >= candDim1 && otherDim2 >= candDim2 && (otherDim1 > candDim1 || otherDim2 > candDim2);
      });

      if (!isDominated) {
        frontier.push(candidate);
      }
    }

    return frontier.sort((a, b) => (a.dimension_scores![dim1] || 0) - (b.dimension_scores![dim1] || 0));
  };

  const paretoFrontier = getParetoFrontier();

  return (
    <div className="p-5">
      <h3>Pareto Archive Visualization</h3>

      <div className="flex gap-5 mb-5">
        <div className="flex gap-2.5 items-center">
          <label className="font-medium">X-Axis:</label>
          <select
            className="input"
            value={selectedDimensions[0]}
            onChange={(e) => toggleDimension(0, e.target.value)}
          >
            {dimensions.map((dim) => (
              <option key={dim} value={dim}>{dim}</option>
            ))}
          </select>
        </div>
        <div className="flex gap-2.5 items-center">
          <label className="font-medium">Y-Axis:</label>
          <select
            className="input"
            value={selectedDimensions[1]}
            onChange={(e) => toggleDimension(1, e.target.value)}
          >
            {dimensions.map((dim) => (
              <option key={dim} value={dim}>{dim}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="mb-5 p-2.5 bg-surface rounded">
        <p className="my-1.5">Total Candidates: {candidates.length}</p>
        <p className="my-1.5">Pareto Frontier Size: {paretoFrontier.length}</p>
      </div>

      <div className="mb-7 text-center">
        <svg width="600" height="400" className={cls('border bg-elevated')}>
          {/* Simple scatter plot - in production, use a proper charting library */}
          <g transform="translate(50, 350)">
            {/* Axes */}
            <line x1="0" y1="0" x2="500" y2="0" className="stroke-default" strokeWidth="2" />
            <line x1="0" y1="0" x2="0" y2="-300" className="stroke-default" strokeWidth="2" />

            {/* Axis labels */}
            <text x="250" y="30" textAnchor="middle" fontSize="12">
              {selectedDimensions[0]}
            </text>
            <text x="-150" y="-10" textAnchor="middle" fontSize="12" transform="rotate(-90, -10, -150)">
              {selectedDimensions[1]}
            </text>

            {/* All candidates */}
            {candidates
              .filter((c) => c.dimension_scores)
              .map((candidate) => {
                const x = (candidate.dimension_scores![selectedDimensions[0]] || 0) * 500;
                const y = -(candidate.dimension_scores![selectedDimensions[1]] || 0) * 300;
                const isPareto = paretoFrontier.some((p) => p.id === candidate.id);

                return (
                  <circle
                    key={candidate.id}
                    cx={x}
                    cy={y}
                    r={isPareto ? 6 : 4}
                    className={cls(isPareto ? 'fill-success stroke-success' : 'fill-muted stroke-none')}
                    opacity={isPareto ? 1 : 0.5}
                    strokeWidth={isPareto ? 2 : 0}
                  >
                    <title>
                      Iteration {candidate.iteration}
                      {'\n'}Score: {candidate.score.toFixed(4)}
                      {'\n'}{selectedDimensions[0]}: {(candidate.dimension_scores![selectedDimensions[0]] || 0).toFixed(3)}
                      {'\n'}{selectedDimensions[1]}: {(candidate.dimension_scores![selectedDimensions[1]] || 0).toFixed(3)}
                    </title>
                  </circle>
                );
              })}

            {/* Pareto frontier line */}
            {paretoFrontier.length > 1 && (
              <polyline
                points={paretoFrontier
                  .map((c) => {
                    const x = (c.dimension_scores![selectedDimensions[0]] || 0) * 500;
                    const y = -(c.dimension_scores![selectedDimensions[1]] || 0) * 300;
                    return `${x},${y}`;
                  })
                  .join(' ')}
                fill="none"
                className="stroke-success"
                strokeWidth="2"
                opacity="0.5"
              />
            )}
          </g>
        </svg>
      </div>

      <div>
        <h4 className="mb-4">Pareto Frontier Candidates</h4>
        {paretoFrontier.length === 0 ? (
          <p>No candidates on the Pareto frontier.</p>
        ) : (
          <ul className="list-none p-0">
            {paretoFrontier.map((candidate) => (
              <li key={candidate.id} className="p-2.5 bg-surface rounded mb-2">
                <strong>Iteration {candidate.iteration}</strong> - Score: {candidate.score.toFixed(4)}
                <br />
                {selectedDimensions[0]}: {(candidate.dimension_scores![selectedDimensions[0]] || 0).toFixed(3)}
                {' | '}
                {selectedDimensions[1]}: {(candidate.dimension_scores![selectedDimensions[1]] || 0).toFixed(3)}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
