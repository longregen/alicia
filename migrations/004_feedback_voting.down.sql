-- Alicia Database Schema
-- Migration 004: Rollback feedback and voting system

-- Drop triggers
DROP TRIGGER IF EXISTS update_session_stats_updated_at ON alicia_session_stats;
DROP TRIGGER IF EXISTS update_notes_updated_at ON alicia_notes;
DROP TRIGGER IF EXISTS update_votes_updated_at ON alicia_votes;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS alicia_session_stats;
DROP TABLE IF EXISTS alicia_notes;
DROP TABLE IF EXISTS alicia_votes;
