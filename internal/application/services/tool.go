package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type ToolService struct {
	toolRepo    ports.ToolRepository
	toolUseRepo ports.ToolUseRepository
	messageRepo ports.MessageRepository
	idGenerator ports.IDGenerator
	executors   map[string]func(ctx context.Context, arguments map[string]any) (any, error)
}

func NewToolService(
	toolRepo ports.ToolRepository,
	toolUseRepo ports.ToolUseRepository,
	messageRepo ports.MessageRepository,
	idGenerator ports.IDGenerator,
) *ToolService {
	return &ToolService{
		toolRepo:    toolRepo,
		toolUseRepo: toolUseRepo,
		messageRepo: messageRepo,
		idGenerator: idGenerator,
		executors:   make(map[string]func(ctx context.Context, arguments map[string]any) (any, error)),
	}
}

func (s *ToolService) RegisterTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	if name == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	if description == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool description cannot be empty")
	}

	if schema == nil {
		schema = make(map[string]any)
	}

	// Check if tool already exists
	existing, err := s.toolRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "tool with this name already exists")
	}

	id := s.idGenerator.GenerateToolID()
	tool := models.NewTool(id, name, description, schema)

	if err := s.toolRepo.Create(ctx, tool); err != nil {
		return nil, domain.NewDomainError(err, "failed to register tool")
	}

	return tool, nil
}

// EnsureTool is an idempotent version of RegisterTool.
// If a tool with the given name already exists, it returns the existing tool.
// Otherwise, it creates a new tool with the given parameters.
func (s *ToolService) EnsureTool(ctx context.Context, name, description string, schema map[string]any) (*models.Tool, error) {
	if name == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	if description == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool description cannot be empty")
	}

	if schema == nil {
		schema = make(map[string]any)
	}

	// Check if tool already exists
	existing, err := s.toolRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		// Tool already exists, return it
		return existing, nil
	}

	// Tool doesn't exist, create it
	id := s.idGenerator.GenerateToolID()
	tool := models.NewTool(id, name, description, schema)

	if err := s.toolRepo.Create(ctx, tool); err != nil {
		return nil, domain.NewDomainError(err, "failed to register tool")
	}

	return tool, nil
}

func (s *ToolService) RegisterExecutor(name string, executor func(ctx context.Context, arguments map[string]any) (any, error)) error {
	if name == "" {
		return domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	if executor == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "executor cannot be nil")
	}

	// Allow re-registration of executors (idempotent behavior for built-in tools)
	s.executors[name] = executor
	return nil
}

func (s *ToolService) GetByID(ctx context.Context, id string) (*models.Tool, error) {
	if id == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "tool ID cannot be empty")
	}

	tool, err := s.toolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool not found")
	}

	if tool.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool has been deleted")
	}

	return tool, nil
}

func (s *ToolService) GetByName(ctx context.Context, name string) (*models.Tool, error) {
	if name == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	tool, err := s.toolRepo.GetByName(ctx, name)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool not found")
	}

	if tool.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool has been deleted")
	}

	return tool, nil
}

func (s *ToolService) Update(ctx context.Context, tool *models.Tool) error {
	if tool == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "tool cannot be nil")
	}

	if tool.ID == "" {
		return domain.NewDomainError(domain.ErrInvalidID, "tool ID cannot be empty")
	}

	// Verify tool exists
	existing, err := s.toolRepo.GetByID(ctx, tool.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrToolNotFound, "tool not found")
	}

	if existing.DeletedAt != nil {
		return domain.NewDomainError(domain.ErrToolNotFound, "cannot update deleted tool")
	}

	if err := s.toolRepo.Update(ctx, tool); err != nil {
		return domain.NewDomainError(err, "failed to update tool")
	}

	return nil
}

func (s *ToolService) Enable(ctx context.Context, id string) (*models.Tool, error) {
	tool, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tool.Enable()
	if err := s.Update(ctx, tool); err != nil {
		return nil, err
	}

	return tool, nil
}

func (s *ToolService) Disable(ctx context.Context, id string) (*models.Tool, error) {
	tool, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tool.Disable()
	if err := s.Update(ctx, tool); err != nil {
		return nil, err
	}

	return tool, nil
}

func (s *ToolService) Delete(ctx context.Context, id string) error {
	tool, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.toolRepo.Delete(ctx, tool.ID); err != nil {
		return domain.NewDomainError(err, "failed to delete tool")
	}

	return nil
}

func (s *ToolService) ListEnabled(ctx context.Context) ([]*models.Tool, error) {
	tools, err := s.toolRepo.ListEnabled(ctx)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list enabled tools")
	}

	return tools, nil
}

func (s *ToolService) ListAll(ctx context.Context) ([]*models.Tool, error) {
	tools, err := s.toolRepo.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list tools")
	}

	return tools, nil
}

func (s *ToolService) CreateToolUse(ctx context.Context, messageID, toolName string, arguments map[string]any) (*models.ToolUse, error) {
	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	if toolName == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	// Verify message exists
	if _, err := s.messageRepo.GetByID(ctx, messageID); err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "message not found")
	}

	// Verify tool exists and is enabled
	tool, err := s.GetByName(ctx, toolName)
	if err != nil {
		return nil, err
	}

	if !tool.Enabled {
		return nil, domain.NewDomainError(domain.ErrToolDisabled, "tool is disabled")
	}

	// Validate arguments against schema
	if err := s.validateArguments(arguments, tool.Schema); err != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToolArgs, err.Error())
	}

	// Get next sequence number for the message
	existingToolUses, err := s.toolUseRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get existing tool uses")
	}
	sequenceNumber := len(existingToolUses)

	id := s.idGenerator.GenerateToolUseID()
	toolUse := models.NewToolUse(id, messageID, toolName, sequenceNumber, arguments)

	if err := s.toolUseRepo.Create(ctx, toolUse); err != nil {
		return nil, domain.NewDomainError(err, "failed to create tool use")
	}

	return toolUse, nil
}

