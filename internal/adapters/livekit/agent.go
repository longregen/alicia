package livekit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/longregen/alicia/internal/ports"
	"github.com/longregen/alicia/pkg/protocol"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	AckTimeout       = 5 * time.Second
	MaxRetries       = 3
	WorkQueueTimeout = 100 * time.Millisecond
)

type PendingMessage struct {
	StanzaID   int32
	Data       []byte
	SentAt     time.Time
	RetryCount int
}

type AgentConfig struct {
	URL                   string
	APIKey                string
	APISecret             string
	AgentIdentity         string
	AgentName             string
	TokenValidityDuration time.Duration
	WorkerCount           int // Number of worker goroutines for event processing
	WorkQueueSize         int // Size of the buffered work queue

	// VAD configuration
	VADModelPath         string  // Path to Silero VAD ONNX model (empty to disable VAD)
	VADSilenceDurationMs int     // Silence duration to mark end of turn (default: 1200ms)
	VADThreshold         float32 // Speech detection threshold (default: 0.5)

	// TTS audio format configuration (for resampling to 48kHz stereo)
	TTSSampleRate int // TTS output sample rate (default: 24000 for Kokoro)
	TTSChannels   int // TTS output channels: 1=mono, 2=stereo (default: 1)
}

func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		URL:                   "ws://localhost:7880",
		APIKey:                "",
		APISecret:             "",
		AgentIdentity:         "alicia-agent",
		AgentName:             "Alicia Agent",
		TokenValidityDuration: 24 * time.Hour,
		WorkerCount:           10,
		WorkQueueSize:         100,

		// VAD defaults
		VADModelPath:         "",   // Disabled by default
		VADSilenceDurationMs: 1200, // 1.2 seconds
		VADThreshold:         0.5,

		// TTS defaults (Kokoro outputs 24kHz mono)
		TTSSampleRate: 24000,
		TTSChannels:   1,
	}
}

