package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// CleanupOrphanedData provides cleanup operations for incomplete streaming data
type CleanupOrphanedData struct {
	messageRepo  ports.MessageRepository
	sentenceRepo ports.SentenceRepository
}

// NewCleanupOrphanedData creates a new cleanup service
func NewCleanupOrphanedData(
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
) *CleanupOrphanedData {
	return &CleanupOrphanedData{
		messageRepo:  messageRepo,
		sentenceRepo: sentenceRepo,
	}
}

// CleanupOrphanedDataInput contains parameters for cleanup operations
type CleanupOrphanedDataInput struct {
	// MaxAge is the maximum age of orphaned data to clean up (e.g., 1 hour)
	// Data older than this will be marked as failed
	MaxAge time.Duration
	// DryRun if true, only reports what would be cleaned without making changes
	DryRun bool
	// ConversationID if set, only cleans data for this conversation
	ConversationID string
}

// CleanupOrphanedDataOutput contains the results of cleanup operations
type CleanupOrphanedDataOutput struct {
	OrphanedMessagesFound    int
	OrphanedMessagesCleaned  int
	OrphanedSentencesFound   int
	OrphanedSentencesCleaned int
	Errors                   []error
}

// Execute performs cleanup of orphaned streaming data
// This identifies messages and sentences that are stuck in 'streaming' or 'pending' state
// and marks them as failed if they're older than the specified age
func (uc *CleanupOrphanedData) Execute(ctx context.Context, input *CleanupOrphanedDataInput) (*CleanupOrphanedDataOutput, error) {
	output := &CleanupOrphanedDataOutput{
		Errors: make([]error, 0),
	}

	log.Printf("Starting cleanup of orphaned data (maxAge: %v, dryRun: %v, conversationID: %s)",
		input.MaxAge, input.DryRun, input.ConversationID)

	// Clean up orphaned sentences first
	sentencesOutput, err := uc.cleanupOrphanedSentences(ctx, input)
	if err != nil {
		output.Errors = append(output.Errors, fmt.Errorf("failed to cleanup sentences: %w", err))
	} else {
		output.OrphanedSentencesFound = sentencesOutput.Found
		output.OrphanedSentencesCleaned = sentencesOutput.Cleaned
	}

	// Then clean up orphaned messages
	messagesOutput, err := uc.cleanupOrphanedMessages(ctx, input)
	if err != nil {
		output.Errors = append(output.Errors, fmt.Errorf("failed to cleanup messages: %w", err))
	} else {
		output.OrphanedMessagesFound = messagesOutput.Found
		output.OrphanedMessagesCleaned = messagesOutput.Cleaned
	}

	log.Printf("Cleanup completed: messages=%d/%d, sentences=%d/%d, errors=%d",
		output.OrphanedMessagesCleaned, output.OrphanedMessagesFound,
		output.OrphanedSentencesCleaned, output.OrphanedSentencesFound,
		len(output.Errors))

	return output, nil
}

type cleanupResult struct {
	Found   int
	Cleaned int
}

// cleanupOrphanedSentences finds and cleans up sentences stuck in incomplete states
func (uc *CleanupOrphanedData) cleanupOrphanedSentences(ctx context.Context, input *CleanupOrphanedDataInput) (*cleanupResult, error) {
	result := &cleanupResult{}

	cutoffTime := time.Now().Add(-input.MaxAge)

	var orphanedSentences []*models.Sentence
	var err error

	if input.ConversationID != "" {
		orphanedSentences, err = uc.sentenceRepo.GetIncompleteByConversation(ctx, input.ConversationID, cutoffTime)
	} else {
		orphanedSentences, err = uc.sentenceRepo.GetIncompleteOlderThan(ctx, cutoffTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get incomplete sentences: %w", err)
	}

	result.Found = len(orphanedSentences)

	if input.DryRun {
		log.Printf("DRY RUN: Would mark %d orphaned sentences as failed", result.Found)
		return result, nil
	}

	for _, sentence := range orphanedSentences {
		sentence.MarkAsFailed()
		if err := uc.sentenceRepo.Update(ctx, sentence); err != nil {
			log.Printf("Failed to mark sentence %s as failed: %v", sentence.ID, err)
		} else {
			result.Cleaned++
		}
	}

	log.Printf("Marked %d/%d orphaned sentences as failed", result.Cleaned, result.Found)
	return result, nil
}

// cleanupOrphanedMessages finds and cleans up messages stuck in incomplete states
func (uc *CleanupOrphanedData) cleanupOrphanedMessages(ctx context.Context, input *CleanupOrphanedDataInput) (*cleanupResult, error) {
	result := &cleanupResult{}

	cutoffTime := time.Now().Add(-input.MaxAge)

	var orphanedMessages []*models.Message
	var err error

	if input.ConversationID != "" {
		orphanedMessages, err = uc.messageRepo.GetIncompleteByConversation(ctx, input.ConversationID, cutoffTime)
	} else {
		orphanedMessages, err = uc.messageRepo.GetIncompleteOlderThan(ctx, cutoffTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get incomplete messages: %w", err)
	}

	result.Found = len(orphanedMessages)

	if input.DryRun {
		log.Printf("DRY RUN: Would mark %d orphaned messages as failed", result.Found)
		return result, nil
	}

	for _, message := range orphanedMessages {
		message.MarkAsFailed()
		if err := uc.messageRepo.Update(ctx, message); err != nil {
			log.Printf("Failed to mark message %s as failed: %v", message.ID, err)
		} else {
			result.Cleaned++
		}
	}

	log.Printf("Marked %d/%d orphaned messages as failed", result.Cleaned, result.Found)
	return result, nil
}

// DeleteOrphanedSentencesForMessage deletes all failed/incomplete sentences for a given message
// This is useful for cleanup after a failed streaming operation
func (uc *CleanupOrphanedData) DeleteOrphanedSentencesForMessage(ctx context.Context, messageID string) error {
	sentences, err := uc.sentenceRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get sentences for message %s: %w", messageID, err)
	}

	deletedCount := 0
	for _, sentence := range sentences {
		if sentence.IsFailed() || sentence.IsStreaming() {
			if err := uc.sentenceRepo.Delete(ctx, sentence.ID); err != nil {
				log.Printf("Failed to delete orphaned sentence %s: %v", sentence.ID, err)
			} else {
				deletedCount++
			}
		}
	}

	log.Printf("Deleted %d orphaned sentences for message %s", deletedCount, messageID)
	return nil
}

// MarkStaleStreamingDataAsFailed marks streaming data that's been in 'streaming' state
// for too long as failed. This handles cases where the cleanup defer didn't run.
func (uc *CleanupOrphanedData) MarkStaleStreamingDataAsFailed(ctx context.Context, maxAge time.Duration) error {
	cutoffTime := time.Now().Add(-maxAge)

	log.Printf("Marking stale streaming data as failed (older than %v)", cutoffTime)

	// Use the cleanup function with appropriate input
	input := &CleanupOrphanedDataInput{
		MaxAge: maxAge,
		DryRun: false,
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to mark stale data as failed: %w", err)
	}

	log.Printf("Marked %d messages and %d sentences as failed",
		output.OrphanedMessagesCleaned, output.OrphanedSentencesCleaned)

	return nil
}
