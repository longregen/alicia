package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type mockToolExecutor struct {
	executeFunc func(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error)
}

func (m *mockToolExecutor) Execute(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, tool, arguments)
	}
	return "default result", nil
}

func TestHandleToolCall_ExecuteNewTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()

	tool := models.NewTool("tool_1", "calculator", "Calculate expressions", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	executor := &mockToolExecutor{
		executeFunc: func(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
			return 42, nil
		},
	}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolName:       "calculator",
		Arguments:      map[string]any{"expression": "6*7"},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
		TimeoutMs:      5000,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.Success {
		t.Errorf("expected success, got failure: %s", output.Error)
	}

	if output.Result != 42 {
		t.Errorf("expected result 42, got %v", output.Result)
	}

	toolUse, _ := toolUseRepo.GetByID(context.Background(), output.ToolUseID)
	if toolUse == nil {
		t.Fatal("tool use not stored")
	}

	if toolUse.Status != models.ToolStatusSuccess {
		t.Errorf("expected status completed, got %s", toolUse.Status)
	}
}

func TestHandleToolCall_ExecuteExistingToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()

	tool := models.NewTool("tool_1", "test_tool", "Test tool", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	existingToolUse := models.NewToolUse("tu_existing", "msg_123", "test_tool", 0, map[string]any{})
	toolUseRepo.Create(context.Background(), existingToolUse)

	executor := &mockToolExecutor{
		executeFunc: func(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
			return "executed result", nil
		},
	}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolUseID:      "tu_existing",
		ToolName:       "test_tool",
		Arguments:      map[string]any{},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.Success {
		t.Error("expected success")
	}

	if output.ToolUseID != "tu_existing" {
		t.Errorf("expected tool use ID tu_existing, got %s", output.ToolUseID)
	}
}

func TestHandleToolCall_ToolNotFound(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolName:       "nonexistent_tool",
		Arguments:      map[string]any{},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for nonexistent tool, got nil")
	}
}

func TestHandleToolCall_ToolDisabled(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	tool := models.NewTool("tool_1", "disabled_tool", "Disabled tool", map[string]any{})
	tool.Disable()
	toolRepo.Create(context.Background(), tool)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolName:       "disabled_tool",
		Arguments:      map[string]any{},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for disabled tool, got nil")
	}
}

func TestHandleToolCall_ExecutionFailure(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()

	tool := models.NewTool("tool_1", "failing_tool", "Tool that fails", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	executor := &mockToolExecutor{
		executeFunc: func(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
			return nil, errors.New("execution failed")
		},
	}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolName:       "failing_tool",
		Arguments:      map[string]any{},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
	}

	output, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error from execution failure")
	}

	if output.Success {
		t.Error("expected success to be false")
	}

	if output.Error == "" {
		t.Error("expected error message to be set")
	}

	toolUse, _ := toolUseRepo.GetByID(context.Background(), output.ToolUseID)
	if toolUse == nil {
		t.Fatal("tool use not stored")
	}

	if toolUse.Status != models.ToolStatusError {
		t.Errorf("expected status failed, got %s", toolUse.Status)
	}
}