func (s *ToolService) ExecuteToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	if toolUseID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "tool use ID cannot be empty")
	}

	toolUse, err := s.toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool use not found")
	}

	if toolUse.IsComplete() {
		return toolUse, nil // Already executed
	}

	// Get executor for the tool
	executor, exists := s.executors[toolUse.ToolName]
	if !exists {
		toolUse.Fail("no executor registered for tool")
		if err := s.toolUseRepo.Update(ctx, toolUse); err != nil {
			log.Printf("ERROR: failed to update tool use %s after executor not found: %v", toolUseID, err)
		}
		return nil, domain.NewDomainError(domain.ErrToolExecutionFailed, "no executor registered for tool")
	}

	// Mark as running
	toolUse.Start()
	if err := s.toolUseRepo.Update(ctx, toolUse); err != nil {
		return nil, domain.NewDomainError(err, "failed to update tool use status")
	}

	// Execute the tool
	result, err := executor(ctx, toolUse.Arguments)
	if err != nil {
		toolUse.Fail(err.Error())
		if updateErr := s.toolUseRepo.Update(ctx, toolUse); updateErr != nil {
			log.Printf("ERROR: failed to update tool use %s after execution error: %v", toolUseID, updateErr)
		}
		return nil, domain.NewDomainError(domain.ErrToolExecutionFailed, err.Error())
	}

	// Mark as complete
	toolUse.Complete(result)
	if err := s.toolUseRepo.Update(ctx, toolUse); err != nil {
		return nil, domain.NewDomainError(err, "failed to update tool use result")
	}

	return toolUse, nil
}

func (s *ToolService) ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
	if toolName == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "tool name cannot be empty")
	}

	// Verify tool exists and is enabled
	tool, err := s.GetByName(ctx, toolName)
	if err != nil {
		return nil, err
	}

	if !tool.Enabled {
		return nil, domain.NewDomainError(domain.ErrToolDisabled, "tool is disabled")
	}

	// Get executor for the tool
	executor, exists := s.executors[toolName]
	if !exists {
		return nil, domain.NewDomainError(domain.ErrToolExecutionFailed, "no executor registered for tool")
	}

	// Validate arguments
	if err := s.validateArguments(arguments, tool.Schema); err != nil {
		return nil, domain.NewDomainError(domain.ErrInvalidToolArgs, err.Error())
	}

	// Execute the tool
	result, err := executor(ctx, arguments)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolExecutionFailed, err.Error())
	}

	return result, nil
}

func (s *ToolService) GetToolUseByID(ctx context.Context, id string) (*models.ToolUse, error) {
	if id == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "tool use ID cannot be empty")
	}

	toolUse, err := s.toolUseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool use not found")
	}

	return toolUse, nil
}

func (s *ToolService) GetToolUsesByMessage(ctx context.Context, messageID string) ([]*models.ToolUse, error) {
	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	toolUses, err := s.toolUseRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get tool uses")
	}

	return toolUses, nil
}

func (s *ToolService) GetPendingToolUses(ctx context.Context, limit int) ([]*models.ToolUse, error) {
	if limit <= 0 {
		limit = 10
	}

	toolUses, err := s.toolUseRepo.GetPending(ctx, limit)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get pending tool uses")
	}

	return toolUses, nil
}

func (s *ToolService) CancelToolUse(ctx context.Context, toolUseID string) (*models.ToolUse, error) {
	if toolUseID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "tool use ID cannot be empty")
	}

	toolUse, err := s.toolUseRepo.GetByID(ctx, toolUseID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrToolNotFound, "tool use not found")
	}

	if toolUse.IsComplete() {
		return toolUse, nil // Already complete, can't cancel
	}

	toolUse.Cancel()
	if err := s.toolUseRepo.Update(ctx, toolUse); err != nil {
		return nil, domain.NewDomainError(err, "failed to cancel tool use")
	}

	return toolUse, nil
}

func (s *ToolService) validateArguments(arguments map[string]any, schema map[string]any) error {
	// Basic validation - this can be extended with JSON Schema validation
	if schema == nil {
		return nil // No schema to validate against
	}

	// Check for required fields
	if required, ok := schema["required"].([]any); ok {
		for _, reqField := range required {
			if fieldName, ok := reqField.(string); ok {
				if _, exists := arguments[fieldName]; !exists {
					return fmt.Errorf("missing required field: %s", fieldName)
				}
			}
		}
	}

	// Validate field types if properties are defined
	if properties, ok := schema["properties"].(map[string]any); ok {
		for argName, argValue := range arguments {
			if propSchema, exists := properties[argName]; exists {
				if propMap, ok := propSchema.(map[string]any); ok {
					if expectedType, ok := propMap["type"].(string); ok {
						if err := s.validateType(argValue, expectedType); err != nil {
							return fmt.Errorf("invalid type for field %s: %v", argName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *ToolService) validateType(value any, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32, json.Number:
			return nil
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "integer":
		switch value.(type) {
		case int, int64, int32, float64:
			return nil
		default:
			return fmt.Errorf("expected integer, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		switch value.(type) {
		case []any:
			return nil
		default:
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	return nil
}