type Agent struct {
	config    *AgentConfig
	callbacks ports.LiveKitAgentCallbacks
	codec     *Codec

	mu        sync.RWMutex
	room      *lksdk.Room
	roomInfo  *ports.LiveKitRoom
	connected bool

	audioTrack     *lksdk.LocalTrack
	audioConverter *AudioConverter

	// VAD for turn detection
	vadProcessor *VADProcessor

	// Acknowledgement tracking
	pendingMu      sync.RWMutex
	pendingAcks    map[int32]*PendingMessage // stanzaId -> PendingMessage
	lastStanzaID   int32                     // Last stanzaId used by server (negative, decrementing)
	ackCheckTicker *time.Ticker
	ackCheckDone   chan bool

	// Worker pool for event processing
	workQueue   chan func()
	workerCount int

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewAgent(config *AgentConfig, callbacks ports.LiveKitAgentCallbacks) (*Agent, error) {
	if config == nil {
		config = DefaultAgentConfig()
	}

	if config.URL == "" {
		return nil, fmt.Errorf("LiveKit URL is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("LiveKit API key is required")
	}

	if config.APISecret == "" {
		return nil, fmt.Errorf("LiveKit API secret is required")
	}

	if config.TokenValidityDuration == 0 {
		config.TokenValidityDuration = 24 * time.Hour
	}

	if config.WorkerCount <= 0 {
		config.WorkerCount = 10
	}

	if config.WorkQueueSize <= 0 {
		config.WorkQueueSize = 100
	}

	// Create cancellable context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		config:       config,
		callbacks:    callbacks,
		codec:        NewCodec(),
		connected:    false,
		pendingAcks:  make(map[int32]*PendingMessage),
		lastStanzaID: 0,
		ackCheckDone: make(chan bool),
		workerCount:  config.WorkerCount,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize VAD processor if model path is configured
	if config.VADModelPath != "" {
		silenceDuration := config.VADSilenceDurationMs
		if silenceDuration <= 0 {
			silenceDuration = VADMinSilenceDurationMs
		}
		threshold := config.VADThreshold
		if threshold <= 0 {
			threshold = VADThreshold
		}

		vadProcessor, err := NewVADProcessor(VADConfig{
			ModelPath:            config.VADModelPath,
			MinSilenceDurationMs: silenceDuration,
			Threshold:            threshold,
			OnTurnStart: func() {
				if agent.callbacks != nil {
					if err := agent.callbacks.OnTurnStart(agent.ctx); err != nil {
						log.Printf("ERROR: OnTurnStart callback failed: %v", err)
					}
				}
			},
			OnTurnEnd: func(durationMs int64) {
				if agent.callbacks != nil {
					if err := agent.callbacks.OnTurnEnd(agent.ctx, durationMs); err != nil {
						log.Printf("ERROR: OnTurnEnd callback failed: %v", err)
					}
				}
			},
		})
		if err != nil {
			cancel() // Clean up context on error
			return nil, fmt.Errorf("failed to create VAD processor: %w", err)
		}
		agent.vadProcessor = vadProcessor
		log.Printf("VAD processor initialized with model: %s, silence threshold: %dms, detection threshold: %.2f",
			config.VADModelPath, silenceDuration, threshold)
	}

	return agent, nil
}

func (a *Agent) SetCallbacks(callbacks ports.LiveKitAgentCallbacks) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.callbacks = callbacks
}

func (a *Agent) Connect(ctx context.Context, roomName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.connected {
		return fmt.Errorf("already connected to a room")
	}

	if roomName == "" {
		return fmt.Errorf("room name is required")
	}

	if a.callbacks == nil {
		return fmt.Errorf("callbacks must be set before connecting")
	}

	// Connect to LiveKit room using API credentials.
	// The SDK handles authentication internally when APIKey and APISecret are provided.
	// This allows the agent to join as a participant with the specified identity.
	room, err := lksdk.ConnectToRoom(a.config.URL, lksdk.ConnectInfo{
		APIKey:              a.config.APIKey,
		APISecret:           a.config.APISecret,
		RoomName:            roomName,
		ParticipantIdentity: a.config.AgentIdentity,
		ParticipantName:     a.config.AgentName,
	}, &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataReceived:    a.onDataReceived,
			OnTrackSubscribed: a.onTrackSubscribed,
		},
		OnParticipantConnected:    a.onParticipantConnected,
		OnParticipantDisconnected: a.onParticipantDisconnected,
	})

	if err != nil {
		a.cancel()
		return fmt.Errorf("failed to connect to room: %w", err)
	}

	a.room = room
	a.roomInfo = &ports.LiveKitRoom{
		Name:         roomName,
		SID:          room.SID(),
		Participants: []*ports.LiveKitParticipant{},
	}
	a.connected = true

	participants := room.GetRemoteParticipants()
	for _, p := range participants {
		a.roomInfo.Participants = append(a.roomInfo.Participants, &ports.LiveKitParticipant{
			ID:       p.SID(),
			Identity: p.Identity(),
			Name:     p.Name(),
		})
	}

	// Initialize worker pool
	a.workQueue = make(chan func(), a.config.WorkQueueSize)
	for i := 0; i < a.workerCount; i++ {
		a.wg.Add(1)
		go a.worker()
	}

	// Start acknowledgement timeout checker
	a.ackCheckTicker = time.NewTicker(AckTimeout)
	a.wg.Add(1)
	go a.checkAckTimeouts()

	return nil
}

func (a *Agent) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	if !a.connected {
		a.mu.Unlock()
		return fmt.Errorf("not connected to a room")
	}

	// Step 1: Cancel context to signal all goroutines to stop
	if a.cancel != nil {
		a.cancel()
	}

	// Stop acknowledgement timeout checker
	if a.ackCheckTicker != nil {
		a.ackCheckTicker.Stop()
		// Use non-blocking send to avoid deadlock if goroutine already exited
		select {
		case a.ackCheckDone <- true:
		default:
		}
	}

	// Disconnect from room to stop receiving new callbacks
	if a.room != nil {
		a.room.Disconnect()
		a.room = nil
	}

	a.connected = false
	a.roomInfo = nil
	a.audioTrack = nil

	// Clean up VAD processor
	if a.vadProcessor != nil {
		if err := a.vadProcessor.Destroy(); err != nil {
			log.Printf("WARNING: Failed to destroy VAD processor: %v", err)
		}
		a.vadProcessor = nil
	}

	// Step 2: Close work queue BEFORE releasing lock to prevent race
	// This prevents callbacks from queuing new work after we unlock
	// Any callback trying to send will immediately see the closed channel
	if a.workQueue != nil {
		close(a.workQueue)
		a.workQueue = nil
	}
	a.mu.Unlock()

	// Step 3: Wait for all worker goroutines to finish
	// Workers will exit via context cancellation (a.ctx.Done()) or closed channel
	a.wg.Wait()

	// Now it's safe to clear pending acknowledgements
	a.pendingMu.Lock()
	a.pendingAcks = make(map[int32]*PendingMessage)
	a.lastStanzaID = 0
	a.pendingMu.Unlock()

	return nil
}

