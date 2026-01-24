package main

import (
	"sync"

	"github.com/longregen/alicia/shared/protocol"
	"github.com/longregen/alicia/shared/ptr"
)

type UserPreferences struct {
	MemoryMinImportance      *int
	MemoryMinHistorical      *int
	MemoryMinPersonal        *int
	MemoryMinFactual         *int
	MemoryRetrievalCount     int
	MaxTokens                int
	ParetoTargetScore        float32
	ParetoMaxGenerations     int
	ParetoBranchesPerGen     int
	ParetoArchiveSize        int
	ParetoEnableCrossover    bool
	NotesSimilarityThreshold float32
	NotesMaxCount            int
}

func DefaultPreferences() UserPreferences {
	return UserPreferences{
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
	}
}

type PreferencesStore struct {
	mu    sync.RWMutex
	prefs map[string]UserPreferences
}

func NewPreferencesStore() *PreferencesStore {
	return &PreferencesStore{
		prefs: make(map[string]UserPreferences),
	}
}

func (s *PreferencesStore) Get(userID string) UserPreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.prefs[userID]; ok {
		return p
	}
	return DefaultPreferences()
}

func (s *PreferencesStore) Update(update protocol.PreferencesUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := UserPreferences{
		MemoryMinImportance:      update.MemoryMinImportance,
		MemoryMinHistorical:      update.MemoryMinHistorical,
		MemoryMinPersonal:        update.MemoryMinPersonal,
		MemoryMinFactual:         update.MemoryMinFactual,
		MemoryRetrievalCount:     update.MemoryRetrievalCount,
		MaxTokens:                update.MaxTokens,
		ParetoTargetScore:        update.ParetoTargetScore,
		ParetoMaxGenerations:     update.ParetoMaxGenerations,
		ParetoBranchesPerGen:     update.ParetoBranchesPerGen,
		ParetoArchiveSize:        update.ParetoArchiveSize,
		ParetoEnableCrossover:    update.ParetoEnableCrossover,
		NotesSimilarityThreshold: update.NotesSimilarityThreshold,
		NotesMaxCount:            update.NotesMaxCount,
	}

	defaults := DefaultPreferences()
	if p.MemoryRetrievalCount == 0 {
		p.MemoryRetrievalCount = defaults.MemoryRetrievalCount
	}
	if p.MaxTokens == 0 {
		p.MaxTokens = defaults.MaxTokens
	}
	if p.ParetoTargetScore == 0 {
		p.ParetoTargetScore = defaults.ParetoTargetScore
	}
	if p.ParetoMaxGenerations == 0 {
		p.ParetoMaxGenerations = defaults.ParetoMaxGenerations
	}
	if p.ParetoBranchesPerGen == 0 {
		p.ParetoBranchesPerGen = defaults.ParetoBranchesPerGen
	}
	if p.ParetoArchiveSize == 0 {
		p.ParetoArchiveSize = defaults.ParetoArchiveSize
	}
	if p.NotesSimilarityThreshold == 0 {
		p.NotesSimilarityThreshold = defaults.NotesSimilarityThreshold
	}
	if p.NotesMaxCount == 0 {
		p.NotesMaxCount = defaults.NotesMaxCount
	}

	s.prefs[update.UserID] = p
}

func (s *PreferencesStore) GetThresholds(userID string) MemoryThresholds {
	p := s.Get(userID)
	return MemoryThresholds{
		Importance: p.MemoryMinImportance,
		Historical: p.MemoryMinHistorical,
		Personal:   p.MemoryMinPersonal,
		Factual:    p.MemoryMinFactual,
	}
}

