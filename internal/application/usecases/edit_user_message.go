package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type EditUserMessage struct {
	messageRepo        ports.MessageRepository
	conversationRepo   ports.ConversationRepository
	memoryService      ports.MemoryService
	generateResponseUC ports.GenerateResponseUseCase
	idGenerator        ports.IDGenerator
	txManager          ports.TransactionManager
}

func NewEditUserMessage(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	memoryService ports.MemoryService,
	generateResponseUC ports.GenerateResponseUseCase,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *EditUserMessage {
	return &EditUserMessage{
		messageRepo:        messageRepo,
		conversationRepo:   conversationRepo,
		memoryService:      memoryService,
		generateResponseUC: generateResponseUC,
		idGenerator:        idGenerator,
		txManager:          txManager,
	}
}

func (uc *EditUserMessage) Execute(ctx context.Context, input *ports.EditUserMessageInput) (*ports.EditUserMessageOutput, error) {
	// 1. Get target message by ID
	targetMessage, err := uc.messageRepo.GetByID(ctx, input.TargetMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target message: %w", err)
	}

	// 2. Validate it's a user message
	if targetMessage.Role != models.MessageRoleUser {
		return nil, fmt.Errorf("cannot edit message: expected user message but got %s", targetMessage.Role)
	}

	// Validate conversation ID matches
	if targetMessage.ConversationID != input.ConversationID {
		return nil, fmt.Errorf("message %s does not belong to conversation %s", input.TargetMessageID, input.ConversationID)
	}

	// Validate new content is provided
	if input.NewContent == "" {
		return nil, fmt.Errorf("new content is required for user message edit")
	}

	// 3. Count existing siblings for informational purposes
	existingSiblings, err := uc.messageRepo.GetSiblings(ctx, targetMessage.ID)
	if err != nil {
		// Non-fatal - just log and continue
		log.Printf("warning: failed to get siblings for message %s: %v", targetMessage.ID, err)
		existingSiblings = []*models.Message{}
	}
	siblingCount := len(existingSiblings)

	// 4. Create a NEW user message as a sibling of the original
	// The new message shares the same PreviousID as the original message,
	// making it a sibling branch rather than replacing the original
	newMessageID := uc.idGenerator.GenerateMessageID()
	now := time.Now().UTC()

	newUserMessage := &models.Message{
		ID:               newMessageID,
		ConversationID:   targetMessage.ConversationID,
		SequenceNumber:   targetMessage.SequenceNumber, // Same sequence as original (sibling)
		PreviousID:       targetMessage.PreviousID,     // Same parent - this makes it a sibling
		Role:             models.MessageRoleUser,
		Contents:         input.NewContent,
		CreatedAt:        now,
		UpdatedAt:        now,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
	}

	// 5. Persist the new message and update conversation tip atomically
	err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create the new sibling message
		if err := uc.messageRepo.Create(txCtx, newUserMessage); err != nil {
			return fmt.Errorf("failed to create new user message: %w", err)
		}

		// Update conversation tip to point to the new message
		// This switches the "active branch" to the new edit
		if err := uc.conversationRepo.UpdateTip(txCtx, targetMessage.ConversationID, newUserMessage.ID); err != nil {
			return fmt.Errorf("failed to update conversation tip: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Printf("Created sibling message %s for edited user message %s (now %d siblings)", newUserMessage.ID, targetMessage.ID, siblingCount+1)

	// 6. Retrieve relevant memories for the new content
	var relevantMemories []*models.Memory
	if uc.memoryService != nil {
		searchResults, err := uc.memoryService.SearchWithScores(ctx, input.NewContent, 0.7, 5)
		if err != nil {
			// Log but don't fail - memory retrieval is optional
			log.Printf("warning: failed to retrieve memories for edited message: %v\n", err)
			relevantMemories = []*models.Memory{}
		} else {
			relevantMemories = make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				relevantMemories[i] = result.Memory
			}
		}
	}

	// If SkipGeneration is true, return early without generating a response
	// This is used when the agent handles response generation via WebSocket
	if input.SkipGeneration {
		return &ports.EditUserMessageOutput{
			UpdatedMessage:   newUserMessage,
			DeletedCount:     0, // No messages deleted - we preserve history now
			RelevantMemories: relevantMemories,
		}, nil
	}

	// 7. Generate new message ID for response
	responseMessageID := uc.idGenerator.GenerateMessageID()

	// 8. Call GenerateResponse to create new assistant response
	// The response will be linked to the new user message, creating a new branch
	generateInput := &ports.GenerateResponseInput{
		ConversationID:   newUserMessage.ConversationID,
		UserMessageID:    newUserMessage.ID,
		MessageID:        responseMessageID,
		PreviousID:       newUserMessage.ID, // Response follows the new user message
		RelevantMemories: relevantMemories,
		EnableTools:      input.EnableTools,
		EnableReasoning:  input.EnableReasoning,
		EnableStreaming:  input.EnableStreaming,
	}

	generateOutput, err := uc.generateResponseUC.Execute(ctx, generateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// 9. Build and return output
	output := &ports.EditUserMessageOutput{
		UpdatedMessage:   newUserMessage,
		DeletedCount:     0, // No messages deleted - branching preserves history
		RelevantMemories: relevantMemories,
	}

	if input.EnableStreaming {
		output.StreamChannel = generateOutput.StreamChannel
	} else {
		output.AssistantMessage = generateOutput.Message
	}

	return output, nil
}
