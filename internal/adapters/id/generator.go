package id

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) generate(prefix string) string {
	id, err := gonanoid.New(21)
	if err != nil {
		return prefix + "_fallback"
	}
	return prefix + "_" + id
}

func (g *Generator) GenerateConversationID() string {
	return g.generate("ac")
}

func (g *Generator) GenerateMessageID() string {
	return g.generate("am")
}

func (g *Generator) GenerateSentenceID() string {
	return g.generate("ams")
}

func (g *Generator) GenerateAudioID() string {
	return g.generate("aa")
}

func (g *Generator) GenerateMemoryID() string {
	return g.generate("amem")
}

func (g *Generator) GenerateMemoryUsageID() string {
	return g.generate("amu")
}

func (g *Generator) GenerateToolID() string {
	return g.generate("at")
}

func (g *Generator) GenerateToolUseID() string {
	return g.generate("atu")
}

func (g *Generator) GenerateReasoningStepID() string {
	return g.generate("ar")
}

func (g *Generator) GenerateCommentaryID() string {
	return g.generate("aucc")
}

func (g *Generator) GenerateMetaID() string {
	return g.generate("amt")
}

func (g *Generator) GenerateMCPServerID() string {
	return g.generate("amcp")
}

func (g *Generator) GenerateLiveKitRoomName() string {
	return g.generate("room")
}

func (g *Generator) GenerateVoteID() string {
	return g.generate("av")
}

func (g *Generator) GenerateNoteID() string {
	return g.generate("an")
}

func (g *Generator) GenerateSessionStatsID() string {
	return g.generate("ass")
}

func (g *Generator) GenerateOptimizationRunID() string {
	return g.generate("aor")
}

func (g *Generator) GeneratePromptCandidateID() string {
	return g.generate("apc")
}

func (g *Generator) GeneratePromptEvaluationID() string {
	return g.generate("ape")
}

func (g *Generator) GenerateTrainingExampleID() string {
	return g.generate("gte")
}

func (g *Generator) GenerateSystemPromptVersionID() string {
	return g.generate("spv")
}
