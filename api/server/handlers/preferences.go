package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/services"
)

func validateThreshold(name string, value *int, min, max int) (*int, error) {
	if value == nil {
		return nil, nil
	}
	if *value < min || *value > max {
		return nil, fmt.Errorf("%s must be %d-%d or null", name, min, max)
	}
	return value, nil
}

type PreferencesBroadcaster interface {
	BroadcastPreferencesUpdate(prefs *domain.UserPreferences)
}

type PreferencesHandler struct {
	prefsSvc *services.PreferencesService
	hub      PreferencesBroadcaster
}

func NewPreferencesHandler(svc *services.PreferencesService, hub PreferencesBroadcaster) *PreferencesHandler {
	return &PreferencesHandler{prefsSvc: svc, hub: hub}
}

func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, "user ID required", http.StatusBadRequest)
		return
	}

	prefs, err := h.prefsSvc.GetPreferences(r.Context(), userID)
	if err != nil {
		respondError(w, "failed to get preferences", http.StatusInternalServerError)
		return
	}

	respondJSON(w, prefs, http.StatusOK)
}

func (h *PreferencesHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, "user ID required", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse raw JSON to detect which fields are present (for null vs absent distinction)
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &rawFields); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req struct {
		Theme                    *string  `json:"theme"`
		AudioOutputEnabled       *bool    `json:"audio_output_enabled"`
		VoiceSpeed               *float32 `json:"voice_speed"`
		MemoryMinImportance      *int     `json:"memory_min_importance"`
		MemoryMinHistorical      *int     `json:"memory_min_historical"`
		MemoryMinPersonal        *int     `json:"memory_min_personal"`
		MemoryMinFactual         *int     `json:"memory_min_factual"`
		MemoryRetrievalCount     *int     `json:"memory_retrieval_count"`
		MaxTokens                *int     `json:"max_tokens"`
		ParetoTargetScore        *float32 `json:"pareto_target_score"`
		ParetoMaxGenerations     *int     `json:"pareto_max_generations"`
		ParetoBranchesPerGen     *int     `json:"pareto_branches_per_gen"`
		ParetoArchiveSize        *int     `json:"pareto_archive_size"`
		ParetoEnableCrossover    *bool    `json:"pareto_enable_crossover"`
		NotesSimilarityThreshold *float32 `json:"notes_similarity_threshold"`
		NotesMaxCount            *int     `json:"notes_max_count"`
		ConfirmDeleteMemory      *bool    `json:"confirm_delete_memory"`
		ShowRelevanceScores      *bool    `json:"show_relevance_scores"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		respondError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get current preferences to merge with updates
	current, err := h.prefsSvc.GetPreferences(r.Context(), userID)
	if err != nil {
		respondError(w, "failed to get preferences", http.StatusInternalServerError)
		return
	}

	// Build updates struct with current values as base
	updates := &domain.UserPreferences{
		UserID:                   userID,
		Theme:                    current.Theme,
		AudioOutputEnabled:       current.AudioOutputEnabled,
		VoiceSpeed:               current.VoiceSpeed,
		MemoryMinImportance:      current.MemoryMinImportance,
		MemoryMinHistorical:      current.MemoryMinHistorical,
		MemoryMinPersonal:        current.MemoryMinPersonal,
		MemoryMinFactual:         current.MemoryMinFactual,
		MemoryRetrievalCount:     current.MemoryRetrievalCount,
		MaxTokens:                current.MaxTokens,
		ParetoTargetScore:        current.ParetoTargetScore,
		ParetoMaxGenerations:     current.ParetoMaxGenerations,
		ParetoBranchesPerGen:     current.ParetoBranchesPerGen,
		ParetoArchiveSize:        current.ParetoArchiveSize,
		ParetoEnableCrossover:    current.ParetoEnableCrossover,
		NotesSimilarityThreshold: current.NotesSimilarityThreshold,
		NotesMaxCount:            current.NotesMaxCount,
		ConfirmDeleteMemory:      current.ConfirmDeleteMemory,
		ShowRelevanceScores:      current.ShowRelevanceScores,
	}

	// Validate and apply fields
	if req.Theme != nil {
		if *req.Theme != "light" && *req.Theme != "dark" && *req.Theme != "system" {
			respondError(w, "theme must be light, dark, or system", http.StatusBadRequest)
			return
		}
		updates.Theme = *req.Theme
	}
	if req.AudioOutputEnabled != nil {
		updates.AudioOutputEnabled = *req.AudioOutputEnabled
	}
	if req.VoiceSpeed != nil {
		if *req.VoiceSpeed < 0.5 || *req.VoiceSpeed > 2.0 {
			respondError(w, "voice_speed must be 0.5-2.0", http.StatusBadRequest)
			return
		}
		updates.VoiceSpeed = *req.VoiceSpeed
	}

	// Memory threshold fields support null (disables filtering on that dimension).
	// We use rawFields to distinguish absent (keep current) from explicit null (clear).
	thresholdFields := []struct {
		jsonKey string
		reqVal  *int
		dest    **int
	}{
		{"memory_min_importance", req.MemoryMinImportance, &updates.MemoryMinImportance},
		{"memory_min_historical", req.MemoryMinHistorical, &updates.MemoryMinHistorical},
		{"memory_min_personal", req.MemoryMinPersonal, &updates.MemoryMinPersonal},
		{"memory_min_factual", req.MemoryMinFactual, &updates.MemoryMinFactual},
	}
	for _, tf := range thresholdFields {
		if _, present := rawFields[tf.jsonKey]; present {
			val, err := validateThreshold(tf.jsonKey, tf.reqVal, 1, 5)
			if err != nil {
				respondError(w, err.Error(), http.StatusBadRequest)
				return
			}
			*tf.dest = val
		}
	}

	if req.MemoryRetrievalCount != nil {
		if *req.MemoryRetrievalCount < 1 || *req.MemoryRetrievalCount > 50 {
			respondError(w, "memory_retrieval_count must be 1-50", http.StatusBadRequest)
			return
		}
		updates.MemoryRetrievalCount = *req.MemoryRetrievalCount
	}
	if req.MaxTokens != nil {
		if *req.MaxTokens < 256 || *req.MaxTokens > 16384 {
			respondError(w, "max_tokens must be 256-16384", http.StatusBadRequest)
			return
		}
		updates.MaxTokens = *req.MaxTokens
	}
	if req.ParetoTargetScore != nil {
		if *req.ParetoTargetScore < 0.5 || *req.ParetoTargetScore > 5.0 {
			respondError(w, "pareto_target_score must be 0.5-5.0", http.StatusBadRequest)
			return
		}
		updates.ParetoTargetScore = *req.ParetoTargetScore
	}
	if req.ParetoMaxGenerations != nil {
		if *req.ParetoMaxGenerations < 1 || *req.ParetoMaxGenerations > 20 {
			respondError(w, "pareto_max_generations must be 1-20", http.StatusBadRequest)
			return
		}
		updates.ParetoMaxGenerations = *req.ParetoMaxGenerations
	}
	if req.ParetoBranchesPerGen != nil {
		if *req.ParetoBranchesPerGen < 1 || *req.ParetoBranchesPerGen > 10 {
			respondError(w, "pareto_branches_per_gen must be 1-10", http.StatusBadRequest)
			return
		}
		updates.ParetoBranchesPerGen = *req.ParetoBranchesPerGen
	}
	if req.ParetoArchiveSize != nil {
		if *req.ParetoArchiveSize < 10 || *req.ParetoArchiveSize > 200 {
			respondError(w, "pareto_archive_size must be 10-200", http.StatusBadRequest)
			return
		}
		updates.ParetoArchiveSize = *req.ParetoArchiveSize
	}
	if req.ParetoEnableCrossover != nil {
		updates.ParetoEnableCrossover = *req.ParetoEnableCrossover
	}
	if req.NotesSimilarityThreshold != nil {
		if *req.NotesSimilarityThreshold < 0.0 || *req.NotesSimilarityThreshold > 1.0 {
			respondError(w, "notes_similarity_threshold must be 0.0-1.0", http.StatusBadRequest)
			return
		}
		updates.NotesSimilarityThreshold = *req.NotesSimilarityThreshold
	}
	if req.NotesMaxCount != nil {
		if *req.NotesMaxCount < 0 || *req.NotesMaxCount > 20 {
			respondError(w, "notes_max_count must be 0-20", http.StatusBadRequest)
			return
		}
		updates.NotesMaxCount = *req.NotesMaxCount
	}
	if req.ConfirmDeleteMemory != nil {
		updates.ConfirmDeleteMemory = *req.ConfirmDeleteMemory
	}
	if req.ShowRelevanceScores != nil {
		updates.ShowRelevanceScores = *req.ShowRelevanceScores
	}

	prefs, err := h.prefsSvc.UpdatePreferences(r.Context(), userID, updates)
	if err != nil {
		respondError(w, "failed to update preferences", http.StatusInternalServerError)
		return
	}

	// Broadcast preferences update to agent and voice services
	if h.hub != nil {
		h.hub.BroadcastPreferencesUpdate(prefs)
	}

	respondJSON(w, prefs, http.StatusOK)
}
