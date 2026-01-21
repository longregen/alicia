package livekit

import (
	"context"
	"fmt"
	"log"

	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

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
	generationManager := NewDefaultResponseGenerationManager()

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

			voicePipeline.SetTranscriptionCallback(func(ctx context.Context, text string, isFinal bool) {
				router.handleVoiceTranscription(ctx, text, isFinal)
			})

			log.Printf("Voice pipeline initialized for conversation: %s", conversationID)
		}
	}

	return router
}

func (r *MessageRouter) OnDataReceived(ctx context.Context, msg *ports.DataChannelMessage) error {
	envelope, err := r.codec.Decode(msg.Data)
	if err != nil {
		log.Printf("Failed to decode message: %v", err)
		_ = r.sendError(ctx, protocol.ErrCodeMalformedData, "Failed to decode message", true)
		return fmt.Errorf("failed to decode message: %w", err)
	}

	if envelope.StanzaID > 0 {
		r.protocolHandler.UpdateClientStanzaID(ctx, envelope.StanzaID)
	}

	return r.dispatcher.DispatchMessage(ctx, envelope)
}

func (r *MessageRouter) OnAudioReceived(ctx context.Context, frame *ports.AudioFrame) error {
	if r.voicePipeline != nil {
		if err := r.voicePipeline.ProcessAudioFrame(ctx, frame); err != nil {
			log.Printf("Voice pipeline failed to process audio frame: %v", err)
			return fmt.Errorf("voice pipeline error: %w", err)
		}
		return nil
	}

	if r.asrService != nil {
		log.Printf("WARNING: Using direct ASR without voice pipeline. Audio may not be buffered properly.")

		format := fmt.Sprintf("pcm_%d_%d", frame.SampleRate, frame.Channels)

		result, err := r.asrService.Transcribe(ctx, frame.Data, format)
		if err != nil {
			log.Printf("ASR transcription failed: %v", err)
			return fmt.Errorf("ASR transcription failed: %w", err)
		}

		if result != nil && result.Text != "" {
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

func (r *MessageRouter) OnParticipantConnected(ctx context.Context, participant *ports.LiveKitParticipant) error {
	log.Printf("Participant connected: %s (%s)", participant.Name, participant.Identity)
	return nil
}

func (r *MessageRouter) OnParticipantDisconnected(ctx context.Context, participant *ports.LiveKitParticipant) error {
	log.Printf("Participant disconnected: %s (%s)", participant.Name, participant.Identity)
	return nil
}

func (r *MessageRouter) OnTurnStart(ctx context.Context) error {
	log.Printf("VAD: Turn started for conversation: %s", r.conversationID)

	if r.generationManager != nil {
		r.generationManager.CancelAll()
		log.Printf("Cancelled ongoing response generation due to user speech")
	}

	return nil
}

func (r *MessageRouter) OnTurnEnd(ctx context.Context, durationMs int64) error {
	log.Printf("VAD: Turn ended for conversation: %s (duration: %dms)", r.conversationID, durationMs)
	return nil
}

func (r *MessageRouter) handleVoiceTranscription(ctx context.Context, text string, isFinal bool) {
	log.Printf("Voice transcription: %s (final: %v)", text, isFinal)

	transcription := &protocol.Transcription{
		ID:             r.idGenerator.GenerateMessageID(),
		ConversationID: r.conversationID,
		Text:           text,
		Final:          isFinal,
	}

	envelope := &protocol.Envelope{
		ConversationID: r.conversationID,
		Type:           protocol.TypeTranscription,
		Body:           transcription,
	}

	if err := r.protocolHandler.SendEnvelope(ctx, envelope); err != nil {
		log.Printf("Failed to send voice transcription: %v", err)
		return
	}

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

func (r *MessageRouter) Cleanup() {
	if r.voicePipeline != nil {
		log.Printf("Stopping voice pipeline for conversation: %s", r.conversationID)
		r.voicePipeline.Stop()
		r.voicePipeline = nil
	}
}

func (r *MessageRouter) sendError(ctx context.Context, code int32, message string, recoverable bool) error {
	return r.protocolHandler.sendError(ctx, code, message, recoverable)
}
