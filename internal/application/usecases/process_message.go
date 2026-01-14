package usecases

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type ProcessUserMessage struct {
	messageRepo      ports.MessageRepository
	audioRepo        ports.AudioRepository
	conversationRepo ports.ConversationRepository
	asrService       ports.ASRService
	memoryService    ports.MemoryService
	idGenerator      ports.IDGenerator
	txManager        ports.TransactionManager
}

func NewProcessUserMessage(
	messageRepo ports.MessageRepository,
	audioRepo ports.AudioRepository,
	conversationRepo ports.ConversationRepository,
	asrService ports.ASRService,
	memoryService ports.MemoryService,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *ProcessUserMessage {
	return &ProcessUserMessage{
		messageRepo:      messageRepo,
		audioRepo:        audioRepo,
		conversationRepo: conversationRepo,
		asrService:       asrService,
		memoryService:    memoryService,
		idGenerator:      idGenerator,
		txManager:        txManager,
	}
}

func (uc *ProcessUserMessage) Execute(ctx context.Context, input *ports.ProcessUserMessageInput) (*ports.ProcessUserMessageOutput, error) {
	var textContent string
	var audio *models.Audio
	var err error

	if len(input.AudioData) > 0 {
		if uc.asrService == nil {
			return nil, fmt.Errorf("audio data provided but ASR service is not available")
		}

		asrResult, err := uc.asrService.Transcribe(ctx, input.AudioData, input.AudioFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to transcribe audio: %w", err)
		}

		textContent = asrResult.Text

		audioID := uc.idGenerator.GenerateAudioID()
		audio = models.NewInputAudio(audioID, input.AudioFormat)
		audio.SetData(input.AudioData, int(asrResult.Duration*1000))

		transcriptionMeta := &models.TranscriptionMeta{
			Language:   asrResult.Language,
			Confidence: asrResult.Confidence,
			Duration:   asrResult.Duration,
			Segments:   asrResult.Segments,
		}
		audio.SetTranscriptionWithMeta(textContent, transcriptionMeta)

	} else {
		textContent = input.TextContent
	}

	if textContent == "" {
		return nil, fmt.Errorf("no text content provided or transcribed")
	}

	// Fetch the conversation to get the current tip for message branching
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	sequenceNumber, err := uc.messageRepo.GetNextSequenceNumber(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next sequence number: %w", err)
	}

	messageID := uc.idGenerator.GenerateMessageID()
	message := models.NewUserMessage(messageID, input.ConversationID, sequenceNumber, textContent)

	// Set previous_id to the current conversation tip for message branching
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		message.SetPreviousMessage(*conversation.TipMessageID)
	}

	// Wrap message and audio creation in a transaction to ensure atomicity.
	// If audio creation fails, the message creation will be rolled back.
	err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.messageRepo.Create(txCtx, message); err != nil {
			return fmt.Errorf("failed to create message: %w", err)
		}

		if audio != nil {
			audio.MessageID = messageID
			if err := uc.audioRepo.Create(txCtx, audio); err != nil {
				return fmt.Errorf("failed to create audio record: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Update conversation tip to point to the new message
	if err := uc.conversationRepo.UpdateTip(ctx, input.ConversationID, message.ID); err != nil {
		// Log but don't fail - this is a non-critical operation
		log.Printf("warning: failed to update conversation tip: %v\n", err)
	}

	var relevantMemories []*models.Memory
	if uc.memoryService != nil {
		searchResults, err := uc.memoryService.SearchWithScores(ctx, textContent, 0.7, 5)
		if err != nil {
			// INTENTIONAL ERROR SWALLOWING: Memory retrieval is optional for message processing.
			// The message has already been successfully created, and memory features
			// are enhancements, not requirements. Logging and continuing allows the core flow
			// to succeed even if optional features fail.
			log.Printf("warning: failed to retrieve memories: %v\n", err)
			relevantMemories = []*models.Memory{}
		} else {
			// Extract memories and track usage with similarity scores
			relevantMemories = make([]*models.Memory, len(searchResults))
			for i, result := range searchResults {
				relevantMemories[i] = result.Memory

				// Track memory usage with similarity score
				_, err := uc.memoryService.TrackUsage(ctx, result.Memory.ID, input.ConversationID, messageID, result.Similarity)
				if err != nil {
					// INTENTIONAL ERROR SWALLOWING: Memory usage tracking is for analytics only.
					// Failing to track usage should not prevent the message from being processed.
					log.Printf("warning: failed to track memory usage: %v\n", err)
				}
			}
		}
	}

	return &ports.ProcessUserMessageOutput{
		Message:          message,
		Audio:            audio,
		RelevantMemories: relevantMemories,
	}, nil
}
