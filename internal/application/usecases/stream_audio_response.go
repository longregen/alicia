package usecases

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/adapters/livekit"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
)

// StreamAudioResponseInput contains parameters for streaming audio responses
type StreamAudioResponseInput struct {
	ConversationID  string
	MessageID       string
	Voice           string
	Speed           float32
	EnableStreaming bool
	LiveKitAgent    ports.LiveKitAgent
	TTSService      ports.TTSService
}

// StreamAudioResponse coordinates streaming text-to-speech output to LiveKit
// It bridges the gap between sentence streaming from the LLM and audio publishing
type StreamAudioResponse struct {
	ttsService       ports.TTSService
	synthesizeSpeech ports.SynthesizeSpeechUseCase
	codec            *livekit.Codec
}

// NewStreamAudioResponse creates a new StreamAudioResponse use case
func NewStreamAudioResponse(
	ttsService ports.TTSService,
	synthesizeSpeech ports.SynthesizeSpeechUseCase,
) *StreamAudioResponse {
	return &StreamAudioResponse{
		ttsService:       ttsService,
		synthesizeSpeech: synthesizeSpeech,
		codec:            livekit.NewCodec(),
	}
}

// StreamSentenceAudio synthesizes and streams a single sentence to LiveKit
func (uc *StreamAudioResponse) StreamSentenceAudio(
	ctx context.Context,
	agent ports.LiveKitAgent,
	conversationID string,
	messageID string,
	sentenceID string,
	sequence int,
	text string,
	isFinal bool,
	voice string,
	stanzaID int32,
) error {
	if !agent.IsConnected() {
		return fmt.Errorf("LiveKit agent not connected")
	}

	if text == "" {
		return nil // Nothing to synthesize
	}

	// Synthesize speech for this sentence
	synthesisInput := &ports.SynthesizeSpeechInput{
		Text:            text,
		MessageID:       messageID,
		SentenceID:      sentenceID,
		Voice:           voice,
		Speed:           1.0,
		OutputFormat:    "pcm",
		EnableStreaming: false, // Use non-streaming for sentence-level synthesis
	}

	output, err := uc.synthesizeSpeech.Execute(ctx, synthesisInput)
	if err != nil {
		return fmt.Errorf("failed to synthesize sentence audio: %w", err)
	}

	// Publish audio to LiveKit track
	// The SendAudio method will handle creating the track on first call
	if err := agent.SendAudio(ctx, output.AudioData, output.Format); err != nil {
		return fmt.Errorf("failed to send audio to LiveKit: %w", err)
	}

	// Create AssistantSentence message
	sentenceMsg := &protocol.AssistantSentence{
		ID:             sentenceID,
		PreviousID:     messageID,
		ConversationID: conversationID,
		Sequence:       int32(sequence),
		Text:           text,
		IsFinal:        isFinal,
		// Audio field is optional - we're streaming via LiveKit track instead
		// Could optionally include audio data or track reference here
	}

	// Encode and send via data channel
	envelope := protocol.NewEnvelope(stanzaID, conversationID, protocol.TypeAssistantSentence, sentenceMsg)
	data, err := uc.codec.Encode(envelope)
	if err != nil {
		return fmt.Errorf("failed to encode AssistantSentence message: %w", err)
	}

	if err := agent.SendData(ctx, data); err != nil {
		return fmt.Errorf("failed to send AssistantSentence message: %w", err)
	}

	return nil
}

// ProcessResponseStream processes a streaming LLM response, synthesizing and
// publishing audio for each sentence as it arrives
func (uc *StreamAudioResponse) ProcessResponseStream(
	ctx context.Context,
	agent ports.LiveKitAgent,
	conversationID string,
	messageID string,
	responseStream <-chan *ports.ResponseStreamChunk,
	voice string,
	startStanzaID int32,
) error {
	if !agent.IsConnected() {
		return fmt.Errorf("LiveKit agent not connected")
	}

	stanzaID := startStanzaID

	for chunk := range responseStream {
		if chunk.Error != nil {
			return fmt.Errorf("error in response stream: %w", chunk.Error)
		}

		// Process sentence chunks with audio
		if chunk.SentenceID != "" && chunk.Text != "" {
			err := uc.StreamSentenceAudio(
				ctx,
				agent,
				conversationID,
				messageID,
				chunk.SentenceID,
				chunk.Sequence,
				chunk.Text,
				chunk.IsFinal,
				voice,
				stanzaID,
			)
			if err != nil {
				return fmt.Errorf("failed to stream sentence audio: %w", err)
			}

			stanzaID-- // Decrement for next server message
		}
	}

	return nil
}
