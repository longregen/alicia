package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// ContinueResponse generates a continuation and appends it to an existing assistant message.
// It composes with GenerateResponse to handle the actual LLM generation.
//
// Note: For more advanced continuation handling (e.g., providing the existing content
// as context for the LLM), the GenerateResponse use case may need a ContinueFromContent
// field added to its input. For now, we use the PreviousID mechanism which creates a
// new message that follows the target message in the conversation chain.
type ContinueResponse struct {
	messageRepo        ports.MessageRepository
	conversationRepo   ports.ConversationRepository
	generateResponseUC *GenerateResponse
	idGenerator        ports.IDGenerator
	txManager          ports.TransactionManager
}

// NewContinueResponse creates a new ContinueResponse use case instance
func NewContinueResponse(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	generateResponseUC *GenerateResponse,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *ContinueResponse {
	return &ContinueResponse{
		messageRepo:        messageRepo,
		conversationRepo:   conversationRepo,
		generateResponseUC: generateResponseUC,
		idGenerator:        idGenerator,
		txManager:          txManager,
	}
}

// Execute generates a continuation for the specified assistant message and appends it.
//
// The process:
// 1. Get target message by ID and validate it's an assistant message
// 2. Get the user message that triggered this response (via PreviousID chain)
// 3. Call GenerateResponse.Execute() with PreviousID set to target message
// 4. If streaming: return stream channel, handle appending in stream processor
// 5. If not streaming: append generated content to target message with "\n\n" separator
// 6. Update target message in repository
// 7. Return updated message and appended content
func (uc *ContinueResponse) Execute(ctx context.Context, input *ports.ContinueResponseInput) (*ports.ContinueResponseOutput, error) {
	// 1. Get target message by ID
	targetMessage, err := uc.messageRepo.GetByID(ctx, input.TargetMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target message: %w", err)
	}
	if targetMessage == nil {
		return nil, fmt.Errorf("target message not found: %s", input.TargetMessageID)
	}

	// 2. Validate it's an assistant message
	if !targetMessage.IsFromAssistant() {
		return nil, fmt.Errorf("cannot continue: target message is not an assistant message (role=%s)", targetMessage.Role)
	}

	// 3. Get the user message that triggered this response (via PreviousID chain)
	if targetMessage.PreviousID == "" {
		return nil, fmt.Errorf("cannot continue: target message has no previous message reference")
	}

	// Verify the previous message exists and is a user message
	userMessage, err := uc.messageRepo.GetByID(ctx, targetMessage.PreviousID)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous message: %w", err)
	}
	if userMessage == nil {
		return nil, fmt.Errorf("previous message not found: %s", targetMessage.PreviousID)
	}
	if !userMessage.IsFromUser() {
		return nil, fmt.Errorf("cannot continue: previous message is not a user message (role=%s)", userMessage.Role)
	}

	// Pre-generate the continuation message ID for tracking
	continuationMsgID := uc.idGenerator.GenerateMessageID()

	// 4. Call GenerateResponse.Execute() with PreviousID set to target message
	// This tells it to continue from that message (creates a new message in the chain)
	generateInput := &ports.GenerateResponseInput{
		ConversationID:  targetMessage.ConversationID,
		UserMessageID:   userMessage.ID,
		MessageID:       continuationMsgID,
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		PreviousID:      targetMessage.ID, // Use target message as previous - this is the continuation point
		Notifier:        input.Notifier,
	}

	generateOutput, err := uc.generateResponseUC.Execute(ctx, generateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to generate continuation: %w", err)
	}

	// 5. If streaming: return stream channel, handle appending in stream processor
	if input.EnableStreaming {
		// For streaming mode, we need to wrap the stream to handle appending
		// when streaming completes
		wrappedStream := make(chan *ports.ResponseStreamChunk, 10)

		go uc.processStreamAndAppend(ctx, targetMessage, generateOutput, wrappedStream)

		return &ports.ContinueResponseOutput{
			TargetMessage:   targetMessage,
			StreamChannel:   wrappedStream,
			GeneratedOutput: generateOutput,
		}, nil
	}

	// 6. If not streaming: append generated content to target message with "\n\n" separator
	appendedContent := ""
	if generateOutput.Message != nil && generateOutput.Message.Contents != "" {
		appendedContent = generateOutput.Message.Contents

		// Append to target message
		err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
			// Append with separator if target message has existing content
			if targetMessage.Contents != "" {
				targetMessage.Contents += "\n\n" + appendedContent
			} else {
				targetMessage.Contents = appendedContent
			}
			targetMessage.UpdatedAt = time.Now().UTC()

			// 7. Update target message in repository
			if err := uc.messageRepo.Update(txCtx, targetMessage); err != nil {
				return fmt.Errorf("failed to update target message: %w", err)
			}

			// Delete the continuation message since we've appended its content to the target
			// This keeps the message chain clean
			if err := uc.messageRepo.Delete(txCtx, generateOutput.Message.ID); err != nil {
				// Log but don't fail - the content has been appended
				log.Printf("warning: failed to delete continuation message %s: %v", generateOutput.Message.ID, err)
			}

			// Update conversation tip to point back to the target message
			// (since we deleted the continuation message)
			if err := uc.conversationRepo.UpdateTip(txCtx, targetMessage.ConversationID, targetMessage.ID); err != nil {
				return fmt.Errorf("failed to update conversation tip: %w", err)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	// 8. Return updated message and appended content
	return &ports.ContinueResponseOutput{
		TargetMessage:   targetMessage,
		AppendedContent: appendedContent,
		GeneratedOutput: generateOutput,
	}, nil
}

// processStreamAndAppend handles streaming mode by forwarding chunks to the wrapped stream
// and appending the final content to the target message when streaming completes.
func (uc *ContinueResponse) processStreamAndAppend(
	ctx context.Context,
	targetMessage *models.Message,
	generateOutput *ports.GenerateResponseOutput,
	wrappedStream chan<- *ports.ResponseStreamChunk,
) {
	defer close(wrappedStream)

	if generateOutput.StreamChannel == nil {
		return
	}

	var contentBuilder strings.Builder

	// Forward all chunks and accumulate content
	for chunk := range generateOutput.StreamChannel {
		// Accumulate text content for appending later
		if chunk.Text != "" {
			contentBuilder.WriteString(chunk.Text)
		}

		// Forward chunk to wrapped stream
		select {
		case wrappedStream <- chunk:
		case <-ctx.Done():
			return
		}

		// Check for errors
		if chunk.Error != nil {
			return
		}
	}

	// Streaming complete - append accumulated content to target message
	appendedContent := strings.TrimSpace(contentBuilder.String())
	if appendedContent == "" {
		return
	}

	// Use a fresh context for the final update since the streaming context may be done
	updateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := uc.txManager.WithTransaction(updateCtx, func(txCtx context.Context) error {
		// Refresh target message to get latest state
		freshTarget, err := uc.messageRepo.GetByID(txCtx, targetMessage.ID)
		if err != nil {
			return fmt.Errorf("failed to refresh target message: %w", err)
		}

		// Append with separator if target message has existing content
		if freshTarget.Contents != "" {
			freshTarget.Contents += "\n\n" + appendedContent
		} else {
			freshTarget.Contents = appendedContent
		}
		freshTarget.UpdatedAt = time.Now().UTC()

		if err := uc.messageRepo.Update(txCtx, freshTarget); err != nil {
			return fmt.Errorf("failed to update target message: %w", err)
		}

		// Delete the continuation message since we've appended its content
		if generateOutput.Message != nil {
			if err := uc.messageRepo.Delete(txCtx, generateOutput.Message.ID); err != nil {
				log.Printf("warning: failed to delete continuation message %s: %v", generateOutput.Message.ID, err)
			}
		}

		// Update conversation tip to point back to the target message
		if err := uc.conversationRepo.UpdateTip(txCtx, targetMessage.ConversationID, targetMessage.ID); err != nil {
			return fmt.Errorf("failed to update conversation tip: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("error: failed to append continuation content: %v", err)
		// Send error to stream
		select {
		case wrappedStream <- &ports.ResponseStreamChunk{Error: err}:
		default:
		}
		return
	}

	// Update the target message reference for callers
	targetMessage.Contents += "\n\n" + appendedContent
	targetMessage.UpdatedAt = time.Now().UTC()
}