func (a *Agent) SendData(ctx context.Context, data []byte) error {
	a.mu.RLock()
	if !a.connected || a.room == nil {
		a.mu.RUnlock()
		return fmt.Errorf("not connected to a room")
	}
	room := a.room // Capture room reference while holding lock
	a.mu.RUnlock()

	if len(data) == 0 {
		return fmt.Errorf("data is required")
	}

	err := room.LocalParticipant.PublishDataPacket(lksdk.UserData(data), lksdk.WithDataPublishReliable(true))
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

func (a *Agent) SendAudio(ctx context.Context, audio []byte, format string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.connected || a.room == nil {
		return fmt.Errorf("not connected to a room")
	}

	if len(audio) == 0 {
		return fmt.Errorf("audio data is required")
	}

	// Create audio track on first call
	if a.audioTrack == nil {
		track, err := lksdk.NewLocalTrack(webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		})
		if err != nil {
			return fmt.Errorf("failed to create audio track: %w", err)
		}

		_, err = a.room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
			Name:   "agent-audio",
			Source: livekit.TrackSource_MICROPHONE,
		})
		if err != nil {
			return fmt.Errorf("failed to publish audio track: %w", err)
		}

		a.audioTrack = track

		// Create audio converter for PCM -> Opus conversion
		// Default to 48kHz, 2 channels (matches track config)
		converter, err := NewAudioConverter(48000, 2)
		if err != nil {
			return fmt.Errorf("failed to create audio converter: %w", err)
		}
		a.audioConverter = converter
	}

	// Convert audio format if needed
	var samples []media.Sample

	// Normalize format string
	normalizedFormat := format
	if format == "" {
		normalizedFormat = "pcm"
	}

	switch normalizedFormat {
	case "pcm", "audio/pcm", "pcm16":
		// Convert PCM to Opus
		if a.audioConverter == nil {
			return fmt.Errorf("audio converter not initialized")
		}

		// Resample TTS audio to 48kHz stereo (LiveKit requirement)
		// TTS typically outputs 24kHz mono, we need 48kHz stereo for Opus
		inputRate := a.config.TTSSampleRate
		inputChannels := a.config.TTSChannels
		if inputRate == 0 {
			inputRate = 24000 // Default for Kokoro
		}
		if inputChannels == 0 {
			inputChannels = 1 // Default mono
		}

		resampledAudio, err := ResamplePCM(audio, inputRate, 48000, inputChannels, 2)
		if err != nil {
			return fmt.Errorf("failed to resample audio: %w", err)
		}

		opusSamples, err := a.audioConverter.ConvertPCMToOpus(resampledAudio)
		if err != nil {
			return fmt.Errorf("failed to convert PCM to Opus: %w", err)
		}
		samples = opusSamples

	case "opus", "audio/opus":
		// Audio is already in Opus format
		sample := media.Sample{
			Data:     audio,
			Duration: time.Millisecond * 20,
		}
		samples = []media.Sample{sample}

	default:
		return fmt.Errorf("unsupported audio format: %s", format)
	}

	// Write all samples to the track
	for _, sample := range samples {
		if err := a.audioTrack.WriteSample(sample, nil); err != nil {
			return fmt.Errorf("failed to write audio sample: %w", err)
		}
	}

	return nil
}

func (a *Agent) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

func (a *Agent) GetRoom() *ports.LiveKitRoom {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.roomInfo
}

