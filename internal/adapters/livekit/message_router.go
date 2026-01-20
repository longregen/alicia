package livekit

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// MessageRouter coordinates message decoding, dispatching, and generation management
// It implements ports.LiveKitAgentCallbacks to handle LiveKit events
type MessageRouter struct {
	codec             *Codec
	dispatcher        MessageDispatcher
	generationManager ResponseGenerationManager
	protocolHandler   *ProtocolHandler
	conversationID    string
	asrService        ports.ASRService
	ttsService        ports.TTSService
	idGenerator       ports.IDGenerator
	voicePipeline     *VoicePipeline
	agent             *Agent
}

// NewMessageRouter creates a new message router
func NewMessageRouter(
	codec *Codec,
	protocolHandler *ProtocolHandler,
	handleToolUseCase ports.HandleToolUseCase,
	generateResponseUseCase ports.GenerateResponseUseCase,
	processUserMessageUseCase ports.ProcessUserMessageUseCase,
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	toolUseRepo ports.ToolUseRepository,
	memoryUsageRepo ports.MemoryUsageRepository,
	voteRepo ports.VoteRepository,
	noteRepo ports.NoteRepository,
	memoryService ports.MemoryService,
	optimizationService ports.OptimizationServiceFull,
	conversationID string,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
	agent *Agent,
	asrMinConfidence float64,
	sendMessageUseCase ports.SendMessageUseCase,
	regenerateResponseUseCase ports.RegenerateResponseUseCase,
	continueResponseUseCase ports.ContinueResponseUseCase,
	editUserMessageUseCase ports.EditUserMessageUseCase,
	editAssistantMessageUseCase ports.EditAssistantMessageUseCase,
	synthesizeSpeechUseCase ports.SynthesizeSpeechUseCase,
) *MessageRouter {
	// Create the generation manager
	generationManager := NewDefaultResponseGenerationManager()

	// Create the dispatcher with all dependencies
	dispatcher := NewDefaultMessageDispatcher(
		protocolHandler,
		handleToolUseCase,
		generateResponseUseCase,
		processUserMessageUseCase,
		conversationRepo,
		messageRepo,
		toolUseRepo,
		memoryUsageRepo,
		voteRepo,
		noteRepo,
		memoryService,
		optimizationService,
		conversationID,
		asrService,
		ttsService,
		idGenerator,
		generationManager,
		sendMessageUseCase,
		regenerateResponseUseCase,
		continueResponseUseCase,
		editUserMessageUseCase,
		editAssistantMessageUseCase,
		synthesizeSpeechUseCase,
	)

	router := &MessageRouter{
		codec:             codec,
		dispatcher:        dispatcher,
		generationManager: generationManager,
		protocolHandler:   protocolHandler,
		conversationID:    conversationID,
		asrService:        asrService,
		ttsService:        ttsService,
		idGenerator:       idGenerator,
		agent:             agent,
	}

	// Initialize voice pipeline if both ASR and TTS services are available
	// Use agent's context so pipeline is properly cancelled when agent disconnects
	if asrService != nil && ttsService != nil && agent != nil && agent.ctx != nil {
		voicePipeline, err := NewVoicePipeline(
			agent.ctx,
			asrService,
			ttsService,
			agent,
			asrMinConfidence,
		)
		if err != nil {
			log.Printf("Failed to create voice pipeline: %v", err)
		} else {
			router.voicePipeline = voicePipeline

			// Set up transcription callback
			voicePipeline.SetTranscriptionCallback(func(ctx context.Context, text string, isFinal bool) {
				router.handleVoiceTranscription(ctx, text, isFinal)
			})

			log.Printf("Voice pipeline initialized for conversation: %s", conversationID)
		}
	}

	return router
}

