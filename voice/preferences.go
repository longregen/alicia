package main

import (
	"sync"

	"github.com/longregen/alicia/shared/protocol"
)

type VoicePreferences struct {
	Speed float64
}

func DefaultVoicePreferences() VoicePreferences {
	return VoicePreferences{Speed: 1.0}
}

type VoicePreferencesStore struct {
	mu    sync.RWMutex
	prefs map[string]VoicePreferences
}

func NewVoicePreferencesStore() *VoicePreferencesStore {
	return &VoicePreferencesStore{
		prefs: make(map[string]VoicePreferences),
	}
}

func (s *VoicePreferencesStore) Get(userID string) VoicePreferences {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.prefs[userID]; ok {
		return p
	}
	return DefaultVoicePreferences()
}

func (s *VoicePreferencesStore) Update(update protocol.PreferencesUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	speed := float64(update.VoiceSpeed)
	if speed <= 0 {
		speed = 1.0
	}

	s.prefs[update.UserID] = VoicePreferences{Speed: speed}
}
