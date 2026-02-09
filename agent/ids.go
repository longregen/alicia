package main

import (
	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/longregen/alicia/shared/id"
)

func NewID(prefix string) string {
	raw, err := nanoid.New(id.DefaultLength)
	if err != nil {
		panic("nanoid generation failed: " + err.Error())
	}
	return prefix + raw
}

var (
	NewMessageID          = id.NewMessage
	NewMemoryID           = id.NewMemory
	NewMemoryUseID        = id.NewMemoryUse
	NewThinkingID         = id.NewThinking
	NewToolUseID          = id.NewToolUse
	NewMemoryTraceID      = id.NewMemoryTrace
	NewMemoryGenerationID = id.NewMemoryGeneration
)
