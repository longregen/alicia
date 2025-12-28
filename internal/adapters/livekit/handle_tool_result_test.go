package livekit

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

var errNotFound = errors.New("not found")

// Helper for tracking errors sent by dispatcher
type mockError struct {
	code        int32
	message     string
	recoverable bool
}

// Mock protocol handler with error tracking for tests
type mockProtocolHandlerWithErrors struct {
	conversationID string
	toolUseRepo    ports.ToolUseRepository
	errors         []mockError
}

func (m *mockProtocolHandlerWithErrors) HandleConfiguration(ctx context.Context, config *protocol.Configuration) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendEnvelope(ctx context.Context, envelope *protocol.Envelope) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendAudio(ctx context.Context, audio []byte, format string) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendToolUseRequest(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendToolUseResult(ctx context.Context, toolUse *models.ToolUse) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendAcknowledgement(ctx context.Context, ackedStanzaID int32, success bool) error {
	return nil
}

func (m *mockProtocolHandlerWithErrors) SendError(ctx context.Context, code int32, message string, recoverable bool) error {
	m.errors = append(m.errors, mockError{code: code, message: message, recoverable: recoverable})
	return nil
}

func (m *mockProtocolHandlerWithErrors) GetToolUseRepo() ports.ToolUseRepository {
	return m.toolUseRepo
}

// Mock ToolUseRepository for testing
type mockToolUseRepoForResult struct {
	toolUses map[string]*models.ToolUse
}

func newMockToolUseRepoForResult() *mockToolUseRepoForResult {
	return &mockToolUseRepoForResult{
		toolUses: make(map[string]*models.ToolUse),
	}
}

func (m *mockToolUseRepoForResult) Create(ctx context.Context, toolUse *models.ToolUse) error {
	m.toolUses[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepoForResult) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	if tu, ok := m.toolUses[id]; ok {
		return tu, nil
	}
	return nil, errNotFound
}

func (m *mockToolUseRepoForResult) Update(ctx context.Context, toolUse *models.ToolUse) error {
	if _, ok := m.toolUses[toolUse.ID]; !ok {
		return errNotFound
	}
	m.toolUses[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepoForResult) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	var result []*models.ToolUse
	for _, tu := range m.toolUses {
		if tu.MessageID == messageID {
			result = append(result, tu)
		}
	}
	return result, nil
}

func (m *mockToolUseRepoForResult) GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	return nil, nil
}

