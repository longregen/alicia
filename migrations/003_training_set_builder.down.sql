-- Migration 003 Rollback: Training Set Builder for GEPA Optimization

ALTER TABLE alicia_conversations DROP COLUMN IF EXISTS system_prompt_version_id;
DROP TABLE IF EXISTS gepa_training_examples;
DROP TABLE IF EXISTS system_prompt_versions;
