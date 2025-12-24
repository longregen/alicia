package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type ToolExecutor interface {
	Execute(ctx context.Context, tool *models.Tool, arguments map[string]any) (any, error)
}

type ExecuteToolInput struct {
	ToolUseID      string
	ToolName       string
	Arguments      map[string]any
	TimeoutMs      int
	MessageID      string
	ConversationID string
}

type ExecuteToolOutput struct {
	ToolUse *models.ToolUse
	Result  any
	Success bool
}

type HandleToolCall struct {
	toolRepo     ports.ToolRepository
	toolUseRepo  ports.ToolUseRepository
	toolExecutor ToolExecutor
	idGenerator  ports.IDGenerator
}

func NewHandleToolCall(
	toolRepo ports.ToolRepository,
	toolUseRepo ports.ToolUseRepository,
	toolExecutor ToolExecutor,
	idGenerator ports.IDGenerator,
) *HandleToolCall {
	return &HandleToolCall{
		toolRepo:     toolRepo,
		toolUseRepo:  toolUseRepo,
		toolExecutor: toolExecutor,
		idGenerator:  idGenerator,
	}
}

// Execute implements ports.HandleToolUseCase
func (uc *HandleToolCall) Execute(ctx context.Context, input *ports.HandleToolInput) (*ports.HandleToolOutput, error) {
	// Convert from ports input to internal input
	internalInput := &ExecuteToolInput{
		ToolUseID:      input.ToolUseID,
		ToolName:       input.ToolName,
		Arguments:      input.Arguments,
		TimeoutMs:      input.TimeoutMs,
		MessageID:      input.MessageID,
		ConversationID: input.ConversationID,
	}

	return uc.execute(ctx, internalInput)
}

func (uc *HandleToolCall) execute(ctx context.Context, input *ExecuteToolInput) (*ports.HandleToolOutput, error) {
	tool, err := uc.toolRepo.GetByName(ctx, input.ToolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool: %w", err)
	}

	if !tool.Enabled {
		return nil, fmt.Errorf("tool '%s' is not enabled", input.ToolName)
	}

	var toolUse *models.ToolUse
	if input.ToolUseID != "" {
		toolUse, err = uc.toolUseRepo.GetByID(ctx, input.ToolUseID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tool use: %w", err)
		}
	} else {
		toolUseID := uc.idGenerator.GenerateToolUseID()
		sequenceNumber := 0 // This should ideally be fetched from the repository
		toolUse = models.NewToolUse(toolUseID, input.MessageID, input.ToolName, sequenceNumber, input.Arguments)

		if err := uc.toolUseRepo.Create(ctx, toolUse); err != nil {
			return nil, fmt.Errorf("failed to create tool use: %w", err)
		}
	}

	toolUse.Start()
	if err := uc.toolUseRepo.Update(ctx, toolUse); err != nil {
		return nil, fmt.Errorf("failed to update tool use status: %w", err)
	}

	timeout := time.Duration(input.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := uc.executeWithTimeout(execCtx, tool, toolUse, input.Arguments)

	if err != nil {
		toolUse.Fail(err.Error())
		if updateErr := uc.toolUseRepo.Update(ctx, toolUse); updateErr != nil {
			return nil, fmt.Errorf("failed to update tool use after error: %w", updateErr)
		}

		return &ports.HandleToolOutput{
			ToolUseID: toolUse.ID,
			Result:    nil,
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	toolUse.Complete(result)
	if err := uc.toolUseRepo.Update(ctx, toolUse); err != nil {
		return nil, fmt.Errorf("failed to update tool use after completion: %w", err)
	}

	return &ports.HandleToolOutput{
		ToolUseID: toolUse.ID,
		Result:    result,
		Success:   true,
		Error:     "",
	}, nil
}

func (uc *HandleToolCall) executeWithTimeout(ctx context.Context, tool *models.Tool, toolUse *models.ToolUse, arguments map[string]any) (any, error) {
	type executionResult struct {
		result any
		err    error
	}

	resultChan := make(chan executionResult, 1)

	go func() {
		result, err := uc.toolExecutor.Execute(ctx, tool, arguments)
		resultChan <- executionResult{result: result, err: err}
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("tool execution timed out")
		}
		return nil, fmt.Errorf("tool execution cancelled: %w", ctx.Err())
	case result := <-resultChan:
		return result.result, result.err
	}
}

func (uc *HandleToolCall) GetToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	toolUse, err := uc.toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool use: %w", err)
	}

	return toolUse, nil
}

func (uc *HandleToolCall) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	toolUses, err := uc.toolUseRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool uses for message: %w", err)
	}

	return toolUses, nil
}

func (uc *HandleToolCall) CancelToolUse(ctx context.Context, toolUseID string) error {
	toolUse, err := uc.toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		return fmt.Errorf("failed to get tool use: %w", err)
	}

	if !toolUse.IsRunning() {
		return fmt.Errorf("tool use is not in running state (status: %s)", toolUse.Status)
	}

	toolUse.Cancel()
	if err := uc.toolUseRepo.Update(ctx, toolUse); err != nil {
		return fmt.Errorf("failed to cancel tool use: %w", err)
	}

	return nil
}

type RegisterToolInput struct {
	Name        string
	Description string
	Schema      map[string]any
	Enabled     bool
}

func (uc *HandleToolCall) RegisterTool(ctx context.Context, input *RegisterToolInput) (*models.Tool, error) {
	existingTool, err := uc.toolRepo.GetByName(ctx, input.Name)
	if err == nil && existingTool != nil {
		return nil, fmt.Errorf("tool with name '%s' already exists", input.Name)
	}

	toolID := uc.idGenerator.GenerateToolID()
	tool := models.NewTool(toolID, input.Name, input.Description, input.Schema)
	if !input.Enabled {
		tool.Disable()
	}

	if err := uc.toolRepo.Create(ctx, tool); err != nil {
		return nil, fmt.Errorf("failed to create tool: %w", err)
	}

	return tool, nil
}

func (uc *HandleToolCall) UpdateToolStatus(ctx context.Context, toolID string, enabled bool) error {
	tool, err := uc.toolRepo.GetByID(ctx, toolID)
	if err != nil {
		return fmt.Errorf("failed to get tool: %w", err)
	}

	if enabled {
		tool.Enable()
	} else {
		tool.Disable()
	}

	if err := uc.toolRepo.Update(ctx, tool); err != nil {
		return fmt.Errorf("failed to update tool status: %w", err)
	}

	return nil
}

func (uc *HandleToolCall) ListTools(ctx context.Context, enabledOnly bool) ([]*models.Tool, error) {
	var tools []*models.Tool
	var err error

	if enabledOnly {
		tools, err = uc.toolRepo.ListEnabled(ctx)
	} else {
		tools, err = uc.toolRepo.ListAll(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return tools, nil
}
