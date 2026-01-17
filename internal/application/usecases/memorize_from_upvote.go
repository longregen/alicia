package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// MemorizeFromUpvoteInput contains the input for creating memories from upvoted content
type MemorizeFromUpvoteInput struct {
	// TargetType is the type of entity that was upvoted ("message", "conversation")
	TargetType string
	// TargetID is the ID of the upvoted entity
	TargetID string
	// Vote is the vote value (1 for upvote, -1 for downvote)
	Vote int
	// MinUpvotes is the minimum net upvotes before triggering memory extraction (default: 1)
	MinUpvotes int
}

// MemorizeFromUpvoteOutput contains the result of memory extraction from upvoted content
type MemorizeFromUpvoteOutput struct {
	// MemoriesCreated is the number of new memories created
	MemoriesCreated int
	// MemoriesSkipped is the number of memories skipped (duplicates or low importance)
	MemoriesSkipped int
	// Memories contains the created memory objects
	Memories []*models.Memory
	// Reasoning explains the extraction decision
	Reasoning string
}

// MemorizeFromUpvote creates memories from upvoted messages or conversations
type MemorizeFromUpvote struct {
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	voteRepo         ports.VoteRepository
	extractMemories  *ExtractMemories
}

// NewMemorizeFromUpvote creates a new MemorizeFromUpvote use case
func NewMemorizeFromUpvote(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	voteRepo ports.VoteRepository,
	extractMemories *ExtractMemories,
) *MemorizeFromUpvote {
	return &MemorizeFromUpvote{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		voteRepo:         voteRepo,
		extractMemories:  extractMemories,
	}
}

// Execute processes an upvote and potentially creates memories from the upvoted content
func (uc *MemorizeFromUpvote) Execute(ctx context.Context, input *MemorizeFromUpvoteInput) (*MemorizeFromUpvoteOutput, error) {
	// Only process upvotes
	if input.Vote != 1 {
		return &MemorizeFromUpvoteOutput{
			Reasoning: "Only upvotes trigger memory extraction",
		}, nil
	}

	// Set default minimum upvotes
	minUpvotes := input.MinUpvotes
	if minUpvotes <= 0 {
		minUpvotes = 1
	}

	// Check vote aggregates to see if threshold is met
	aggregates, err := uc.voteRepo.GetAggregates(ctx, input.TargetType, input.TargetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vote aggregates: %w", err)
	}

	// Only extract memories if net score meets threshold
	if aggregates.NetScore < minUpvotes {
		return &MemorizeFromUpvoteOutput{
			Reasoning: fmt.Sprintf("Net score %d below threshold %d", aggregates.NetScore, minUpvotes),
		}, nil
	}

	// Get content based on target type
	var conversationText string
	var conversationID string
	var messageID string

	switch input.TargetType {
	case "message":
		text, convID, msgID, err := uc.getMessageContent(ctx, input.TargetID)
		if err != nil {
			return nil, err
		}
		conversationText = text
		conversationID = convID
		messageID = msgID

	case "conversation":
		text, err := uc.getConversationContent(ctx, input.TargetID)
		if err != nil {
			return nil, err
		}
		conversationText = text
		conversationID = input.TargetID

	default:
		return &MemorizeFromUpvoteOutput{
			Reasoning: fmt.Sprintf("Unsupported target type for memory extraction: %s", input.TargetType),
		}, nil
	}

	if conversationText == "" {
		return &MemorizeFromUpvoteOutput{
			Reasoning: "No content to extract memories from",
		}, nil
	}

	// Extract memories using the dedicated use case
	extractOutput, err := uc.extractMemories.Execute(ctx, &ExtractMemoriesInput{
		ConversationText:    conversationText,
		ConversationContext: "This content was upvoted by the user, indicating it contains valuable information worth remembering.",
		ConversationID:      conversationID,
		MessageID:           messageID,
		DuplicateThreshold:  0.85,
		MinImportance:       0.4, // Slightly higher threshold for upvoted content
	})
	if err != nil {
		return nil, fmt.Errorf("failed to extract memories: %w", err)
	}

	log.Printf("info: extracted %d memories from upvoted %s %s (skipped %d)\n",
		len(extractOutput.CreatedMemories), input.TargetType, input.TargetID, extractOutput.SkippedCount)

	return &MemorizeFromUpvoteOutput{
		MemoriesCreated: len(extractOutput.CreatedMemories),
		MemoriesSkipped: extractOutput.SkippedCount,
		Memories:        extractOutput.CreatedMemories,
		Reasoning:       extractOutput.Reasoning,
	}, nil
}

// getMessageContent retrieves the content of a message and its context
func (uc *MemorizeFromUpvote) getMessageContent(ctx context.Context, messageID string) (text, conversationID, msgID string, err error) {
	message, err := uc.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get message: %w", err)
	}

	// Get surrounding context (previous messages)
	messages, err := uc.messageRepo.GetLatestByConversation(ctx, message.ConversationID, 10)
	if err != nil {
		log.Printf("warning: failed to get conversation context: %v\n", err)
		// Fall back to just the upvoted message
		return message.Contents, message.ConversationID, message.ID, nil
	}

	// Build context including the upvoted message
	var sb strings.Builder
	for _, msg := range messages {
		if msg.SequenceNumber <= message.SequenceNumber {
			fmt.Fprintf(&sb, "%s: %s\n\n", msg.Role, msg.Contents)
		}
	}

	return sb.String(), message.ConversationID, message.ID, nil
}

// getConversationContent retrieves all messages from a conversation
func (uc *MemorizeFromUpvote) getConversationContent(ctx context.Context, conversationID string) (string, error) {
	messages, err := uc.messageRepo.GetByConversation(ctx, conversationID)
	if err != nil {
		return "", fmt.Errorf("failed to get conversation messages: %w", err)
	}

	var sb strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&sb, "%s: %s\n\n", msg.Role, msg.Contents)
	}

	return sb.String(), nil
}
