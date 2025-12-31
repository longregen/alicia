package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

var (
	errToolNotFound = errors.New("not found")
)

// Mock implementations

type mockToolRepo struct {
	store map[string]*models.Tool
}

func newMockToolRepo() *mockToolRepo {
	return &mockToolRepo{
		store: make(map[string]*models.Tool),
	}
}

func (m *mockToolRepo) Create(ctx context.Context, tool *models.Tool) error {
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	if tool, ok := m.store[id]; ok {
		return tool, nil
	}
	return nil, errToolNotFound
}

func (m *mockToolRepo) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	for _, tool := range m.store {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, errToolNotFound
}

func (m *mockToolRepo) Update(ctx context.Context, tool *models.Tool) error {
	if _, ok := m.store[tool.ID]; !ok {
		return errToolNotFound
	}
	m.store[tool.ID] = tool
	return nil
}

func (m *mockToolRepo) Delete(ctx context.Context, id string) error {
	if tool, ok := m.store[id]; ok {
		now := time.Now()
		tool.DeletedAt = &now
		tool.UpdatedAt = now
		m.store[id] = tool
		return nil
	}
	return errToolNotFound
}

func (m *mockToolRepo) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	tools := make([]*models.Tool, 0)
	for _, tool := range m.store {
		if tool.Enabled && tool.DeletedAt == nil {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func (m *mockToolRepo) ListAll(ctx context.Context) ([]*models.Tool, error) {
	tools := make([]*models.Tool, 0)
	for _, tool := range m.store {
		if tool.DeletedAt == nil {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

type mockToolUseRepo struct {
	store map[string]*models.ToolUse
}

func newMockToolUseRepo() *mockToolUseRepo {
	return &mockToolUseRepo{
		store: make(map[string]*models.ToolUse),
	}
}

func (m *mockToolUseRepo) Create(ctx context.Context, toolUse *models.ToolUse) error {
	m.store[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepo) GetByID(ctx context.Context, id string) (*models.ToolUse, error) {
	if toolUse, ok := m.store[id]; ok {
		return toolUse, nil
	}
	return nil, errToolNotFound
}

func (m *mockToolUseRepo) Update(ctx context.Context, toolUse *models.ToolUse) error {
	if _, ok := m.store[toolUse.ID]; !ok {
		return errToolNotFound
	}
	m.store[toolUse.ID] = toolUse
	return nil
}

func (m *mockToolUseRepo) GetByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	toolUses := make([]*models.ToolUse, 0)
	for _, tu := range m.store {
		if tu.MessageID == messageID {
			toolUses = append(toolUses, tu)
		}
	}
	return toolUses, nil
}

func (m *mockToolUseRepo) GetPending(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	toolUses := make([]*models.ToolUse, 0)
	count := 0
	for _, tu := range m.store {
		if tu.Status == models.ToolStatusPending {
			toolUses = append(toolUses, tu)
			count++
			if count >= limit {
				break
			}
		}
	}
	return toolUses, nil
}

type mockMessageRepo struct {
	store map[string]*models.Message
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		store: make(map[string]*models.Message),
	}
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.store[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if msg, ok := m.store[id]; ok {
		return msg, nil
	}
	return nil, errToolNotFound
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	if _, ok := m.store[msg.ID]; !ok {
		return errToolNotFound
	}
	m.store[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *mockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	messages := make([]*models.Message, 0)
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	messages := make([]*models.Message, 0)
	for _, msg := range m.store {
		if msg.ConversationID == conversationID {
			messages = append(messages, msg)
			if len(messages) >= limit {
				break
			}
		}
	}
	return messages, nil
}

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return 0, nil
}

func (m *mockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errToolNotFound
}

func (m *mockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *mockMessageRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	// Simple implementation: return message chain by following PreviousID
	var chain []*models.Message
	currentID := tipMessageID

	for currentID != "" {
		msg, ok := m.store[currentID]
		if !ok {
			break
		}
		chain = append([]*models.Message{msg}, chain...)
		if msg.PreviousID == "" {
			break
		}
		currentID = msg.PreviousID
	}

	return chain, nil
}

func (m *mockMessageRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	msg, ok := m.store[messageID]
	if !ok {
		return nil, errToolNotFound
	}

	// Find all messages with the same PreviousID
	var siblings []*models.Message
	for _, m := range m.store {
		if msg.PreviousID == "" && m.PreviousID == "" {
			if m.ID != messageID && m.ConversationID == msg.ConversationID {
				siblings = append(siblings, m)
			}
		} else if msg.PreviousID != "" && m.PreviousID != "" && m.PreviousID == msg.PreviousID {
			if m.ID != messageID {
				siblings = append(siblings, m)
			}
		}
	}

	return siblings, nil
}

// Tests

func TestToolService_RegisterTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}

	tool, err := svc.RegisterTool(context.Background(), "search", "Search for information", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool.ID != "tool_test1" {
		t.Errorf("expected ID tool_test1, got %s", tool.ID)
	}

	if tool.Name != "search" {
		t.Errorf("expected name 'search', got %s", tool.Name)
	}

	if !tool.Enabled {
		t.Error("expected tool to be enabled by default")
	}

	// Verify it was stored
	stored, _ := toolRepo.GetByName(context.Background(), "search")
	if stored == nil {
		t.Error("tool not stored in repository")
	}
}

func TestToolService_RegisterTool_EmptyName(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	_, err := svc.RegisterTool(context.Background(), "", "Description", nil)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestToolService_RegisterTool_Duplicate(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register first tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Try to register with same name
	_, err := svc.RegisterTool(context.Background(), "search", "Another search", nil)
	if err == nil {
		t.Fatal("expected error for duplicate tool name, got nil")
	}
}

func TestToolService_GetByName(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register a tool
	registered, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Get it back
	tool, err := svc.GetByName(context.Background(), "search")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool.ID != registered.ID {
		t.Errorf("expected ID %s, got %s", registered.ID, tool.ID)
	}
}

func TestToolService_GetByName_NotFound(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	_, err := svc.GetByName(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent tool, got nil")
	}
}

func TestToolService_Enable(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register and disable a tool
	tool, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	svc.Disable(context.Background(), tool.ID)

	// Enable it
	enabled, err := svc.Enable(context.Background(), tool.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !enabled.Enabled {
		t.Error("expected tool to be enabled")
	}

	// Verify it's updated in the store
	stored, _ := toolRepo.GetByID(context.Background(), tool.ID)
	if !stored.Enabled {
		t.Error("tool not enabled in store")
	}
}

func TestToolService_Disable(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register a tool (enabled by default)
	tool, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Disable it
	disabled, err := svc.Disable(context.Background(), tool.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if disabled.Enabled {
		t.Error("expected tool to be disabled")
	}

	// Verify it's updated in the store
	stored, _ := toolRepo.GetByID(context.Background(), tool.ID)
	if stored.Enabled {
		t.Error("tool not disabled in store")
	}
}

func TestToolService_ListEnabled(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register some tools
	tool1, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	svc.RegisterTool(context.Background(), "calculator", "Calculator tool", nil)
	tool3, _ := svc.RegisterTool(context.Background(), "weather", "Weather tool", nil)

	// Disable one
	svc.Disable(context.Background(), tool1.ID)

	// Delete one
	svc.Delete(context.Background(), tool3.ID)

	// List enabled
	enabled, err := svc.ListEnabled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled tool, got %d", len(enabled))
	}

	if enabled[0].Name != "calculator" {
		t.Errorf("expected 'calculator' to be enabled, got %s", enabled[0].Name)
	}
}

func TestToolService_RegisterExecutor(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "result", nil
	}

	err := svc.RegisterExecutor("search", executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify executor is registered (internal check)
	if _, exists := svc.executors["search"]; !exists {
		t.Error("executor not registered")
	}
}

func TestToolService_RegisterExecutor_Duplicate(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "result", nil
	}

	svc.RegisterExecutor("search", executor)

	// Try to register again - should succeed (idempotent behavior)
	err := svc.RegisterExecutor("search", executor)
	if err != nil {
		t.Fatalf("expected nil error for idempotent re-registration, got %v", err)
	}
}

func TestToolService_ExecuteTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Register executor
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		query := args["query"].(string)
		return "Results for: " + query, nil
	}
	svc.RegisterExecutor("search", executor)

	// Execute
	result, err := svc.ExecuteTool(context.Background(), "search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "Results for: test" {
		t.Errorf("expected 'Results for: test', got %v", result)
	}
}

func TestToolService_ExecuteTool_Disabled(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register and disable tool
	tool, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	svc.Disable(context.Background(), tool.ID)

	// Register executor
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "result", nil
	}
	svc.RegisterExecutor("search", executor)

	// Try to execute - should fail
	_, err := svc.ExecuteTool(context.Background(), "search", map[string]any{})
	if err == nil {
		t.Fatal("expected error when executing disabled tool, got nil")
	}
}

func TestToolService_ExecuteTool_NoExecutor(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register tool but no executor
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Try to execute - should fail
	_, err := svc.ExecuteTool(context.Background(), "search", map[string]any{})
	if err == nil {
		t.Fatal("expected error when no executor registered, got nil")
	}
}

func TestToolService_ExecuteTool_ExecutorError(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Register executor that fails
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, errors.New("execution failed")
	}
	svc.RegisterExecutor("search", executor)

	// Execute - should propagate error
	_, err := svc.ExecuteTool(context.Background(), "search", map[string]any{})
	if err == nil {
		t.Fatal("expected error from executor, got nil")
	}
}

func TestToolService_CreateToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register a tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Create tool use
	toolUse, err := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{"query": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolUse.MessageID != "msg_123" {
		t.Errorf("expected message ID msg_123, got %s", toolUse.MessageID)
	}

	if toolUse.ToolName != "search" {
		t.Errorf("expected tool name 'search', got %s", toolUse.ToolName)
	}

	if toolUse.Status != models.ToolStatusPending {
		t.Errorf("expected status pending, got %s", toolUse.Status)
	}
}

func TestToolService_CreateToolUse_DisabledTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register and disable a tool
	tool, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	svc.Disable(context.Background(), tool.ID)

	// Try to create tool use - should fail
	_, err := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})
	if err == nil {
		t.Fatal("expected error when creating tool use for disabled tool, got nil")
	}
}

