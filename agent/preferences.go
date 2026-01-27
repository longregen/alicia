package main

import (
	"sync"

	"github.com/longregen/alicia/shared/config"
	"github.com/longregen/alicia/shared/preferences"
	"github.com/longregen/alicia/shared/protocol"
	"github.com/longregen/alicia/shared/ptr"
)

// defaultTemperature returns the default temperature from env var or shared default.
func defaultTemperature() float32 {
	return float32(config.GetEnvFloat("ALICIA_LLM_TEMPERATURE", float64(preferences.Get().Temperature)))
}

type UserPreferences struct {
	MemoryMinImportance      *int
	MemoryMinHistorical      *int
	MemoryMinPersonal        *int
	MemoryMinFactual         *int
	MemoryRetrievalCount     int
	MaxTokens                int
	MaxToolIterations        int
	Temperature              float32
	ParetoTargetScore        float32
	ParetoMaxGenerations     int
	ParetoBranchesPerGen     int
	ParetoArchiveSize        int
	ParetoEnableCrossover    bool
	NotesSimilarityThreshold float32
	NotesMaxCount            int
}

func DefaultPreferences() UserPreferences {
	d := preferences.Get()
	return UserPreferences{
		MemoryMinImportance:      ptr.To(d.MemoryMinImportance),
		MemoryMinHistorical:      ptr.To(d.MemoryMinHistorical),
		MemoryMinPersonal:        ptr.To(d.MemoryMinPersonal),
		MemoryMinFactual:         ptr.To(d.MemoryMinFactual),
		MemoryRetrievalCount:     d.MemoryRetrievalCount,
		MaxTokens:                d.MaxTokens,
		MaxToolIterations:        d.MaxToolIterations,
		Temperature:              defaultTemperature(),
		ParetoTargetScore:        d.ParetoTargetScore,
		ParetoMaxGenerations:     d.ParetoMaxGenerations,
		ParetoBranchesPerGen:     d.ParetoBranchesPerGen,
		ParetoArchiveSize:        d.ParetoArchiveSize,
		ParetoEnableCrossover:    d.ParetoEnableCrossover,
		NotesSimilarityThreshold: d.NotesSimilarityThreshold,
		NotesMaxCount:            d.NotesMaxCount,
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
		Temperature:              update.Temperature,
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
	if p.MaxToolIterations == 0 {
		p.MaxToolIterations = defaults.MaxToolIterations
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


