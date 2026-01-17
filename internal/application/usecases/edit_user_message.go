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
	generateResponseUC *GenerateResponse
	idGenerator        ports.IDGenerator
	txManager          ports.TransactionManager
}

func NewEditUserMessage(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	memoryService ports.MemoryService,
	generateResponseUC *GenerateResponse,
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

	// 3. Count messages that will be deleted (for output)
	messagesAfter, err := uc.messageRepo.GetAfterSequence(ctx, targetMessage.ConversationID, targetMessage.SequenceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages after sequence: %w", err)
	}
	deletedCount := len(messagesAfter)

	// 4. Update message content and delete downstream messages atomically
	err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Update message content and timestamp
		targetMessage.Contents = input.NewContent
		targetMessage.UpdatedAt = time.Now().UTC()
		if err := uc.messageRepo.Update(txCtx, targetMessage); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}

		// Delete all messages after target sequence number
		if err := uc.messageRepo.DeleteAfterSequence(txCtx, targetMessage.ConversationID, targetMessage.SequenceNumber); err != nil {
			return fmt.Errorf("failed to delete downstream messages: %w", err)
		}

		// Update conversation tip to point to the edited message
		if err := uc.conversationRepo.UpdateTip(txCtx, targetMessage.ConversationID, targetMessage.ID); err != nil {
			return fmt.Errorf("failed to update conversation tip: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Printf("Edited user message %s, deleted %d downstream messages", targetMessage.ID, deletedCount)

	// 5. Retrieve relevant memories for the new content
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
			UpdatedMessage:   targetMessage,
			DeletedCount:     deletedCount,
			RelevantMemories: relevantMemories,
		}, nil
	}

	// 6. Generate new message ID for response
	responseMessageID := uc.idGenerator.GenerateMessageID()

	// 7. Call GenerateResponse to create new assistant response
	generateInput := &ports.GenerateResponseInput{
		ConversationID:   targetMessage.ConversationID,
		UserMessageID:    targetMessage.ID,
		MessageID:        responseMessageID,
		RelevantMemories: relevantMemories,
		EnableTools:      input.EnableTools,
		EnableReasoning:  input.EnableReasoning,
		EnableStreaming:  input.EnableStreaming,
	}

	generateOutput, err := uc.generateResponseUC.Execute(ctx, generateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// 8. Build and return output
	output := &ports.EditUserMessageOutput{
		UpdatedMessage:   targetMessage,
		DeletedCount:     deletedCount,
		RelevantMemories: relevantMemories,
	}

	if input.EnableStreaming {
		output.StreamChannel = generateOutput.StreamChannel
	} else {
		output.AssistantMessage = generateOutput.Message
	}

	return output, nil
}
