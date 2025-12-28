-- Alicia Database Schema
-- Migration 005: Rollback optimization tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_usage_patterns_updated_at ON tool_usage_patterns;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS tool_usage_patterns;
DROP TABLE IF EXISTS tool_result_formatters;
DROP TABLE IF EXISTS optimized_tools;
DROP TABLE IF EXISTS optimized_programs;
DROP TABLE IF EXISTS prompt_evaluations;
DROP TABLE IF EXISTS prompt_candidates;
DROP TABLE IF EXISTS prompt_optimization_runs;

-- Drop enum type
DROP TYPE IF EXISTS optimization_status;