func TestHandleToolCall_ExecutionTimeout(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()

	tool := models.NewTool("tool_1", "slow_tool", "Tool that times out", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	executor := &mockToolExecutor{
		executeFunc: func(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error) {
			select {
			case <-time.After(2 * time.Second):
				return "result", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &ports.HandleToolInput{
		ToolName:       "slow_tool",
		Arguments:      map[string]any{},
		MessageID:      "msg_123",
		ConversationID: "conv_123",
		TimeoutMs:      100,
	}

	output, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if output.Success {
		t.Error("expected success to be false on timeout")
	}

	toolUse, _ := toolUseRepo.GetByID(context.Background(), output.ToolUseID)
	if toolUse.Status != models.ToolStatusError {
		t.Errorf("expected status failed, got %s", toolUse.Status)
	}
}

func TestHandleToolCall_RegisterTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &RegisterToolInput{
		Name:        "new_tool",
		Description: "A newly registered tool",
		Schema:      map[string]any{"type": "object"},
		Enabled:     true,
	}

	tool, err := uc.RegisterTool(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tool.Name != "new_tool" {
		t.Errorf("expected name 'new_tool', got %s", tool.Name)
	}

	if !tool.Enabled {
		t.Error("expected tool to be enabled")
	}

	stored, _ := toolRepo.GetByName(context.Background(), "new_tool")
	if stored == nil {
		t.Error("tool not stored in repository")
	}
}

func TestHandleToolCall_RegisterDuplicateTool(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	existing := models.NewTool("tool_1", "existing_tool", "Existing tool", map[string]any{})
	toolRepo.Create(context.Background(), existing)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	input := &RegisterToolInput{
		Name:        "existing_tool",
		Description: "Duplicate tool",
		Schema:      map[string]any{},
		Enabled:     true,
	}

	_, err := uc.RegisterTool(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for duplicate tool registration, got nil")
	}
}

func TestHandleToolCall_UpdateToolStatus(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	tool := models.NewTool("tool_1", "test_tool", "Test tool", map[string]any{})
	tool.Enable()
	toolRepo.Create(context.Background(), tool)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	err := uc.UpdateToolStatus(context.Background(), "tool_1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := toolRepo.GetByID(context.Background(), "tool_1")
	if updated.Enabled {
		t.Error("expected tool to be disabled")
	}

	err = uc.UpdateToolStatus(context.Background(), "tool_1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ = toolRepo.GetByID(context.Background(), "tool_1")
	if !updated.Enabled {
		t.Error("expected tool to be enabled")
	}
}

func TestHandleToolCall_ListTools(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	tool1 := models.NewTool("tool_1", "enabled_tool", "Enabled tool", map[string]any{})
	tool1.Enable()
	toolRepo.Create(context.Background(), tool1)

	tool2 := models.NewTool("tool_2", "disabled_tool", "Disabled tool", map[string]any{})
	tool2.Disable()
	toolRepo.Create(context.Background(), tool2)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	enabledTools, err := uc.ListTools(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enabledTools) != 1 {
		t.Errorf("expected 1 enabled tool, got %d", len(enabledTools))
	}

	allTools, err := uc.ListTools(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(allTools) != 2 {
		t.Errorf("expected 2 total tools, got %d", len(allTools))
	}
}

func TestHandleToolCall_CancelToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	toolUse := models.NewToolUse("tu_1", "msg_123", "test_tool", 0, map[string]any{})
	toolUse.Start()
	toolUseRepo.Create(context.Background(), toolUse)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	err := uc.CancelToolUse(context.Background(), "tu_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancelled, _ := toolUseRepo.GetByID(context.Background(), "tu_1")
	if cancelled.Status != models.ToolStatusCancelled {
		t.Errorf("expected status cancelled, got %s", cancelled.Status)
	}
}

func TestHandleToolCall_CancelNonRunningToolUse(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	toolUse := models.NewToolUse("tu_1", "msg_123", "test_tool", 0, map[string]any{})
	toolUseRepo.Create(context.Background(), toolUse)

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	err := uc.CancelToolUse(context.Background(), "tu_1")
	if err == nil {
		t.Fatal("expected error when cancelling non-running tool use, got nil")
	}
}

func TestHandleToolCall_GetToolUsesByMessage(t *testing.T) {
	toolRepo := newMockToolRepo()
	toolUseRepo := newMockToolUseRepo()
	idGen := newMockIDGenerator()
	executor := &mockToolExecutor{}

	uc := NewHandleToolCall(toolRepo, toolUseRepo, executor, idGen)

	toolUses, err := uc.GetToolUsesByMessage(context.Background(), "msg_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if toolUses == nil {
		t.Error("expected non-nil tool uses list")
	}
}