// OnDataReceived implements ports.LiveKitAgentCallbacks
func (r *MessageRouter) OnDataReceived(ctx context.Context, msg *ports.DataChannelMessage) error {
	// Decode the protocol message
	envelope, err := r.codec.Decode(msg.Data)
	if err != nil {
		log.Printf("Failed to decode message: %v", err)
		_ = r.sendError(ctx, protocol.ErrCodeMalformedData, "Failed to decode message", true)
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Update client stanza ID for tracking reconnection
	// Do this before routing to ensure we track even if routing fails
	if envelope.StanzaID > 0 {
		r.protocolHandler.UpdateClientStanzaID(ctx, envelope.StanzaID)
	}

	// Dispatch the message to the appropriate handler
	return r.dispatcher.DispatchMessage(ctx, envelope)
}

// OnAudioReceived implements ports.LiveKitAgentCallbacks
func (r *MessageRouter) OnAudioReceived(ctx context.Context, frame *ports.AudioFrame) error {
	// Use voice pipeline if available (preferred method with buffering and silence detection)
	if r.voicePipeline != nil {
		if err := r.voicePipeline.ProcessAudioFrame(ctx, frame); err != nil {
			log.Printf("Voice pipeline failed to process audio frame: %v", err)
			return fmt.Errorf("voice pipeline error: %w", err)
		}
		return nil
	}

	// Fallback: Direct ASR transcription (without buffering - not recommended for real-time audio)
	if r.asrService != nil {
		log.Printf("WARNING: Using direct ASR without voice pipeline. Audio may not be buffered properly.")

		// Determine audio format based on sample rate and channels
		format := fmt.Sprintf("pcm_%d_%d", frame.SampleRate, frame.Channels)

		result, err := r.asrService.Transcribe(ctx, frame.Data, format)
		if err != nil {
			log.Printf("ASR transcription failed: %v", err)
			return fmt.Errorf("ASR transcription failed: %w", err)
		}

		if result != nil && result.Text != "" {
			// Send transcription result back to client
			transcription := &protocol.Transcription{
				ID:             r.idGenerator.GenerateMessageID(),
				ConversationID: r.conversationID,
				Text:           result.Text,
				Confidence:     result.Confidence,
				Language:       result.Language,
				Final:          true,
			}

			envelope := &protocol.Envelope{
				ConversationID: r.conversationID,
				Type:           protocol.TypeTranscription,
				Body:           transcription,
			}

			if err := r.protocolHandler.SendEnvelope(ctx, envelope); err != nil {
				log.Printf("Failed to send transcription: %v", err)
				return fmt.Errorf("failed to send transcription: %w", err)
			}

			log.Printf("Sent transcription: %s (confidence: %.2f)", result.Text, result.Confidence)
		}
	}

	return nil
}

// OnParticipantConnected implements ports.LiveKitAgentCallbacks
func (r *MessageRouter) OnParticipantConnected(ctx context.Context, participant *ports.LiveKitParticipant) error {
	log.Printf("Participant connected: %s (%s)", participant.Name, participant.Identity)
	return nil
}

// OnParticipantDisconnected implements ports.LiveKitAgentCallbacks
func (r *MessageRouter) OnParticipantDisconnected(ctx context.Context, participant *ports.LiveKitParticipant) error {
	log.Printf("Participant disconnected: %s (%s)", participant.Name, participant.Identity)
	return nil
}

// OnTurnStart implements ports.LiveKitAgentCallbacks
// Called when VAD detects the user has started speaking
func (r *MessageRouter) OnTurnStart(ctx context.Context) error {
	log.Printf("VAD: Turn started for conversation: %s", r.conversationID)

	// Cancel any ongoing response generation when user starts speaking
	if r.generationManager != nil {
		r.generationManager.CancelAll()
		log.Printf("Cancelled ongoing response generation due to user speech")
	}

	return nil
}

// OnTurnEnd implements ports.LiveKitAgentCallbacks
// Called when VAD detects the user has finished speaking (after silence threshold)
func (r *MessageRouter) OnTurnEnd(ctx context.Context, durationMs int64) error {
	log.Printf("VAD: Turn ended for conversation: %s (duration: %dms)", r.conversationID, durationMs)

	// The actual response generation will be triggered by the transcription
	// callback when the voice pipeline finalizes the transcript.
	// This callback can be used for logging/metrics or to trigger
	// any turn-based processing that should happen after user finishes speaking.

	return nil
}

// handleVoiceTranscription processes transcription from the voice pipeline
// This is called when the voice pipeline detects speech and completes transcription
func (r *MessageRouter) handleVoiceTranscription(ctx context.Context, text string, isFinal bool) {
	log.Printf("Voice transcription: %s (final: %v)", text, isFinal)

	// Create transcription message
	transcription := &protocol.Transcription{
		ID:             r.idGenerator.GenerateMessageID(),
		ConversationID: r.conversationID,
		Text:           text,
		Final:          isFinal,
	}

	// Send transcription to client
	envelope := &protocol.Envelope{
		ConversationID: r.conversationID,
		Type:           protocol.TypeTranscription,
		Body:           transcription,
	}

	if err := r.protocolHandler.SendEnvelope(ctx, envelope); err != nil {
		log.Printf("Failed to send voice transcription: %v", err)
		return
	}

	// If this is a final transcription, trigger the dispatcher to handle it
	// This will create a user message and generate a response
	if isFinal {
		transcriptionEnvelope := &protocol.Envelope{
			ConversationID: r.conversationID,
			Type:           protocol.TypeTranscription,
			Body:           transcription,
		}

		if err := r.dispatcher.DispatchMessage(ctx, transcriptionEnvelope); err != nil {
			log.Printf("Failed to dispatch voice transcription: %v", err)
		}
	}
}

// Cleanup stops the voice pipeline and cleans up resources
func (r *MessageRouter) Cleanup() {
	if r.voicePipeline != nil {
		log.Printf("Stopping voice pipeline for conversation: %s", r.conversationID)
		r.voicePipeline.Stop()
		r.voicePipeline = nil
	}
}

// sendError sends an error message via the protocol handler
func (r *MessageRouter) sendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return r.protocolHandler.sendError(ctx, code, message, recoverable)
}