func TestHandleToolUseResult_Success(t *testing.T) {
	ctx := context.Background()
	conversationID := "conv_test"
	toolUseID := "tool_use_123"
	messageID := "msg_456"

	// Create mock repositories
	toolUseRepo := newMockToolUseRepoForResult()

	// Create a pending tool use
	toolUse := models.NewToolUse(toolUseID, messageID, "test_tool", 1, map[string]any{"param": "value"})
	toolUseRepo.Create(ctx, toolUse)

	// Create mock protocol handler
	mockHandler := &mockProtocolHandlerWithErrors{
		conversationID: conversationID,
		toolUseRepo:    toolUseRepo,
	}

	// Create dispatcher
	dispatcher := &DefaultMessageDispatcher{
		protocolHandler: mockHandler,
		conversationID:  conversationID,
		toolUseRepo:     toolUseRepo,
	}

	// Create a successful tool result
	result := &protocol.ToolUseResult{
		ID:             "result_123",
		RequestID:      toolUseID,
		ConversationID: conversationID,
		Success:        true,
		Result:         map[string]any{"output": "success data"},
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           result,
	}

	// Handle the tool result
	err := dispatcher.handleToolUseResult(ctx, envelope)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the tool use was updated
	updatedToolUse, err := toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		t.Fatalf("Failed to get updated tool use: %v", err)
	}

	if updatedToolUse.Status != models.ToolStatusSuccess {
		t.Errorf("Expected status %v, got %v", models.ToolStatusSuccess, updatedToolUse.Status)
	}

	if updatedToolUse.Result == nil {
		t.Error("Expected result to be set")
	}

	if updatedToolUse.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestHandleToolUseResult_Failure(t *testing.T) {
	ctx := context.Background()
	conversationID := "conv_test"
	toolUseID := "tool_use_123"
	messageID := "msg_456"

	// Create mock repositories
	toolUseRepo := newMockToolUseRepoForResult()

	// Create a pending tool use
	toolUse := models.NewToolUse(toolUseID, messageID, "test_tool", 1, map[string]any{"param": "value"})
	toolUseRepo.Create(ctx, toolUse)

	// Create mock protocol handler
	mockHandler := &mockProtocolHandlerWithErrors{
		conversationID: conversationID,
		toolUseRepo:    toolUseRepo,
	}

	// Create dispatcher
	dispatcher := &DefaultMessageDispatcher{
		protocolHandler: mockHandler,
		conversationID:  conversationID,
		toolUseRepo:     toolUseRepo,
	}

	// Create a failed tool result
	result := &protocol.ToolUseResult{
		ID:             "result_123",
		RequestID:      toolUseID,
		ConversationID: conversationID,
		Success:        false,
		ErrorCode:      "EXECUTION_ERROR",
		ErrorMessage:   "Tool execution failed",
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           result,
	}

	// Handle the tool result
	err := dispatcher.handleToolUseResult(ctx, envelope)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the tool use was updated
	updatedToolUse, err := toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		t.Fatalf("Failed to get updated tool use: %v", err)
	}

	if updatedToolUse.Status != models.ToolStatusError {
		t.Errorf("Expected status %v, got %v", models.ToolStatusError, updatedToolUse.Status)
	}

	expectedError := "EXECUTION_ERROR: Tool execution failed"
	if updatedToolUse.ErrorMessage != expectedError {
		t.Errorf("Expected error message %q, got %q", expectedError, updatedToolUse.ErrorMessage)
	}

	if updatedToolUse.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestHandleToolUseResult_NotFound(t *testing.T) {
	ctx := context.Background()
	conversationID := "conv_test"
	toolUseID := "tool_use_nonexistent"

	// Create mock repositories
	toolUseRepo := newMockToolUseRepoForResult()

	// Create mock protocol handler with error tracking
	mockHandler := &mockProtocolHandlerWithErrors{
		conversationID: conversationID,
		toolUseRepo:    toolUseRepo,
		errors:         []mockError{},
	}

	// Create dispatcher
	dispatcher := &DefaultMessageDispatcher{
		protocolHandler: mockHandler,
		conversationID:  conversationID,
		toolUseRepo:     toolUseRepo,
	}

	// Create a tool result for a non-existent tool use
	result := &protocol.ToolUseResult{
		ID:             "result_123",
		RequestID:      toolUseID,
		ConversationID: conversationID,
		Success:        true,
		Result:         map[string]any{"output": "success data"},
	}

	envelope := &protocol.Envelope{
		ConversationID: conversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           result,
	}

	// Handle the tool result - should return an error
	err := dispatcher.handleToolUseResult(ctx, envelope)
	if err == nil {
		t.Fatal("Expected error for non-existent tool use, got nil")
	}

	// Verify error was sent to client
	if len(mockHandler.errors) == 0 {
		t.Error("Expected error to be sent to client")
	}
}

func TestHandleToolUseResult_ConversationMismatch(t *testing.T) {
	ctx := context.Background()
	conversationID := "conv_test"
	wrongConversationID := "conv_wrong"

	// Create mock repositories
	toolUseRepo := newMockToolUseRepoForResult()

	// Create mock protocol handler with error tracking
	mockHandler := &mockProtocolHandlerWithErrors{
		conversationID: conversationID,
		toolUseRepo:    toolUseRepo,
		errors:         []mockError{},
	}

	// Create dispatcher
	dispatcher := &DefaultMessageDispatcher{
		protocolHandler: mockHandler,
		conversationID:  conversationID,
		toolUseRepo:     toolUseRepo,
	}

	// Create a tool result with wrong conversation ID
	result := &protocol.ToolUseResult{
		ID:             "result_123",
		RequestID:      "tool_use_123",
		ConversationID: wrongConversationID,
		Success:        true,
		Result:         map[string]any{"output": "success data"},
	}

	envelope := &protocol.Envelope{
		ConversationID: wrongConversationID,
		Type:           protocol.TypeToolUseResult,
		Body:           result,
	}

	// Handle the tool result - should return an error
	err := dispatcher.handleToolUseResult(ctx, envelope)
	if err == nil {
		t.Fatal("Expected error for conversation mismatch, got nil")
	}

	// Verify correct error code was sent
	if len(mockHandler.errors) == 0 {
		t.Fatal("Expected error to be sent to client")
	}

	if mockHandler.errors[0].code != protocol.ErrCodeConversationNotFound {
		t.Errorf("Expected error code %v, got %v", protocol.ErrCodeConversationNotFound, mockHandler.errors[0].code)
	}
}