func (a *Agent) SendMessageWithAck(ctx context.Context, conversationID string, msgType protocol.MessageType, body interface{}) (int32, error) {
	a.mu.RLock()
	if !a.connected {
		a.mu.RUnlock()
		return 0, fmt.Errorf("not connected to a room")
	}
	a.mu.RUnlock()

	// Generate server-side stanzaId (negative, decrementing)
	a.pendingMu.Lock()
	a.lastStanzaID--
	stanzaID := a.lastStanzaID
	a.pendingMu.Unlock()

	// Encode message
	data, err := a.codec.EncodeMessage(stanzaID, conversationID, msgType, body)
	if err != nil {
		return 0, fmt.Errorf("failed to encode message: %w", err)
	}

	// Send message
	if err := a.SendData(ctx, data); err != nil {
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	// Track for acknowledgement
	a.pendingMu.Lock()
	a.pendingAcks[stanzaID] = &PendingMessage{
		StanzaID:   stanzaID,
		Data:       data,
		SentAt:     time.Now(),
		RetryCount: 0,
	}
	a.pendingMu.Unlock()

	return stanzaID, nil
}

func (a *Agent) SendAcknowledgement(ctx context.Context, conversationID string, ackedStanzaID int32) error {
	ack := &protocol.Acknowledgement{
		ConversationID: conversationID,
		AckedStanzaID:  ackedStanzaID,
		Success:        true,
	}

	// Acknowledgements don't need their own stanzaId tracking or acks
	// Use a stanzaId of 0 for acknowledgements to indicate they're control messages
	data, err := a.codec.EncodeMessage(0, conversationID, protocol.TypeAcknowledgement, ack)
	if err != nil {
		return fmt.Errorf("failed to encode acknowledgement: %w", err)
	}

	if err := a.SendData(ctx, data); err != nil {
		return fmt.Errorf("failed to send acknowledgement: %w", err)
	}

	return nil
}

func (a *Agent) SendErrorMessage(ctx context.Context, conversationID string, code int32, message string, severity protocol.Severity, recoverable bool, originatingStanzaID int32) error {
	errorMsg := &protocol.ErrorMessage{
		ConversationID: conversationID,
		Code:           code,
		Message:        message,
		Severity:       severity,
		Recoverable:    recoverable,
	}

	// If there's an originating stanza ID, include it
	if originatingStanzaID != 0 {
		errorMsg.OriginatingID = fmt.Sprintf("%d", originatingStanzaID)
	}

	// Error messages don't need their own stanzaId tracking
	// Use a stanzaId of 0 for error messages to indicate they're control messages
	data, err := a.codec.EncodeMessage(0, conversationID, protocol.TypeErrorMessage, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to encode error message: %w", err)
	}

	// Send directly without going through the work queue to avoid infinite loop
	a.mu.RLock()
	if !a.connected || a.room == nil {
		a.mu.RUnlock()
		return fmt.Errorf("not connected to a room")
	}
	room := a.room
	a.mu.RUnlock()

	if err := room.LocalParticipant.PublishDataPacket(lksdk.UserData(data), lksdk.WithDataPublishReliable(true)); err != nil {
		return fmt.Errorf("failed to send error message: %w", err)
	}

	return nil
}

func (a *Agent) handleAcknowledgement(ack *protocol.Acknowledgement) {
	a.pendingMu.Lock()
	defer a.pendingMu.Unlock()

	// Remove from pending acknowledgements
	if pending, exists := a.pendingAcks[ack.AckedStanzaID]; exists {
		delete(a.pendingAcks, ack.AckedStanzaID)
		log.Printf("Received acknowledgement for stanzaId %d (sent at %v, acknowledged after %v)",
			ack.AckedStanzaID, pending.SentAt, time.Since(pending.SentAt))
	}
}

func (a *Agent) checkAckTimeouts() {
	defer a.wg.Done()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.ackCheckDone:
			return
		case <-a.ackCheckTicker.C:
			a.retryUnacknowledgedMessages()
		}
	}
}

