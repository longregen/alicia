package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// ParetoContinueResponse continues an existing assistant response using the Pareto search approach.
// This wraps the ParetoResponseGenerator for the continue use case.
type ParetoContinueResponse struct {
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	paretoGenerator  *ParetoResponseGenerator
	idGenerator      ports.IDGenerator
	txManager        ports.TransactionManager
}

// NewParetoContinueResponse creates a new ParetoContinueResponse use case.
func NewParetoContinueResponse(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	paretoGenerator *ParetoResponseGenerator,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *ParetoContinueResponse {
	return &ParetoContinueResponse{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		paretoGenerator:  paretoGenerator,
		idGenerator:      idGenerator,
		txManager:        txManager,
	}
}

// Execute continues an existing assistant message using the Pareto search approach.
// It generates additional content to append to the existing message.
func (c *ParetoContinueResponse) Execute(ctx context.Context, input *ports.ContinueResponseInput) (*ports.ContinueResponseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.TargetMessageID == "" {
		return nil, fmt.Errorf("target message ID is required")
	}

	// Get the assistant message to continue
	targetMessage, err := c.messageRepo.GetByID(ctx, input.TargetMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target message: %w", err)
	}

	if targetMessage.Role != models.MessageRoleAssistant {
		return nil, fmt.Errorf("can only continue assistant messages")
	}

	// Get the conversation
	conversation, err := c.conversationRepo.GetByID(ctx, targetMessage.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Find the original user message that triggered this response
	var userMessage *models.Message
	if targetMessage.PreviousID != "" {
		userMessage, err = c.messageRepo.GetByID(ctx, targetMessage.PreviousID)
		if err != nil {
			return nil, fmt.Errorf("failed to get original user message: %w", err)
		}
	} else {
		// Fall back to finding by sequence
		messages, err := c.messageRepo.GetByConversation(ctx, targetMessage.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation messages: %w", err)
		}

		for i, msg := range messages {
			if msg.ID == targetMessage.ID && i > 0 {
				userMessage = messages[i-1]
				break
			}
		}
	}

	if userMessage == nil {
		return nil, fmt.Errorf("could not find original user message")
	}

	// Create a continuation prompt that includes the existing content
	existingContent := targetMessage.Contents
	continuationSeed := fmt.Sprintf(`You are continuing an existing response. Here is what you've already written:

---BEGIN EXISTING RESPONSE---
%s
---END EXISTING RESPONSE---

Continue from where you left off. Do NOT repeat what was already written.
Start your continuation immediately after the existing content.
Make sure the continuation flows naturally from what came before.`, existingContent)

	// Generate continuation using Pareto search with a custom seed strategy
	paretoInput := &ParetoResponseInput{
		ConversationID:  conversation.ID,
		UserMessageID:   userMessage.ID,
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		Notifier:        input.Notifier,
		SeedStrategy:    continuationSeed,
	}

	paretoOutput, err := c.paretoGenerator.Execute(ctx, paretoInput)
	if err != nil {
		return nil, fmt.Errorf("pareto response generation failed: %w", err)
	}

	// Append the new content to the target message
	appendedContent := ""
	if paretoOutput.Message != nil {
		appendedContent = paretoOutput.Message.Contents

		// Update the original target message with the appended content
		targetMessage.Contents = existingContent + "\n\n" + appendedContent
		if err := c.messageRepo.Update(ctx, targetMessage); err != nil {
			return nil, fmt.Errorf("failed to update target message: %w", err)
		}

		// Delete the temporary message created by Pareto generator
		paretoOutput.Message.SoftDelete()
		_ = c.messageRepo.Update(ctx, paretoOutput.Message)
	}

	return &ports.ContinueResponseOutput{
		TargetMessage:   targetMessage,
		AppendedContent: appendedContent,
		StreamChannel:   paretoOutput.StreamChannel,
	}, nil
}

// Ensure ParetoContinueResponse implements ports.ContinueResponseUseCase
var _ ports.ContinueResponseUseCase = (*ParetoContinueResponse)(nil)
