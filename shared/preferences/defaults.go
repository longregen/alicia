// Package preferences provides shared preference defaults embedded at compile time.
package preferences

import (
	_ "embed"
	"encoding/json"
	"log"
)

//go:embed defaults.json
var defaultsJSON []byte

// Defaults holds all preference default values, parsed from defaults.json at init.
type Defaults struct {
	Theme                    string  `json:"theme"`
	AudioOutputEnabled       bool    `json:"audio_output_enabled"`
	VoiceSpeed               float32 `json:"voice_speed"`
	MemoryMinImportance      int     `json:"memory_min_importance"`
	MemoryMinHistorical      int     `json:"memory_min_historical"`
	MemoryMinPersonal        int     `json:"memory_min_personal"`
	MemoryMinFactual         int     `json:"memory_min_factual"`
	MemoryRetrievalCount     int     `json:"memory_retrieval_count"`
	MaxTokens                int     `json:"max_tokens"`
	Temperature              float32 `json:"temperature"`
	ParetoTargetScore        float32 `json:"pareto_target_score"`
	ParetoMaxGenerations     int     `json:"pareto_max_generations"`
	ParetoBranchesPerGen     int     `json:"pareto_branches_per_gen"`
	ParetoArchiveSize        int     `json:"pareto_archive_size"`
	ParetoEnableCrossover    bool    `json:"pareto_enable_crossover"`
	NotesSimilarityThreshold float32 `json:"notes_similarity_threshold"`
	NotesMaxCount            int     `json:"notes_max_count"`
	ConfirmDeleteMemory      bool    `json:"confirm_delete_memory"`
	ShowRelevanceScores      bool    `json:"show_relevance_scores"`
}

var defaults Defaults

func init() {
	if err := json.Unmarshal(defaultsJSON, &defaults); err != nil {
		log.Fatalf("failed to parse embedded preferences defaults: %v", err)
	}
}

// Get returns the parsed preference defaults.
func Get() Defaults {
	return defaults
}
