//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/adapters/id"
	"github.com/longregen/alicia/internal/adapters/postgres"
	"github.com/longregen/alicia/internal/application/services"
	"github.com/longregen/alicia/internal/domain/models"
)

func TestToolFlow_RegisterAndExecute(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	idGen := id.NewGenerator()
	mockExecutor := &mockToolExecutor{}

	toolSvc := services.NewToolService(toolRepo, toolUseRepo, mockExecutor, idGen)

	// Test: Register a tool
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
		"required": []string{"query"},
	}

	tool, err := toolSvc.RegisterTool(ctx, &services.RegisterToolInput{
		Name:        "web_search",
		Description: "Search the web for information",
		Schema:      schema,
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	if tool.ID == "" {
		t.Fatal("tool ID should not be empty")
	}
	if tool.Name != "web_search" {
		t.Errorf("expected name 'web_search', got '%s'", tool.Name)
	}
	if !tool.Enabled {
		t.Error("tool should be enabled by default")
	}

	// Test: Retrieve the tool
	retrieved, err := toolSvc.GetToolByName(ctx, "web_search")
	if err != nil {
		t.Fatalf("failed to retrieve tool: %v", err)
	}

	if retrieved.ID != tool.ID {
		t.Errorf("expected ID %s, got %s", tool.ID, retrieved.ID)
	}

	// Test: List tools
	tools, err := toolSvc.ListTools(ctx)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	// Test: Disable tool
	disabledTool, err := toolSvc.DisableTool(ctx, tool.ID)
	if err != nil {
		t.Fatalf("failed to disable tool: %v", err)
	}

	if disabledTool.Enabled {
		t.Error("tool should be disabled")
	}

	// Test: Enable tool
	enabledTool, err := toolSvc.EnableTool(ctx, tool.ID)
	if err != nil {
		t.Fatalf("failed to enable tool: %v", err)
	}

	if !enabledTool.Enabled {
		t.Error("tool should be enabled")
	}
}

func TestToolFlow_ToolUseLifecycle(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	conversationRepo := postgres.NewConversationRepository(db.Pool)
	messageRepo := postgres.NewMessageRepository(db.Pool)
	idGen := id.NewGenerator()

	// Create test data
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Test Conversation")
	message := fixtures.CreateMessage(ctx, t, "msg1", conversation.ID, models.MessageRoleAssistant, "Let me search for that", 1)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}
	tool := fixtures.CreateTool(ctx, t, "tool1", "search", "Search tool", schema)

	// Test: Create a tool use (pending)
	toolUse := models.NewToolUse(
		idGen.GenerateToolUseID(),
		message.ID,
		tool.Name,
		1,
		map[string]any{"query": "Go programming"},
	)

	err := toolUseRepo.Create(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to create tool use: %v", err)
	}

	if toolUse.Status != models.ToolStatusPending {
		t.Errorf("expected status 'pending', got '%s'", toolUse.Status)
	}

	// Test: Start tool execution
	toolUse.Start()
	err = toolUseRepo.Update(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to update tool use: %v", err)
	}

	retrieved, err := toolUseRepo.GetByID(ctx, toolUse.ID)
	if err != nil {
		t.Fatalf("failed to retrieve tool use: %v", err)
	}

	if retrieved.Status != models.ToolStatusRunning {
		t.Errorf("expected status 'running', got '%s'", retrieved.Status)
	}

	// Test: Complete tool execution with result
	result := map[string]any{
		"results": []string{"Result 1", "Result 2"},
		"count":   2,
	}
	toolUse.Complete(result)
	err = toolUseRepo.Update(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to complete tool use: %v", err)
	}

	completed, err := toolUseRepo.GetByID(ctx, toolUse.ID)
	if err != nil {
		t.Fatalf("failed to retrieve completed tool use: %v", err)
	}

	if completed.Status != models.ToolStatusSuccess {
		t.Errorf("expected status 'success', got '%s'", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
	if completed.Result == nil {
		t.Error("result should be set")
	}
}

func TestToolFlow_ToolUseError(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	idGen := id.NewGenerator()

	// Create test data
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Test Conversation")
	message := fixtures.CreateMessage(ctx, t, "msg1", conversation.ID, models.MessageRoleAssistant, "Let me try that", 1)

	schema := map[string]any{"type": "object"}
	tool := fixtures.CreateTool(ctx, t, "tool1", "failing_tool", "Tool that fails", schema)

	// Create a tool use
	toolUse := models.NewToolUse(
		idGen.GenerateToolUseID(),
		message.ID,
		tool.Name,
		1,
		map[string]any{"param": "value"},
	)

	err := toolUseRepo.Create(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to create tool use: %v", err)
	}

	// Start and fail the tool execution
	toolUse.Start()
	toolUse.Fail("Connection timeout")

	err = toolUseRepo.Update(ctx, toolUse)
	if err != nil {
		t.Fatalf("failed to update tool use: %v", err)
	}

	failed, err := toolUseRepo.GetByID(ctx, toolUse.ID)
	if err != nil {
		t.Fatalf("failed to retrieve failed tool use: %v", err)
	}

	if failed.Status != models.ToolStatusError {
		t.Errorf("expected status 'error', got '%s'", failed.Status)
	}
	if failed.ErrorMessage != "Connection timeout" {
		t.Errorf("expected error message 'Connection timeout', got '%s'", failed.ErrorMessage)
	}
	if failed.CompletedAt == nil {
		t.Error("completed_at should be set even for failed executions")
	}
}

func TestToolFlow_MultipleToolUses(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()
	fixtures := NewFixtures(db)

	toolUseRepo := postgres.NewToolUseRepository(db.Pool)

	// Create test data
	conversation := fixtures.CreateConversation(ctx, t, "conv1", "Test Conversation")
	message := fixtures.CreateMessage(ctx, t, "msg1", conversation.ID, models.MessageRoleAssistant, "Using multiple tools", 1)

	schema := map[string]any{"type": "object"}
	tool1 := fixtures.CreateTool(ctx, t, "tool1", "search", "Search tool", schema)
	tool2 := fixtures.CreateTool(ctx, t, "tool2", "calculator", "Calculator tool", schema)

	// Create multiple tool uses in sequence
	toolUse1 := fixtures.CreateToolUse(ctx, t, "tu1", message.ID, tool1.Name, map[string]any{"query": "test"}, 1)
	toolUse2 := fixtures.CreateToolUse(ctx, t, "tu2", message.ID, tool2.Name, map[string]any{"expr": "1+1"}, 2)
	toolUse3 := fixtures.CreateToolUse(ctx, t, "tu3", message.ID, tool1.Name, map[string]any{"query": "more"}, 3)

	// List tool uses for message
	toolUses, err := toolUseRepo.ListByMessage(ctx, message.ID, 100, 0)
	if err != nil {
		t.Fatalf("failed to list tool uses: %v", err)
	}

	if len(toolUses) != 3 {
		t.Errorf("expected 3 tool uses, got %d", len(toolUses))
	}

	// Verify sequence order
	if toolUses[0].SequenceNumber != 1 || toolUses[1].SequenceNumber != 2 || toolUses[2].SequenceNumber != 3 {
		t.Error("tool uses not in correct sequence order")
	}

	// List by tool name
	searchToolUses, err := toolUseRepo.ListByToolName(ctx, tool1.Name, 100, 0)
	if err != nil {
		t.Fatalf("failed to list tool uses by tool name: %v", err)
	}

	if len(searchToolUses) != 2 {
		t.Errorf("expected 2 tool uses for search tool, got %d", len(searchToolUses))
	}
}

func TestToolFlow_UnregisterTool(t *testing.T) {
	db := SetupTestDB(t)
	ctx := context.Background()

	toolRepo := postgres.NewToolRepository(db.Pool)
	toolUseRepo := postgres.NewToolUseRepository(db.Pool)
	idGen := id.NewGenerator()
	mockExecutor := &mockToolExecutor{}

	toolSvc := services.NewToolService(toolRepo, toolUseRepo, mockExecutor, idGen)

	// Register a tool
	schema := map[string]any{"type": "object"}
	tool, err := toolSvc.RegisterTool(ctx, &services.RegisterToolInput{
		Name:        "test_tool",
		Description: "Test tool",
		Schema:      schema,
	})
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// Unregister the tool
	err = toolSvc.UnregisterTool(ctx, tool.ID)
	if err != nil {
		t.Fatalf("failed to unregister tool: %v", err)
	}

	// Verify tool is deleted
	_, err = toolSvc.GetToolByName(ctx, "test_tool")
	if err == nil {
		t.Error("expected error when retrieving unregistered tool")
	}
}

// mockToolExecutor simulates tool execution
type mockToolExecutor struct{}

func (m *mockToolExecutor) Execute(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
	return map[string]any{"result": "success"}, nil
}
