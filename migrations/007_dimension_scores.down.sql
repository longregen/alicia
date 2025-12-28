-- Rollback dimension scores migration

-- Drop pareto_archive table
DROP TABLE IF EXISTS pareto_archive;

-- Remove dimension_weights from prompt_optimization_runs
ALTER TABLE prompt_optimization_runs
DROP COLUMN IF EXISTS dimension_weights;

-- Remove dimension_scores from prompt_evaluations
ALTER TABLE prompt_evaluations
DROP COLUMN IF EXISTS dimension_scores;

-- Remove dimension_scores from prompt_candidates
ALTER TABLE prompt_candidates
DROP COLUMN IF EXISTS dimension_scores;
