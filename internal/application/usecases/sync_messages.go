package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// SyncMessages handles offline message sync with deduplication and conflict detection
type SyncMessages struct {
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	idGenerator      ports.IDGenerator
	txManager        ports.TransactionManager
}

// NewSyncMessages creates a new SyncMessages use case
func NewSyncMessages(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *SyncMessages {
	return &SyncMessages{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		idGenerator:      idGenerator,
		txManager:        txManager,
	}
}

// Execute processes a batch of messages for offline sync
func (uc *SyncMessages) Execute(ctx context.Context, input *ports.SyncMessagesInput) (*ports.SyncMessagesOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if input.ConversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	// Verify conversation exists
	_, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	results := make([]ports.SyncedMessageResult, 0, len(input.Messages))

	// Process each message within a transaction for atomicity
	// Track database errors separately from conflicts - conflicts are expected, database errors should rollback
	err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		var firstDBError error

		for _, msgInput := range input.Messages {
			result, err := uc.processMessage(txCtx, input.ConversationID, msgInput)
			if err != nil {
				// This is a database/infrastructure error - track it for rollback
				// but continue processing to collect all results for reporting
				results = append(results, ports.SyncedMessageResult{
					LocalID: msgInput.LocalID,
					Status:  "error",
				})
				if firstDBError == nil {
					firstDBError = err
				}
				continue
			}
			results = append(results, result)
		}

		// If any database error occurred, return it to trigger transaction rollback
		// Conflicts (status="conflict") are not errors - they are expected and should commit
		if firstDBError != nil {
			return firstDBError
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to sync messages: %w", err)
	}

	return &ports.SyncMessagesOutput{
		Results:  results,
		SyncedAt: time.Now().UTC(),
	}, nil
}

// processMessage handles a single message sync with deduplication and conflict detection
func (uc *SyncMessages) processMessage(ctx context.Context, conversationID string, msgInput ports.SyncMessageItem) (ports.SyncedMessageResult, error) {
	// Validation
	if msgInput.LocalID == "" {
		return ports.SyncedMessageResult{
			LocalID: msgInput.LocalID,
			Status:  "conflict",
		}, nil
	}

	if msgInput.Role == "" {
		return ports.SyncedMessageResult{
			LocalID: msgInput.LocalID,
			Status:  "conflict",
		}, nil
	}

	// Step 1: Check for duplicates by LocalID
	existingMsg, err := uc.messageRepo.GetByLocalID(ctx, msgInput.LocalID)
	if err != nil && !isNotFoundError(err) {
		return ports.SyncedMessageResult{}, fmt.Errorf("failed to check for existing message: %w", err)
	}

	// Step 2: If exists with different content → mark as conflict
	if existingMsg != nil {
		if existingMsg.Contents != msgInput.Contents {
			// Content differs - conflict detected
			existingMsg.MarkAsConflict()
			if err := uc.messageRepo.Update(ctx, existingMsg); err != nil {
				return ports.SyncedMessageResult{}, fmt.Errorf("failed to update message conflict status: %w", err)
			}

			return ports.SyncedMessageResult{
				LocalID:  msgInput.LocalID,
				ServerID: existingMsg.ServerID,
				Status:   "conflict",
				Message:  existingMsg,
			}, nil
		}

		// Step 3: If exists with same content → return as already synced
		return ports.SyncedMessageResult{
			LocalID:  msgInput.LocalID,
			ServerID: existingMsg.ServerID,
			Status:   "synced",
			Message:  existingMsg,
		}, nil
	}

	// Step 4: If new → create with sync metadata
	serverID := uc.idGenerator.GenerateMessageID()

	// Use provided timestamps or default to now
	createdAt := msgInput.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	updatedAt := msgInput.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	now := time.Now().UTC()
	message := &models.Message{
		ID:               serverID,
		ConversationID:   conversationID,
		SequenceNumber:   msgInput.SequenceNumber,
		PreviousID:       msgInput.PreviousID,
		Role:             models.MessageRole(msgInput.Role),
		Contents:         msgInput.Contents,
		LocalID:          msgInput.LocalID,
		ServerID:         serverID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted, // Synced messages are always completed
		SyncedAt:         &now,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return ports.SyncedMessageResult{}, fmt.Errorf("failed to create message: %w", err)
	}

	return ports.SyncedMessageResult{
		LocalID:  msgInput.LocalID,
		ServerID: serverID,
		Status:   "synced",
		Message:  message,
	}, nil
}

// isNotFoundError checks if the error is a "not found" error
// This handles pgx.ErrNoRows and domain-level not found errors
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrMessageNotFound)
}