func (a *Agent) retryUnacknowledgedMessages() {
	a.pendingMu.Lock()
	defer a.pendingMu.Unlock()

	now := time.Now()
	for stanzaID, pending := range a.pendingAcks {
		if now.Sub(pending.SentAt) > AckTimeout {
			if pending.RetryCount >= MaxRetries {
				// Max retries exceeded, give up
				log.Printf("WARNING: Message with stanzaId %d exceeded max retries, removing from pending", stanzaID)
				delete(a.pendingAcks, stanzaID)
			} else {
				// Retry sending the message
				log.Printf("Retrying message with stanzaId %d (attempt %d/%d)",
					stanzaID, pending.RetryCount+1, MaxRetries)

				a.mu.RLock()
				if a.connected && a.room != nil {
					// Keep lock held during PublishDataPacket to prevent room from becoming nil
					err := a.room.LocalParticipant.PublishDataPacket(lksdk.UserData(pending.Data), lksdk.WithDataPublishReliable(true))
					a.mu.RUnlock()
					if err != nil {
						log.Printf("ERROR: Failed to retry message with stanzaId %d: %v", stanzaID, err)
					} else {
						pending.RetryCount++
						pending.SentAt = now
					}
				} else {
					a.mu.RUnlock()
				}
			}
		}
	}
}

func (a *Agent) worker() {
	defer a.wg.Done()
	for {
		select {
		case <-a.ctx.Done():
			return
		case work, ok := <-a.workQueue:
			if !ok {
				// Work queue closed, exit worker
				return
			}
			// Execute the work with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("ERROR: Panic recovered in worker: %v", r)
					}
				}()
				work()
			}()
		}
	}
}

func (a *Agent) onDataReceived(data []byte, params lksdk.DataReceiveParams) {
	// Decode the message to check its type
	envelope, err := a.codec.Decode(data)
	if err != nil {
		log.Printf("ERROR: Failed to decode message: %v", err)
		return
	}

	// Handle acknowledgements separately
	if envelope.Type == protocol.TypeAcknowledgement {
		if ack, ok := envelope.Body.(*protocol.Acknowledgement); ok {
			a.handleAcknowledgement(ack)
		}
		// Don't pass acknowledgements to the callback
		return
	}

	// For other message types, send an acknowledgement if they have a positive stanzaId
	// (client messages have positive stanzaIds, server messages have negative)
	if envelope.StanzaID > 0 {
		conversationID := envelope.ConversationID
		stanzaID := envelope.StanzaID

		// Synchronization approach: We read workQueue under lock to ensure we either see
		// the queue before it's closed (and can safely send) or see nil (and drop the message).
		// This prevents the race where we read a non-nil queue, then Disconnect() closes it,
		// then we try to send on the closed channel.
		a.mu.RLock()
		workQueue := a.workQueue
		a.mu.RUnlock()

		if workQueue == nil {
			// Agent is disconnecting or disconnected, drop the message
			log.Printf("WARNING: Work queue closed, dropping acknowledgement for stanzaId %d", stanzaID)
			return
		}

		// Try to enqueue with timeout-based backpressure
		// Note: Even with the lock, there's a theoretical window where the channel could be closed
		// between our check and the send. We use a recover to handle sends on closed channels.
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was closed during send - this is expected during disconnect
					log.Printf("WARNING: Work queue closed during send, dropping acknowledgement for stanzaId %d", stanzaID)
				}
			}()

			select {
			case workQueue <- func() {
				select {
				case <-a.ctx.Done():
					return
				default:
					if err := a.SendAcknowledgement(a.ctx, conversationID, stanzaID); err != nil {
						log.Printf("ERROR: Failed to send acknowledgement for stanzaId %d: %v", stanzaID, err)
					}
				}
			}:
				// Successfully enqueued
			case <-time.After(WorkQueueTimeout):
				// Timeout - queue still full after waiting
				queueLen := len(workQueue)
				queueCap := cap(workQueue)
				log.Printf("WARNING: Work queue full after %v timeout (depth: %d/%d), rejecting acknowledgement for stanzaId %d",
					WorkQueueTimeout, queueLen, queueCap, stanzaID)

				// Send error message with backpressure indication
				if err := a.SendErrorMessage(a.ctx, conversationID, protocol.ErrCodeQueueOverflow,
					fmt.Sprintf("Server overloaded - message queue at capacity (%d/%d). Unable to process acknowledgement for message %d. Please slow down and retry.",
						queueLen, queueCap, stanzaID),
					protocol.SeverityWarning, true, stanzaID); err != nil {
					log.Printf("ERROR: Failed to send queue overflow error for stanzaId %d: %v", stanzaID, err)
				}
			}
		}()
	}

	// Pass the message to the callback
	msg := &ports.DataChannelMessage{
		Data:     data,
		SenderID: params.SenderIdentity,
		Topic:    params.Topic,
	}

	// Synchronization approach: Read workQueue under lock to ensure atomic visibility
	// of the disconnect state. See comment above for race prevention details.
	a.mu.RLock()
	workQueue := a.workQueue
	a.mu.RUnlock()

	if workQueue == nil {
		// Agent is disconnecting or disconnected, drop the message
		log.Printf("WARNING: Work queue closed, dropping message from %s (stanzaId: %d)", params.SenderIdentity, envelope.StanzaID)
		return
	}

	// Try to enqueue with timeout-based backpressure
	// Protect against sends on closed channel during concurrent disconnect
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was closed during send - this is expected during disconnect
				log.Printf("WARNING: Work queue closed during send, dropping message from %s (stanzaId: %d)", params.SenderIdentity, envelope.StanzaID)
			}
		}()

		select {
		case workQueue <- func() {
			select {
			case <-a.ctx.Done():
				return
			default:
				if err := a.callbacks.OnDataReceived(a.ctx, msg); err != nil {
					log.Printf("ERROR: OnDataReceived callback failed: %v", err)
				}
			}
		}:
			// Successfully enqueued
		case <-time.After(WorkQueueTimeout):
			// Timeout - queue still full after waiting
			queueLen := len(workQueue)
			queueCap := cap(workQueue)
			log.Printf("WARNING: Work queue full after %v timeout (depth: %d/%d), rejecting message from %s (stanzaId: %d)",
				WorkQueueTimeout, queueLen, queueCap, params.SenderIdentity, envelope.StanzaID)

			// Send error message with backpressure indication
			if err := a.SendErrorMessage(a.ctx, envelope.ConversationID, protocol.ErrCodeQueueOverflow,
				fmt.Sprintf("Server overloaded - message queue at capacity (%d/%d). Unable to process message %d. Please slow down and retry.",
					queueLen, queueCap, envelope.StanzaID),
				protocol.SeverityWarning, true, envelope.StanzaID); err != nil {
				log.Printf("ERROR: Failed to send queue overflow error for message from %s: %v", params.SenderIdentity, err)
			}
		}
	}()
}

