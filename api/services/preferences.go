package services

import (
	"context"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/shared/preferences"
	"github.com/longregen/alicia/shared/ptr"
)

type PreferencesService struct {
	store *store.Store
}

func NewPreferencesService(s *store.Store) *PreferencesService {
	return &PreferencesService{store: s}
}

func DefaultPreferences(userID string) *domain.UserPreferences {
	d := preferences.Get()
	now := time.Now().UTC()
	return &domain.UserPreferences{
		UserID:                   userID,
		Theme:                    d.Theme,
		AudioOutputEnabled:       d.AudioOutputEnabled,
		VoiceSpeed:               d.VoiceSpeed,
		MemoryMinImportance:      ptr.To(d.MemoryMinImportance),
		MemoryMinHistorical:      ptr.To(d.MemoryMinHistorical),
		MemoryMinPersonal:        ptr.To(d.MemoryMinPersonal),
		MemoryMinFactual:         ptr.To(d.MemoryMinFactual),
		MemoryRetrievalCount:     d.MemoryRetrievalCount,
		MaxTokens:                d.MaxTokens,
		Temperature:              d.Temperature,
		ParetoTargetScore:        d.ParetoTargetScore,
		ParetoMaxGenerations:     d.ParetoMaxGenerations,
		ParetoBranchesPerGen:     d.ParetoBranchesPerGen,
		ParetoArchiveSize:        d.ParetoArchiveSize,
		ParetoEnableCrossover:    d.ParetoEnableCrossover,
		NotesSimilarityThreshold: d.NotesSimilarityThreshold,
		NotesMaxCount:            d.NotesMaxCount,
		ConfirmDeleteMemory:      d.ConfirmDeleteMemory,
		ShowRelevanceScores:      d.ShowRelevanceScores,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

func (svc *PreferencesService) GetPreferences(ctx context.Context, userID string) (*domain.UserPreferences, error) {
	prefs, err := svc.store.GetUserPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create defaults if not found
	if prefs == nil {
		prefs = DefaultPreferences(userID)
		if err := svc.store.UpsertUserPreferences(ctx, prefs); err != nil {
			return nil, err
		}
	}

	return prefs, nil
}

// The updates struct should contain all preference fields (handler merges current values).
func (svc *PreferencesService) UpdatePreferences(ctx context.Context, userID string, updates *domain.UserPreferences) (*domain.UserPreferences, error) {
	updates.UserID = userID
	if err := svc.store.UpsertUserPreferences(ctx, updates); err != nil {
		return nil, err
	}
	return updates, nil
}
