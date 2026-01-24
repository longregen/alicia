// Package id provides ID generation helpers used across services.
package id

import (
	nanoid "github.com/matoous/go-nanoid/v2"
)

const DefaultLength = 21

const (
	PrefixConversation = "conv"
	PrefixMessage      = "msg"
	PrefixMemory       = "mem"
	PrefixMemoryUse    = "memu"
	PrefixTool         = "tool"
	PrefixToolUse      = "tu"
	PrefixNote         = "note"
	PrefixMCPServer    = "mcp"

	PrefixMessageFeedback   = "msgfb"
	PrefixToolUseFeedback   = "tufb"
	PrefixMemoryUseFeedback = "memufb"

	PrefixThinking    = "th"
	PrefixReasoning   = "rs"
	PrefixMemoryTrace = "mt"
)

func New(prefix string) string {
	id, err := nanoid.New(DefaultLength)
	if err != nil {
		panic("nanoid generation failed: " + err.Error())
	}
	return prefix + "_" + id
}

func NewWithLength(prefix string, length int) string {
	id, err := nanoid.New(length)
	if err != nil {
		panic("nanoid generation failed: " + err.Error())
	}
	return prefix + "_" + id
}

func NewConversation() string      { return New(PrefixConversation) }
func NewMessage() string           { return New(PrefixMessage) }
func NewMemory() string            { return New(PrefixMemory) }
func NewMemoryUse() string         { return New(PrefixMemoryUse) }
func NewTool() string              { return New(PrefixTool) }
func NewToolUse() string           { return New(PrefixToolUse) }
func NewNote() string              { return New(PrefixNote) }
func NewMCPServer() string         { return New(PrefixMCPServer) }
func NewMessageFeedback() string   { return New(PrefixMessageFeedback) }
func NewToolUseFeedback() string   { return New(PrefixToolUseFeedback) }
func NewMemoryUseFeedback() string { return New(PrefixMemoryUseFeedback) }
func NewThinking() string          { return New(PrefixThinking) }
func NewReasoning() string         { return New(PrefixReasoning) }
func NewMemoryTrace() string       { return New(PrefixMemoryTrace) }
