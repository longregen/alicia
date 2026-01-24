package services

import (
	"context"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
	"github.com/longregen/alicia/shared/ptr"
)

type PreferencesService struct {
	store *store.Store
}

func NewPreferencesService(s *store.Store) *PreferencesService {
	return &PreferencesService{store: s}
}

func DefaultPreferences(userID string) *domain.UserPreferences {
	now := time.Now().UTC()
	return &domain.UserPreferences{
		UserID:                   userID,
		Theme:                    domain.ThemeSystem,
		AudioOutputEnabled:       false,
		VoiceSpeed:               1.0,
		MemoryMinImportance:      ptr.To(3),
		MemoryMinHistorical:      ptr.To(2),
		MemoryMinPersonal:        ptr.To(2),
		MemoryMinFactual:         ptr.To(2),
		MemoryRetrievalCount:     4,
		MaxTokens:                16384,
		ParetoTargetScore:        3.0,
		ParetoMaxGenerations:     7,
		ParetoBranchesPerGen:     3,
		ParetoArchiveSize:        50,
		ParetoEnableCrossover:    true,
		NotesSimilarityThreshold: 0.7,
		NotesMaxCount:            3,
		ConfirmDeleteMemory:      true,
		ShowRelevanceScores:      false,
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
