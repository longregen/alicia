package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

// ToolExecutor executes a tool with given arguments.
type ToolExecutor func(ctx context.Context, args map[string]any) (any, error)

// ToolService handles tool registration and execution.
type ToolService struct {
	store     *store.Store
	executors map[string]ToolExecutor
	mu        sync.RWMutex
}

// NewToolService creates a new tool service.
func NewToolService(s *store.Store) *ToolService {
	return &ToolService{
		store:     s,
		executors: make(map[string]ToolExecutor),
	}
}

// RegisterTool registers a tool in the database.
func (svc *ToolService) RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*domain.Tool, error) {
	// Check if tool already exists
	existing, err := svc.store.GetToolByName(ctx, name)
	if err == nil {
		// Update existing tool
		existing.Description = description
		existing.Schema = schema
		if err := svc.store.UpdateTool(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	tool := &domain.Tool{
		ID:          store.NewToolID(),
		Name:        name,
		Description: description,
		Schema:      schema,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := svc.store.CreateTool(ctx, tool); err != nil {
		return nil, err
	}
	return tool, nil
}

// RegisterExecutor registers an executor function for a tool.
func (svc *ToolService) RegisterExecutor(name string, executor ToolExecutor) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.executors[name] = executor
}

// UnregisterExecutor removes an executor.
func (svc *ToolService) UnregisterExecutor(name string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.executors, name)
}

// HasExecutor checks if a tool has a registered executor.
func (svc *ToolService) HasExecutor(name string) bool {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	_, ok := svc.executors[name]
	return ok
}

// GetTool retrieves a tool by ID.
func (svc *ToolService) GetTool(ctx context.Context, id string) (*domain.Tool, error) {
	return svc.store.GetTool(ctx, id)
}

// GetToolByName retrieves a tool by name.
func (svc *ToolService) GetToolByName(ctx context.Context, name string) (*domain.Tool, error) {
	return svc.store.GetToolByName(ctx, name)
}

// ListTools returns all tools.
func (svc *ToolService) ListTools(ctx context.Context) ([]*domain.Tool, error) {
	return svc.store.ListTools(ctx)
}

// ListEnabledTools returns enabled tools.
func (svc *ToolService) ListEnabledTools(ctx context.Context) ([]*domain.Tool, error) {
	return svc.store.ListEnabledTools(ctx)
}

// ListAvailableTools returns enabled tools with registered executors.
func (svc *ToolService) ListAvailableTools(ctx context.Context) ([]*domain.Tool, error) {
	tools, err := svc.store.ListEnabledTools(ctx)
	if err != nil {
		return nil, err
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	available := make([]*domain.Tool, 0, len(tools))
	for _, t := range tools {
		if _, ok := svc.executors[t.Name]; ok {
			available = append(available, t)
		}
	}
	return available, nil
}

// EnableTool enables a tool.
func (svc *ToolService) EnableTool(ctx context.Context, id string) error {
	tool, err := svc.store.GetTool(ctx, id)
	if err != nil {
		return err
	}
	tool.Enabled = true
	return svc.store.UpdateTool(ctx, tool)
}

// DisableTool disables a tool.
func (svc *ToolService) DisableTool(ctx context.Context, id string) error {
	tool, err := svc.store.GetTool(ctx, id)
	if err != nil {
		return err
	}
	tool.Enabled = false
	return svc.store.UpdateTool(ctx, tool)
}

// ExecuteTool executes a tool and records the usage.
func (svc *ToolService) ExecuteTool(ctx context.Context, messageID, toolName string, args map[string]any) (*domain.ToolUse, error) {
	svc.mu.RLock()
	executor, ok := svc.executors[toolName]
	svc.mu.RUnlock()

	tu := &domain.ToolUse{
		ID:        store.NewToolUseID(),
		MessageID: messageID,
		ToolName:  toolName,
		Arguments: args,
		Status:    domain.ToolUseStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	if err := svc.store.CreateToolUse(ctx, tu); err != nil {
		return nil, err
	}

	if !ok {
		tu.Status = domain.ToolUseStatusError
		tu.Error = fmt.Sprintf("no executor registered for tool: %s", toolName)
		_ = svc.store.UpdateToolUse(ctx, tu)
		return tu, fmt.Errorf("%s", tu.Error)
	}

	result, err := executor(ctx, args)
	if err != nil {
		tu.Status = domain.ToolUseStatusError
		tu.Error = err.Error()
	} else {
		tu.Status = domain.ToolUseStatusSuccess
		tu.Result = result
	}

	if updateErr := svc.store.UpdateToolUse(ctx, tu); updateErr != nil {
		return tu, updateErr
	}

	return tu, err
}

// GetToolUse retrieves a tool use by ID.
func (svc *ToolService) GetToolUse(ctx context.Context, id string) (*domain.ToolUse, error) {
	return svc.store.GetToolUse(ctx, id)
}

// GetToolUsesByMessage returns tool uses for a message.
func (svc *ToolService) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*domain.ToolUse, error) {
	return svc.store.GetToolUsesByMessage(ctx, messageID)
}

// ListToolUses returns all tool uses with pagination and total count.
func (svc *ToolService) ListToolUses(ctx context.Context, limit, offset int) ([]*domain.ToolUse, int, error) {
	return svc.store.ListToolUses(ctx, limit, offset)
}