func (a *Agent) onTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	if track.Kind() != webrtc.RTPCodecTypeAudio {
		return
	}

	a.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ERROR: Panic recovered in OnAudioReceived callback goroutine: %v", r)
			}
		}()
		defer a.wg.Done()

		// Create an AudioConverter for decoding Opus to float32 for VAD processing
		// Using LiveKit's sample rate (48kHz) and channels (stereo)
		var vadDecoder *AudioConverter
		a.mu.RLock()
		hasVAD := a.vadProcessor != nil
		a.mu.RUnlock()

		if hasVAD {
			var err error
			vadDecoder, err = NewAudioConverter(LiveKitSampleRate, LiveKitChannels)
			if err != nil {
				log.Printf("ERROR: Failed to create audio converter for VAD: %v", err)
				// Continue without VAD - audio callbacks will still work
			}
		}

		for {
			select {
			case <-a.ctx.Done():
				return
			default:
				rtp, _, err := track.ReadRTP()
				if err != nil {
					return
				}

				frame := &ports.AudioFrame{
					Data:       rtp.Payload,
					SampleRate: int(track.Codec().ClockRate),
					Channels:   int(track.Codec().Channels),
					TrackSID:   publication.SID(),
				}

				// Feed decoded audio to VAD processor if available
				if vadDecoder != nil {
					a.mu.RLock()
					vadProcessor := a.vadProcessor
					a.mu.RUnlock()

					if vadProcessor != nil {
						// Decode Opus to float32 for VAD
						samples, err := vadDecoder.ConvertOpusToPCMFloat(rtp.Payload)
						if err != nil {
							log.Printf("WARNING: Failed to decode audio for VAD: %v", err)
						} else {
							// Process through VAD
							if err := vadProcessor.ProcessAudio(samples); err != nil {
								log.Printf("WARNING: VAD processing failed: %v", err)
							}
						}
					}
				}

				if err := a.callbacks.OnAudioReceived(a.ctx, frame); err != nil {
					log.Printf("ERROR: OnAudioReceived callback failed: %v", err)
					return
				}
			}
		}
	}()
}

