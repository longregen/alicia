package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/ports"
)

type mockLiveKitAgent struct {
	connected      bool
	sentAudio      []byte
	sentData       []byte
	sendAudioError error
	sendDataError  error
}

func newMockLiveKitAgent() *mockLiveKitAgent {
	return &mockLiveKitAgent{
		connected: true,
	}
}

func (m *mockLiveKitAgent) Connect(ctx context.Context, roomName string) error {
	m.connected = true
	return nil
}

func (m *mockLiveKitAgent) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockLiveKitAgent) SendData(ctx context.Context, data []byte) error {
	if m.sendDataError != nil {
		return m.sendDataError
	}
	m.sentData = append(m.sentData, data...)
	return nil
}

func (m *mockLiveKitAgent) SendAudio(ctx context.Context, audio []byte, format string) error {
	if m.sendAudioError != nil {
		return m.sendAudioError
	}
	m.sentAudio = append(m.sentAudio, audio...)
	return nil
}

func (m *mockLiveKitAgent) IsConnected() bool {
	return m.connected
}

func (m *mockLiveKitAgent) GetRoom() *ports.LiveKitRoom {
	return &ports.LiveKitRoom{Name: "test_room"}
}

func TestStreamAudioResponse_StreamSentenceAudio(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"Hello world",
		false,
		"test_voice",
		-1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agent.sentAudio) == 0 {
		t.Error("expected audio to be sent to LiveKit")
	}

	if len(agent.sentData) == 0 {
		t.Error("expected data message to be sent to LiveKit")
	}
}

func TestStreamAudioResponse_StreamSentenceAudioNotConnected(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()
	agent.connected = false

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"Hello world",
		false,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error when agent not connected, got nil")
	}
}

func TestStreamAudioResponse_StreamSentenceAudioEmptyText(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"",
		false,
		"test_voice",
		-1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agent.sentAudio) != 0 {
		t.Error("expected no audio to be sent for empty text")
	}

	if len(agent.sentData) != 0 {
		t.Error("expected no data to be sent for empty text")
	}
}

func TestStreamAudioResponse_StreamSentenceAudioSynthesisFails(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	ttsService.synthesizeFunc = func(ctx context.Context, text string, options *ports.TTSOptions) (*ports.TTSResult, error) {
		return nil, errors.New("synthesis failed")
	}

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"Hello world",
		false,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error when synthesis fails, got nil")
	}
}

func TestStreamAudioResponse_StreamSentenceAudioSendAudioFails(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()
	agent.sendAudioError = errors.New("send audio failed")

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"Hello world",
		false,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error when sending audio fails, got nil")
	}
}

func TestStreamAudioResponse_StreamSentenceAudioSendDataFails(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()
	agent.sendDataError = errors.New("send data failed")

	err := uc.StreamSentenceAudio(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		"sent_1",
		0,
		"Hello world",
		false,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error when sending data fails, got nil")
	}
}

func TestStreamAudioResponse_ProcessResponseStream(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	streamChan := make(chan *ports.ResponseStreamChunk, 3)
	go func() {
		defer close(streamChan)
		streamChan <- &ports.ResponseStreamChunk{
			SentenceID: "sent_1",
			Sequence:   0,
			Text:       "First sentence.",
			IsFinal:    false,
		}
		streamChan <- &ports.ResponseStreamChunk{
			SentenceID: "sent_2",
			Sequence:   1,
			Text:       "Second sentence.",
			IsFinal:    true,
		}
	}()

	err := uc.ProcessResponseStream(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		streamChan,
		"test_voice",
		-1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agent.sentAudio) == 0 {
		t.Error("expected audio to be sent to LiveKit")
	}
}

func TestStreamAudioResponse_ProcessResponseStreamNotConnected(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()
	agent.connected = false

	streamChan := make(chan *ports.ResponseStreamChunk)
	close(streamChan)

	err := uc.ProcessResponseStream(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		streamChan,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error when agent not connected, got nil")
	}
}

func TestStreamAudioResponse_ProcessResponseStreamWithError(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	streamChan := make(chan *ports.ResponseStreamChunk, 2)
	go func() {
		defer close(streamChan)
		streamChan <- &ports.ResponseStreamChunk{
			SentenceID: "sent_1",
			Sequence:   0,
			Text:       "First sentence.",
			IsFinal:    false,
		}
		streamChan <- &ports.ResponseStreamChunk{
			Error: errors.New("stream error"),
		}
	}()

	err := uc.ProcessResponseStream(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		streamChan,
		"test_voice",
		-1,
	)

	if err == nil {
		t.Fatal("expected error from stream, got nil")
	}
}

func TestStreamAudioResponse_ProcessResponseStreamEmptyChunks(t *testing.T) {
	ttsService := newMockTTSService()
	audioRepo := newMockAudioRepo()
	sentenceRepo := newMockSentenceRepo()
	idGen := newMockIDGenerator()

	synthesizeSpeech := NewSynthesizeSpeech(audioRepo, sentenceRepo, ttsService, idGen)
	uc := NewStreamAudioResponse(ttsService, synthesizeSpeech)

	agent := newMockLiveKitAgent()

	streamChan := make(chan *ports.ResponseStreamChunk, 2)
	go func() {
		defer close(streamChan)
		streamChan <- &ports.ResponseStreamChunk{
			SentenceID: "",
			Text:       "",
		}
		streamChan <- &ports.ResponseStreamChunk{
			SentenceID: "sent_1",
			Text:       "",
		}
	}()

	err := uc.ProcessResponseStream(
		context.Background(),
		agent,
		"conv_123",
		"msg_123",
		streamChan,
		"test_voice",
		-1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(agent.sentAudio) != 0 {
		t.Error("expected no audio to be sent for empty chunks")
	}
}
