-- Alicia Database Schema
-- Migration 001: Rollback consolidated schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_usage_patterns_updated_at ON tool_usage_patterns;
DROP TRIGGER IF EXISTS update_session_stats_updated_at ON alicia_session_stats;
DROP TRIGGER IF EXISTS update_notes_updated_at ON alicia_notes;
DROP TRIGGER IF EXISTS update_votes_updated_at ON alicia_votes;
DROP TRIGGER IF EXISTS update_mcp_servers_updated_at ON alicia_mcp_servers;
DROP TRIGGER IF EXISTS update_meta_updated_at ON alicia_meta;
DROP TRIGGER IF EXISTS update_commentaries_updated_at ON alicia_user_conversation_commentaries;
DROP TRIGGER IF EXISTS update_reasoning_steps_updated_at ON alicia_reasoning_steps;
DROP TRIGGER IF EXISTS update_tool_uses_updated_at ON alicia_tool_uses;
DROP TRIGGER IF EXISTS update_tools_updated_at ON alicia_tools;
DROP TRIGGER IF EXISTS update_memory_used_updated_at ON alicia_memory_used;
DROP TRIGGER IF EXISTS update_memory_updated_at ON alicia_memory;
DROP TRIGGER IF EXISTS update_audio_updated_at ON alicia_audio;
DROP TRIGGER IF EXISTS update_sentences_updated_at ON alicia_sentences;
DROP TRIGGER IF EXISTS update_messages_updated_at ON alicia_messages;
DROP TRIGGER IF EXISTS update_conversations_updated_at ON alicia_conversations;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS pareto_archive;
DROP TABLE IF EXISTS tool_usage_patterns;
DROP TABLE IF EXISTS tool_result_formatters;
DROP TABLE IF EXISTS optimized_tools;
DROP TABLE IF EXISTS optimized_programs;
DROP TABLE IF EXISTS prompt_evaluations;
DROP TABLE IF EXISTS prompt_candidates;
DROP TABLE IF EXISTS prompt_optimization_runs;
DROP TABLE IF EXISTS alicia_session_stats;
DROP TABLE IF EXISTS alicia_notes;
DROP TABLE IF EXISTS alicia_votes;
DROP TABLE IF EXISTS alicia_mcp_servers;
DROP TABLE IF EXISTS alicia_meta;
DROP TABLE IF EXISTS alicia_user_conversation_commentaries;
DROP TABLE IF EXISTS alicia_reasoning_steps;
DROP TABLE IF EXISTS alicia_tool_uses;
DROP TABLE IF EXISTS alicia_tools;
DROP TABLE IF EXISTS alicia_memory_used;
DROP TABLE IF EXISTS alicia_memory;
DROP TABLE IF EXISTS alicia_audio;
DROP TABLE IF EXISTS alicia_sentences;
DROP TABLE IF EXISTS alicia_messages;
DROP TABLE IF EXISTS alicia_conversations;

-- Drop enum types
DROP TYPE IF EXISTS optimization_status;
DROP TYPE IF EXISTS completion_status;
DROP TYPE IF EXISTS sync_status;
DROP TYPE IF EXISTS audio_type;
DROP TYPE IF EXISTS conversation_status;
DROP TYPE IF EXISTS tool_status;
DROP TYPE IF EXISTS message_role;

-- Drop helper function
DROP FUNCTION IF EXISTS generate_random_id(TEXT);

-- Note: We don't drop the extensions as they might be used by other schemas
-- DROP EXTENSION IF EXISTS vector;
-- DROP EXTENSION IF EXISTS pgcrypto;