func (a *Agent) onParticipantConnected(rp *lksdk.RemoteParticipant) {
	participant := &ports.LiveKitParticipant{
		ID:       rp.SID(),
		Identity: rp.Identity(),
		Name:     rp.Name(),
	}

	a.mu.Lock()
	if a.roomInfo != nil {
		a.roomInfo.Participants = append(a.roomInfo.Participants, participant)
	}
	// Synchronization approach: Read workQueue under lock to ensure atomic visibility
	// of the disconnect state. See comment in onDataReceived for race prevention details.
	workQueue := a.workQueue
	a.mu.Unlock()

	if workQueue == nil {
		// Agent is disconnecting or disconnected, drop the callback
		log.Printf("WARNING: Work queue closed, dropping OnParticipantConnected callback for participant %s", participant.Identity)
		return
	}

	// Try to enqueue with timeout-based backpressure
	// Protect against sends on closed channel during concurrent disconnect
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was closed during send - this is expected during disconnect
				log.Printf("WARNING: Work queue closed during send, dropping OnParticipantConnected callback for participant %s", participant.Identity)
			}
		}()

		select {
		case workQueue <- func() {
			select {
			case <-a.ctx.Done():
				return
			default:
				if err := a.callbacks.OnParticipantConnected(a.ctx, participant); err != nil {
					log.Printf("ERROR: OnParticipantConnected callback failed: %v", err)
				}
			}
		}:
			// Successfully enqueued
		case <-time.After(WorkQueueTimeout):
			// Timeout - queue still full after waiting
			queueLen := len(workQueue)
			queueCap := cap(workQueue)
			log.Printf("WARNING: Work queue full after %v timeout (depth: %d/%d), dropping OnParticipantConnected callback for participant %s",
				WorkQueueTimeout, queueLen, queueCap, participant.Identity)
		}
	}()
}

func (a *Agent) onParticipantDisconnected(rp *lksdk.RemoteParticipant) {
	participant := &ports.LiveKitParticipant{
		ID:       rp.SID(),
		Identity: rp.Identity(),
		Name:     rp.Name(),
	}

	a.mu.Lock()
	if a.roomInfo != nil {
		for i, p := range a.roomInfo.Participants {
			if p.ID == participant.ID {
				a.roomInfo.Participants = append(a.roomInfo.Participants[:i], a.roomInfo.Participants[i+1:]...)
				break
			}
		}
	}
	// Synchronization approach: Read workQueue under lock to ensure atomic visibility
	// of the disconnect state. See comment in onDataReceived for race prevention details.
	workQueue := a.workQueue
	a.mu.Unlock()

	if workQueue == nil {
		// Agent is disconnecting or disconnected, drop the callback
		log.Printf("WARNING: Work queue closed, dropping OnParticipantDisconnected callback for participant %s", participant.Identity)
		return
	}

	// Try to enqueue with timeout-based backpressure
	// Protect against sends on closed channel during concurrent disconnect
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was closed during send - this is expected during disconnect
				log.Printf("WARNING: Work queue closed during send, dropping OnParticipantDisconnected callback for participant %s", participant.Identity)
			}
		}()

		select {
		case workQueue <- func() {
			select {
			case <-a.ctx.Done():
				return
			default:
				if err := a.callbacks.OnParticipantDisconnected(a.ctx, participant); err != nil {
					log.Printf("ERROR: OnParticipantDisconnected callback failed: %v", err)
				}
			}
		}:
			// Successfully enqueued
		case <-time.After(WorkQueueTimeout):
			// Timeout - queue still full after waiting
			queueLen := len(workQueue)
			queueCap := cap(workQueue)
			log.Printf("WARNING: Work queue full after %v timeout (depth: %d/%d), dropping OnParticipantDisconnected callback for participant %s",
				WorkQueueTimeout, queueLen, queueCap, participant.Identity)
		}
	}()
}