func TestToolService_ExecuteToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register a tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Register executor
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "search results", nil
	}
	svc.RegisterExecutor("search", executor)

	// Create tool use
	toolUse, _ := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{"query": "test"})

	// Execute it
	executed, err := svc.ExecuteToolUse(context.Background(), toolUse.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if executed.Status != models.ToolStatusSuccess {
		t.Errorf("expected status success, got %s", executed.Status)
	}

	if executed.Result != "search results" {
		t.Errorf("expected result 'search results', got %v", executed.Result)
	}

	if executed.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestToolService_ExecuteToolUse_AlreadyComplete(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register a tool and executor
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "results", nil
	}
	svc.RegisterExecutor("search", executor)

	// Create and execute tool use
	toolUse, _ := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})
	svc.ExecuteToolUse(context.Background(), toolUse.ID)

	// Execute again - should return without re-executing
	executed, err := svc.ExecuteToolUse(context.Background(), toolUse.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if executed.Status != models.ToolStatusSuccess {
		t.Error("expected tool use to remain successful")
	}
}

func TestToolService_CancelToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register a tool
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Create tool use
	toolUse, _ := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})

	// Cancel it
	cancelled, err := svc.CancelToolUse(context.Background(), toolUse.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cancelled.Status != models.ToolStatusCancelled {
		t.Errorf("expected status cancelled, got %s", cancelled.Status)
	}

	if cancelled.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestToolService_Delete(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Register a tool
	tool, _ := svc.RegisterTool(context.Background(), "search", "Search tool", nil)

	// Delete it
	err := svc.Delete(context.Background(), tool.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's soft-deleted
	stored, _ := toolRepo.GetByID(context.Background(), tool.ID)
	if stored.DeletedAt == nil {
		t.Error("tool not soft-deleted")
	}

	// GetByID should now return error for deleted tool
	_, err = svc.GetByID(context.Background(), tool.ID)
	if err == nil {
		t.Error("expected error when getting deleted tool")
	}
}

func TestToolService_GetPendingToolUses(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	msgRepo := newMockMessageRepo()
	idGen := &mockIDGenerator{}

	svc := NewToolService(toolRepo, toolUseRepo, msgRepo, idGen)

	// Create a message
	msg := models.NewMessage("msg_123", "conv_123", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	// Register a tool and executor
	svc.RegisterTool(context.Background(), "search", "Search tool", nil)
	executor := func(ctx context.Context, args map[string]any) (any, error) {
		return "results", nil
	}
	svc.RegisterExecutor("search", executor)

	// Create multiple tool uses
	tu1, _ := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})
	svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})
	tu3, _ := svc.CreateToolUse(context.Background(), "msg_123", "search", map[string]any{})

	// Execute one
	svc.ExecuteToolUse(context.Background(), tu1.ID)

	// Cancel one
	svc.CancelToolUse(context.Background(), tu3.ID)

	// Get pending
	pending, err := svc.GetPendingToolUses(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("expected 1 pending tool use, got %d", len(pending))
	}
}
