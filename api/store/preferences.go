package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// Returns nil if not found (caller should create defaults).
func (s *Store) GetUserPreferences(ctx context.Context, userID string) (*domain.UserPreferences, error) {
	query := `
		SELECT user_id, theme, audio_output_enabled, voice_speed,
		       memory_min_importance, memory_min_historical, memory_min_personal, memory_min_factual,
		       memory_retrieval_count, max_tokens, temperature,
		       pareto_target_score, pareto_max_generations, pareto_branches_per_gen, pareto_archive_size, pareto_enable_crossover,
		       notes_similarity_threshold, notes_max_count,
		       confirm_delete_memory, show_relevance_scores,
		       created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1`

	prefs := &domain.UserPreferences{}
	err := s.conn(ctx).QueryRow(ctx, query, userID).Scan(
		&prefs.UserID, &prefs.Theme, &prefs.AudioOutputEnabled, &prefs.VoiceSpeed,
		&prefs.MemoryMinImportance, &prefs.MemoryMinHistorical, &prefs.MemoryMinPersonal, &prefs.MemoryMinFactual,
		&prefs.MemoryRetrievalCount, &prefs.MaxTokens, &prefs.Temperature,
		&prefs.ParetoTargetScore, &prefs.ParetoMaxGenerations, &prefs.ParetoBranchesPerGen, &prefs.ParetoArchiveSize, &prefs.ParetoEnableCrossover,
		&prefs.NotesSimilarityThreshold, &prefs.NotesMaxCount,
		&prefs.ConfirmDeleteMemory, &prefs.ShowRelevanceScores,
		&prefs.CreatedAt, &prefs.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found, return nil
		}
		return nil, fmt.Errorf("get user preferences: %w", err)
	}
	return prefs, nil
}

func (s *Store) UpsertUserPreferences(ctx context.Context, prefs *domain.UserPreferences) error {
	query := `
		INSERT INTO user_preferences (
			user_id, theme, audio_output_enabled, voice_speed,
			memory_min_importance, memory_min_historical, memory_min_personal, memory_min_factual,
			memory_retrieval_count, max_tokens, temperature,
			pareto_target_score, pareto_max_generations, pareto_branches_per_gen, pareto_archive_size, pareto_enable_crossover,
			notes_similarity_threshold, notes_max_count,
			confirm_delete_memory, show_relevance_scores,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (user_id) DO UPDATE SET
			theme = EXCLUDED.theme,
			audio_output_enabled = EXCLUDED.audio_output_enabled,
			voice_speed = EXCLUDED.voice_speed,
			memory_min_importance = EXCLUDED.memory_min_importance,
			memory_min_historical = EXCLUDED.memory_min_historical,
			memory_min_personal = EXCLUDED.memory_min_personal,
			memory_min_factual = EXCLUDED.memory_min_factual,
			memory_retrieval_count = EXCLUDED.memory_retrieval_count,
			max_tokens = EXCLUDED.max_tokens,
			temperature = EXCLUDED.temperature,
			pareto_target_score = EXCLUDED.pareto_target_score,
			pareto_max_generations = EXCLUDED.pareto_max_generations,
			pareto_branches_per_gen = EXCLUDED.pareto_branches_per_gen,
			pareto_archive_size = EXCLUDED.pareto_archive_size,
			pareto_enable_crossover = EXCLUDED.pareto_enable_crossover,
			notes_similarity_threshold = EXCLUDED.notes_similarity_threshold,
			notes_max_count = EXCLUDED.notes_max_count,
			confirm_delete_memory = EXCLUDED.confirm_delete_memory,
			show_relevance_scores = EXCLUDED.show_relevance_scores,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now

	_, err := s.conn(ctx).Exec(ctx, query,
		prefs.UserID, prefs.Theme, prefs.AudioOutputEnabled, prefs.VoiceSpeed,
		prefs.MemoryMinImportance, prefs.MemoryMinHistorical, prefs.MemoryMinPersonal, prefs.MemoryMinFactual,
		prefs.MemoryRetrievalCount, prefs.MaxTokens, prefs.Temperature,
		prefs.ParetoTargetScore, prefs.ParetoMaxGenerations, prefs.ParetoBranchesPerGen, prefs.ParetoArchiveSize, prefs.ParetoEnableCrossover,
		prefs.NotesSimilarityThreshold, prefs.NotesMaxCount,
		prefs.ConfirmDeleteMemory, prefs.ShowRelevanceScores,
		prefs.CreatedAt, prefs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert user preferences: %w", err)
	}
	return nil
}
