-- Alicia Database Schema
-- Migration 005: Optimization tables for DSPy + GEPA

-- Create enum type for optimization run status
CREATE TYPE optimization_status AS ENUM ('pending', 'running', 'completed', 'failed');

-- ============================================================================
-- prompt_optimization_runs
-- ============================================================================
CREATE TABLE prompt_optimization_runs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('por'),
    signature_name VARCHAR(255) NOT NULL,
    status optimization_status NOT NULL DEFAULT 'pending',
    config JSONB DEFAULT '{}',
    best_score REAL,
    iterations INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_optimization_runs_status ON prompt_optimization_runs(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_signature ON prompt_optimization_runs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_created ON prompt_optimization_runs(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimization_runs_completed ON prompt_optimization_runs(completed_at DESC) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;

COMMENT ON TABLE prompt_optimization_runs IS 'GEPA optimization run records tracking prompt optimization sessions';
COMMENT ON COLUMN prompt_optimization_runs.config IS 'Optimization configuration including hyperparameters and constraints';

-- ============================================================================
-- prompt_candidates
-- ============================================================================
CREATE TABLE prompt_candidates (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('pc'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    demos JSONB,
    coverage INTEGER NOT NULL DEFAULT 0,
    avg_score REAL,
    generation INTEGER NOT NULL DEFAULT 0,
    parent_id TEXT REFERENCES prompt_candidates(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_candidates_run ON prompt_candidates(run_id, generation) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_score ON prompt_candidates(run_id, avg_score DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_candidates_parent ON prompt_candidates(parent_id) WHERE parent_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_candidates_coverage ON prompt_candidates(run_id, coverage DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE prompt_candidates IS 'Candidate prompts representing the Pareto frontier during optimization';
COMMENT ON COLUMN prompt_candidates.coverage IS 'Number of examples this candidate covers successfully';
COMMENT ON COLUMN prompt_candidates.generation IS 'Generation number in the evolutionary optimization process';

-- ============================================================================
-- prompt_evaluations
-- ============================================================================
CREATE TABLE prompt_evaluations (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('pe'),
    candidate_id TEXT NOT NULL REFERENCES prompt_candidates(id) ON DELETE CASCADE,
    example_id VARCHAR(255) NOT NULL,
    score REAL NOT NULL,
    feedback TEXT,
    trace JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_evaluations_candidate ON prompt_evaluations(candidate_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_example ON prompt_evaluations(example_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evaluations_score ON prompt_evaluations(candidate_id, score DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE prompt_evaluations IS 'Evaluation results for candidate prompts against test examples';
COMMENT ON COLUMN prompt_evaluations.trace IS 'Execution trace for debugging and analysis';

-- ============================================================================
-- optimized_programs
-- ============================================================================
CREATE TABLE optimized_programs (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('op'),
    run_id TEXT NOT NULL REFERENCES prompt_optimization_runs(id) ON DELETE CASCADE,
    signature_name VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL,
    demos JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_programs_run ON optimized_programs(run_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_signature ON optimized_programs(signature_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_programs_created ON optimized_programs(created_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE optimized_programs IS 'Final optimized prompt programs ready for deployment';

-- ============================================================================
-- optimized_tools
-- ============================================================================
CREATE TABLE optimized_tools (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('ot'),
    tool_id TEXT NOT NULL REFERENCES alicia_tools(id) ON DELETE CASCADE,
    optimized_description TEXT NOT NULL,
    optimized_schema JSONB,
    result_template TEXT,
    examples JSONB,
    version INTEGER NOT NULL DEFAULT 1,
    score REAL,
    optimized_at TIMESTAMP NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT false,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_optimized_tools_tool ON optimized_tools(tool_id, version DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_optimized_tools_active ON optimized_tools(tool_id, active) WHERE deleted_at IS NULL AND active = true;
CREATE INDEX idx_optimized_tools_score ON optimized_tools(score DESC NULLS LAST) WHERE deleted_at IS NULL;

COMMENT ON TABLE optimized_tools IS 'Optimized tool configurations with improved descriptions and schemas';
COMMENT ON COLUMN optimized_tools.active IS 'Whether this optimized version is currently in use';

-- ============================================================================
-- tool_result_formatters
-- ============================================================================
CREATE TABLE tool_result_formatters (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('trf'),
    tool_name VARCHAR(255) NOT NULL UNIQUE,
    template TEXT NOT NULL,
    max_length INTEGER NOT NULL DEFAULT 2000,
    summarize_at INTEGER NOT NULL DEFAULT 1000,
    summary_prompt TEXT,
    key_fields JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_formatters_tool_name ON tool_result_formatters(tool_name) WHERE deleted_at IS NULL;

COMMENT ON TABLE tool_result_formatters IS 'Learned result formatting rules for optimizing tool output presentation';
COMMENT ON COLUMN tool_result_formatters.summarize_at IS 'Character threshold at which to apply summarization';

-- ============================================================================
-- tool_usage_patterns
-- ============================================================================
CREATE TABLE tool_usage_patterns (
    id TEXT PRIMARY KEY DEFAULT generate_random_id('tup'),
    tool_name VARCHAR(255) NOT NULL,
    user_intent_pattern TEXT NOT NULL,
    success_rate REAL,
    avg_result_quality REAL,
    common_arguments JSONB,
    sample_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_usage_patterns_tool ON tool_usage_patterns(tool_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_success ON tool_usage_patterns(tool_name, success_rate DESC NULLS LAST) WHERE deleted_at IS NULL;
CREATE INDEX idx_usage_patterns_updated ON tool_usage_patterns(updated_at DESC) WHERE deleted_at IS NULL;

COMMENT ON TABLE tool_usage_patterns IS 'Tool usage analytics for identifying optimization opportunities';
COMMENT ON COLUMN tool_usage_patterns.user_intent_pattern IS 'Pattern describing common user intent for this tool usage';

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================
CREATE TRIGGER update_usage_patterns_updated_at
    BEFORE UPDATE ON tool_usage_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
