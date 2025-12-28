-- Add dimension scores to prompt candidates and evaluations
-- This enables multi-objective GEPA optimization across 7 dimensions

-- Add dimension_scores JSONB column to prompt_candidates
ALTER TABLE prompt_candidates
ADD COLUMN IF NOT EXISTS dimension_scores JSONB DEFAULT '{}';

-- Add dimension_scores JSONB column to prompt_evaluations
ALTER TABLE prompt_evaluations
ADD COLUMN IF NOT EXISTS dimension_scores JSONB DEFAULT '{}';

-- Add dimension_weights to prompt_optimization_runs for tracking optimization preferences
ALTER TABLE prompt_optimization_runs
ADD COLUMN IF NOT EXISTS dimension_weights JSONB DEFAULT '{"successRate": 0.25, "quality": 0.20, "efficiency": 0.15, "robustness": 0.15, "generalization": 0.10, "diversity": 0.10, "innovation": 0.05}';

-- Add pareto_archive for storing elite solutions
CREATE TABLE IF NOT EXISTS pareto_archive (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('pa'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB DEFAULT '[]',
    dimension_scores JSONB NOT NULL DEFAULT '{}',
    generation INT NOT NULL DEFAULT 0,
    coverage INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pareto_archive_run ON pareto_archive(run_id);
CREATE INDEX IF NOT EXISTS idx_pareto_archive_scores ON pareto_archive USING GIN (dimension_scores);

-- Add comment for documentation
COMMENT ON TABLE pareto_archive IS 'Stores elite solutions from GEPA Pareto optimization, representing non-dominated trade-offs across 7 dimensions';
COMMENT ON COLUMN pareto_archive.dimension_scores IS 'Per-dimension performance metrics: successRate, quality, efficiency, robustness, generalization, diversity, innovation';
COMMENT ON COLUMN pareto_archive.coverage IS 'Number of examples this solution solves best, used for coverage-based selection';
