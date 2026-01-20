package usecases

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/ports"
)

// SendMessage orchestrates user message creation and response generation.
// It combines ProcessUserMessage and GenerateResponse into a single use case,
// allowing adapters (REST, LiveKit) to use a single call for the complete
// send-and-respond flow.
type SendMessage struct {
	conversationRepo   ports.ConversationRepository
	messageRepo        ports.MessageRepository
	processUserMessage *ProcessUserMessage
	generateResponse   ports.GenerateResponseUseCase
	txManager          ports.TransactionManager
}

// NewSendMessage creates a new SendMessage use case
func NewSendMessage(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	processUserMessage *ProcessUserMessage,
	generateResponse ports.GenerateResponseUseCase,
	txManager ports.TransactionManager,
) *SendMessage {
	return &SendMessage{
		conversationRepo:   conversationRepo,
		messageRepo:        messageRepo,
		processUserMessage: processUserMessage,
		generateResponse:   generateResponse,
		txManager:          txManager,
	}
}

// Execute processes a user message and generates an assistant response.
// It orchestrates the ProcessUserMessage and GenerateResponse use cases.
func (uc *SendMessage) Execute(ctx context.Context, input *ports.SendMessageInput) (*ports.SendMessageOutput, error) {
	// 1. Validate conversation exists and is active
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("conversation not found: %s", input.ConversationID)
	}

	if conversation.Status != "active" {
		return nil, fmt.Errorf("conversation is not active: %s (status: %s)", input.ConversationID, conversation.Status)
	}

	// 2. Call ProcessUserMessage.Execute() to create the user message
	processInput := &ports.ProcessUserMessageInput{
		ConversationID: input.ConversationID,
		TextContent:    input.TextContent,
		AudioData:      input.AudioData,
		AudioFormat:    input.AudioFormat,
		PreviousID:     input.PreviousID,
	}

	processOutput, err := uc.processUserMessage.Execute(ctx, processInput)
	if err != nil {
		return nil, fmt.Errorf("failed to process user message: %w", err)
	}

	// 3. Call GenerateResponse.Execute() to generate the assistant response
	generateInput := &ports.GenerateResponseInput{
		ConversationID:   input.ConversationID,
		UserMessageID:    processOutput.Message.ID,
		RelevantMemories: processOutput.RelevantMemories,
		EnableTools:      input.EnableTools,
		EnableReasoning:  input.EnableReasoning,
		EnableStreaming:  input.EnableStreaming,
		PreviousID:       processOutput.Message.ID,
	}

	generateOutput, err := uc.generateResponse.Execute(ctx, generateInput)
	if err != nil {
		// Compensating action: delete the orphaned user message since response generation failed.
		// This prevents orphaned user messages when GenerateResponse fails to start.
		// Note: For streaming mode, if the stream starts successfully but fails mid-stream,
		// the GenerateResponse.processStream goroutine handles its own cleanup.
		if deleteErr := uc.messageRepo.Delete(ctx, processOutput.Message.ID); deleteErr != nil {
			log.Printf("warning: failed to clean up user message %s after response generation failure: %v\n",
				processOutput.Message.ID, deleteErr)
		}
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// 4. Return combined output with UserMessage, Audio, RelevantMemories, AssistantMessage/StreamChannel
	return &ports.SendMessageOutput{
		UserMessage:      processOutput.Message,
		Audio:            processOutput.Audio,
		RelevantMemories: processOutput.RelevantMemories,
		AssistantMessage: generateOutput.Message,
		StreamChannel:    generateOutput.StreamChannel,
	}, nil
}
