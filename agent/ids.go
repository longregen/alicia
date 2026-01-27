package main

import (
	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/longregen/alicia/shared/id"
)

// NewID generates a new ID with the given prefix (no separator added).
// For prefixes that should include underscore, pass "prefix_".
// This maintains backward compatibility with legacy code.
func NewID(prefix string) string {
	raw, err := nanoid.New(id.DefaultLength)
	if err != nil {
		panic("nanoid generation failed: " + err.Error())
	}
	return prefix + raw
}

// Re-export ID functions from internal/id for backward compatibility
var (
	NewMessageID     = id.NewMessage
	NewMemoryID      = id.NewMemory
	NewMemoryUseID   = id.NewMemoryUse
	NewThinkingID    = id.NewThinking
	NewReasoningID   = id.NewReasoning
	NewToolUseID     = id.NewToolUse
	NewMemoryTraceID      = id.NewMemoryTrace
	NewMemoryGenerationID = id.NewMemoryGeneration
)
