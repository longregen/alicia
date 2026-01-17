package services

import (
	"context"
	"fmt"
)

// Shared mock implementations for testing

type mockIDGenerator struct {
	conversationCounter int
	messageCounter      int
	sentenceCounter     int
	audioCounter        int
	memoryCounter       int
	memoryUsageCounter  int
	toolCounter         int
	toolUseCounter      int
	reasoningCounter    int
	commentaryCounter   int
	metaCounter         int
}

func (m *mockIDGenerator) GenerateConversationID() string {
	m.conversationCounter++
	return fmt.Sprintf("ac_test%d", m.conversationCounter)
}

func (m *mockIDGenerator) GenerateMessageID() string {
	m.messageCounter++
	return fmt.Sprintf("msg_test%d", m.messageCounter)
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	m.sentenceCounter++
	return fmt.Sprintf("sent_test%d", m.sentenceCounter)
}

func (m *mockIDGenerator) GenerateAudioID() string {
	m.audioCounter++
	return fmt.Sprintf("audio_test%d", m.audioCounter)
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	m.memoryCounter++
	return fmt.Sprintf("mem_test%d", m.memoryCounter)
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	m.memoryUsageCounter++
	return fmt.Sprintf("mu_test%d", m.memoryUsageCounter)
}

func (m *mockIDGenerator) GenerateToolID() string {
	m.toolCounter++
	return fmt.Sprintf("tool_test%d", m.toolCounter)
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	m.toolUseCounter++
	return fmt.Sprintf("tu_test%d", m.toolUseCounter)
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	m.reasoningCounter++
	return fmt.Sprintf("rs_test%d", m.reasoningCounter)
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	m.commentaryCounter++
	return fmt.Sprintf("comm_test%d", m.commentaryCounter)
}

func (m *mockIDGenerator) GenerateMetaID() string {
	m.metaCounter++
	return fmt.Sprintf("meta_test%d", m.metaCounter)
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	return "amcp_test"
}

func (m *mockIDGenerator) GenerateVoteID() string {
	return "av_test"
}

func (m *mockIDGenerator) GenerateNoteID() string {
	return "an_test"
}

func (m *mockIDGenerator) GenerateSessionStatsID() string {
	return "ass_test"
}

func (m *mockIDGenerator) GenerateOptimizationRunID() string {
	return "aor_test"
}

func (m *mockIDGenerator) GeneratePromptCandidateID() string {
	return "apc_test"
}

func (m *mockIDGenerator) GeneratePromptEvaluationID() string {
	return "ape_test"
}

func (m *mockIDGenerator) GenerateTrainingExampleID() string {
	return "gte_test"
}

func (m *mockIDGenerator) GenerateSystemPromptVersionID() string {
	return "spv_test"
}

func (m *mockIDGenerator) GenerateRequestID() string {
	return "areq_test"
}

type mockTransactionManager struct{}

func (m *mockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Simply execute the function without actual transaction management
	return fn(ctx)
}
