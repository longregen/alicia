package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/longregen/alicia/shared/protocol"
	"github.com/longregen/alicia/pkg/otel"
	"github.com/longregen/alicia/shared/backoff"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ttsItem struct {
	spanCtx   trace.SpanContext
	text      string
	messageID string
	sequence  int
}

type VoiceSession struct {
	ConversationID string
	UserID         string

	cfg *Config
	lk  *LiveKitClient
	asr *ASRClient
	tts *TTSClient
	ws  *WSClient

	ttsQueue     chan ttsItem
	isSpeaking   bool
	speakingMu   sync.RWMutex
	currentMsgID string
	voiceSpeed   float64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type SessionManager struct {
	cfg      *Config
	sessions map[string]*VoiceSession
	mu       sync.RWMutex

	wsClient   *WSClient
	asr        *ASRClient
	tts        *TTSClient
	prefsStore *VoicePreferencesStore

	ctx    context.Context
	cancel context.CancelFunc
}


func NewSessionManager(cfg *Config) *SessionManager {
	return &SessionManager{
		cfg:        cfg,
		sessions:   make(map[string]*VoiceSession),
		asr:        NewASRClient(cfg),
		tts:        NewTTSClient(cfg),
		prefsStore: NewVoicePreferencesStore(),
	}
}

func (m *SessionManager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wsClient = NewWSClient(m.cfg)
	m.wsClient.SetCallbacks(
		m.onSentence,
		m.onGenerationStart,
		m.onVoiceJoinRequest,
		m.onVoiceLeaveRequest,
	)
	m.wsClient.SetPreferencesCallback(m.onPreferencesUpdate)

	if err := m.connectWithBackoff(); err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}

	go m.monitorWebSocket()
	go m.monitorSessions()

	slog.Info("session manager started")
	return nil
}

func (m *SessionManager) connectWithBackoff() error {
	return backoff.RetryWithCallback(m.ctx, backoff.Standard, func(ctx context.Context, attempt int) error {
		return m.wsClient.Connect(ctx)
	}, func(attempt int, err error, delay time.Duration) {
		slog.Warn("session manager connection attempt failed", "attempt", attempt, "error", err, "retry_in", delay)
	})
}

func (m *SessionManager) Stop() {
	m.cancel()

	m.mu.Lock()
	for _, session := range m.sessions {
		session.Stop()
	}
	m.sessions = make(map[string]*VoiceSession)
	m.mu.Unlock()

	if m.wsClient != nil {
		m.wsClient.Disconnect()
	}
}

func (m *SessionManager) monitorWebSocket() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if !m.wsClient.IsConnected() {
				slog.Warn("session manager websocket disconnected, reconnecting")
				if err := m.wsClient.Reconnect(m.ctx); err != nil {
					slog.Error("session manager reconnect failed", "error", err)
				}
			}
		}
	}
}

func (m *SessionManager) monitorSessions() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			slog.Info("session manager monitor stopped")
			return
		case <-ticker.C:
			m.cleanupSessions()
		}
	}
}

func (m *SessionManager) cleanupSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for convID, session := range m.sessions {
		isConnected := session.lk.IsConnected()
		participantCount := session.lk.GetParticipantCount()

		if !isConnected {
			slog.Info("session manager cleaning up disconnected session", "conversation_id", convID)
			session.Stop()
			delete(m.sessions, convID)
		} else if participantCount == 0 {
			slog.Info("session manager leaving empty room", "conversation_id", convID)
			session.Stop()
			delete(m.sessions, convID)
		}
	}
}

func (m *SessionManager) onVoiceJoinRequest(req *protocol.VoiceJoinRequest) {
	userID := req.UserID
	if userID == "" {
		userID = "voice-user"
	}
	slog.Info("voice join", "conversation_id", req.ConversationID, "user_id", userID)

	_, err := m.JoinRoom(req.ConversationID, userID)
	if err != nil {
		slog.Error("failed to join room", "conversation_id", req.ConversationID, "error", err)
		if err := m.wsClient.SendVoiceJoinAck(req.ConversationID, false, err.Error(), 0); err != nil {
			slog.Error("failed to send join ack", "error", err)
		}
		return
	}

	if err := m.wsClient.SendVoiceJoinAck(req.ConversationID, true, "", m.cfg.TTSSampleRate); err != nil {
		slog.Error("failed to send join ack", "error", err)
	}
}

func (m *SessionManager) onVoiceLeaveRequest(req *protocol.VoiceLeaveRequest) {
	slog.Info("voice leave", "conversation_id", req.ConversationID)
	m.LeaveRoom(req.ConversationID)
	if err := m.wsClient.SendVoiceLeaveAck(req.ConversationID, true, ""); err != nil {
		slog.Error("failed to send leave ack", "error", err)
	}
}

func (m *SessionManager) JoinRoom(convID, userID string) (*VoiceSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[convID]; ok {
		return session, nil
	}

	session, err := m.createSession(convID, userID)
	if err != nil {
		return nil, err
	}

	m.sessions[convID] = session
	return session, nil
}

func (m *SessionManager) LeaveRoom(convID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[convID]; ok {
		session.Stop()
		delete(m.sessions, convID)
	}
}

func (m *SessionManager) createSession(convID, userID string) (*VoiceSession, error) {
	ctx, cancel := context.WithCancel(m.ctx)

	lk, err := NewLiveKitClient(m.cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create livekit client: %w", err)
	}

	prefs := m.prefsStore.Get(userID)

	session := &VoiceSession{
		ConversationID: convID,
		UserID:         userID,
		cfg:            m.cfg,
		lk:             lk,
		asr:            m.asr,
		tts:            m.tts,
		ws:             m.wsClient,
		ttsQueue:       make(chan ttsItem, 100),
		voiceSpeed:     prefs.Speed,
		ctx:            ctx,
		cancel:         cancel,
	}

	session.lk.SetCallbacks(
		func(audio []byte) {
			session.wg.Add(1)
			defer session.wg.Done()
			session.onUtterance(audio)
		},
		session.onUserJoin,
	)

	if err := session.lk.Connect(ctx, convID); err != nil {
		cancel()
		return nil, fmt.Errorf("connect to livekit: %w", err)
	}

	if err := m.wsClient.Subscribe(convID); err != nil {
		session.lk.Disconnect()
		cancel()
		return nil, fmt.Errorf("subscribe to conversation: %w", err)
	}

	session.startTTSWorker()

	slog.Info("session manager created session", "conversation_id", convID)
	return session, nil
}

func (m *SessionManager) onSentence(ctx context.Context, convID string, sentence *protocol.AssistantSentence) {
	m.mu.RLock()
	session, ok := m.sessions[convID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	session.handleSentence(ctx, sentence)
}

func (m *SessionManager) onGenerationStart(ctx context.Context, convID string, start *protocol.StartAnswer) {
	m.mu.RLock()
	session, ok := m.sessions[convID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	session.handleGenerationStart(ctx, start)
}

func (m *SessionManager) onPreferencesUpdate(update *protocol.PreferencesUpdate) {
	m.prefsStore.Update(*update)

	// Update speed on all sessions belonging to this user
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefs := m.prefsStore.Get(update.UserID)
	for _, session := range m.sessions {
		if session.UserID == update.UserID {
			session.speakingMu.Lock()
			session.voiceSpeed = prefs.Speed
			session.speakingMu.Unlock()
		}
	}
}

func (s *VoiceSession) Stop() {
	s.cancel()
	s.ws.Unsubscribe(s.ConversationID)
	s.lk.Disconnect()
	s.wg.Wait()
}

func (s *VoiceSession) onUtterance(audio []byte) {
	bytesPerMs := s.cfg.SampleRate * s.cfg.Channels * 2 / 1000
	if bytesPerMs == 0 {
		bytesPerMs = 1
	}
	audioDurationMs := len(audio) / bytesPerMs
	slog.Debug("session: processing utterance", "bytes", len(audio), "duration_ms", audioDurationMs, "conversation_id", s.ConversationID)

	ctx, span := otel.Tracer("alicia-voice").Start(s.ctx, "voice.user_turn",
		trace.WithAttributes(
			attribute.String("conversation.id", s.ConversationID),
			attribute.String("user.id", s.UserID),
			attribute.Int("audio.bytes", len(audio)),
		))
	defer span.End()

	slog.Debug("session: sending audio to asr", "bytes", len(audio))
	text, err := s.asr.Transcribe(ctx, audio)
	if err != nil {
		slog.Error("session: asr error", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "ASR transcription failed")
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		span.SetAttributes(attribute.Bool("transcription.empty", true))
		span.SetStatus(codes.Ok, "empty transcription")
		return
	}

	slog.Debug("session: transcribed", "chars", len(text), "preview", truncateString(text, 50))
	span.SetAttributes(
		attribute.Int("transcription.length", len(text)),
		attribute.String("transcription.preview", truncateString(text, 100)),
	)

	if err := s.ws.SendUserMessage(s.ConversationID, s.UserID, text); err != nil {
		slog.Error("session: failed to send user message", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send user message")
		return
	}

	span.SetStatus(codes.Ok, "user turn completed")
}

func (s *VoiceSession) onUserJoin(identity string) {
	slog.Info("voice session user joined", "identity", identity, "conversation_id", s.ConversationID)
}

func (s *VoiceSession) handleSentence(ctx context.Context, sentence *protocol.AssistantSentence) {
	text := strings.TrimSpace(sentence.Text)
	if text == "" {
		return
	}

	item := ttsItem{spanCtx: trace.SpanContextFromContext(ctx), text: text, messageID: sentence.MessageID, sequence: sentence.Sequence}

	select {
	case s.ttsQueue <- item:
	default:
		// Queue is full; attempt to drain stale items from older messages
		slog.Warn("session: tts queue full, attempting drain", "message_id", sentence.MessageID, "queue_cap", cap(s.ttsQueue))

		s.speakingMu.RLock()
		currentMsgID := s.currentMsgID
		s.speakingMu.RUnlock()

		drained := s.drainStaleItems(currentMsgID)
		slog.Debug("session: drained stale tts items", "drained", drained)

		select {
		case s.ttsQueue <- item:
			s.ws.SendVoiceStatus(s.ConversationID, &protocol.VoiceStatus{
				ConversationID: s.ConversationID,
				Status:         "queue_ok",
				QueueLength:    len(s.ttsQueue),
			})
		default:
			slog.Warn("session: tts queue full, dropping sentence", "message_id", sentence.MessageID, "sequence", sentence.Sequence)
			s.ws.SendVoiceStatus(s.ConversationID, &protocol.VoiceStatus{
				ConversationID: s.ConversationID,
				Status:         "queue_full",
				QueueLength:    len(s.ttsQueue),
				Error:          fmt.Sprintf("TTS queue full, dropped sentence %d", sentence.Sequence),
			})
		}
	}
}

func (s *VoiceSession) drainStaleItems(currentMsgID string) int {
	drained := 0
	queueLen := len(s.ttsQueue)

	for i := 0; i < queueLen; i++ {
		select {
		case existing := <-s.ttsQueue:
			if existing.messageID != currentMsgID {
				drained++
				continue
			}
			select {
			case s.ttsQueue <- existing:
			default:
				drained++
			}
		default:
			return drained
		}
	}

	return drained
}

func (s *VoiceSession) handleGenerationStart(ctx context.Context, start *protocol.StartAnswer) {
	s.speakingMu.Lock()
	s.currentMsgID = start.MessageID
	s.speakingMu.Unlock()
}

func (s *VoiceSession) startTTSWorker() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.ttsWorker()
	}()
}

func (s *VoiceSession) ttsWorker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case item := <-s.ttsQueue:
			s.speakingMu.RLock()
			currentMsgID := s.currentMsgID
			s.speakingMu.RUnlock()

			if item.messageID != currentMsgID {
				continue
			}

			s.ws.SendVoiceSpeaking(s.ConversationID, &protocol.VoiceSpeaking{
				ConversationID: s.ConversationID,
				MessageID:      item.messageID,
				Speaking:       true,
				SentenceSeq:    item.sequence,
			})

			itemCtx := trace.ContextWithSpanContext(s.ctx, item.spanCtx)
			s.speakText(itemCtx, item.text)

			s.ws.SendVoiceSpeaking(s.ConversationID, &protocol.VoiceSpeaking{
				ConversationID: s.ConversationID,
				MessageID:      item.messageID,
				Speaking:       false,
				SentenceSeq:    item.sequence,
			})
		}
	}
}

func (s *VoiceSession) speakText(parentCtx context.Context, text string) {
	s.speakingMu.Lock()
	s.isSpeaking = true
	speed := s.voiceSpeed
	s.speakingMu.Unlock()

	defer func() {
		s.speakingMu.Lock()
		s.isSpeaking = false
		s.speakingMu.Unlock()
	}()

	spanCtx := parentCtx
	if spanCtx == nil {
		spanCtx = s.ctx
	}

	ctx, span := otel.Tracer("alicia-voice").Start(spanCtx, "voice.assistant_speak",
		trace.WithAttributes(
			attribute.String("conversation.id", s.ConversationID),
			attribute.Int("text.length", len(text)),
			attribute.String("text.preview", truncateString(text, 100)),
			attribute.Float64("tts.speed", speed),
		))
	defer span.End()

	audio, err := s.tts.Synthesize(ctx, text, speed)
	if err != nil {
		slog.Error("session: tts synthesis error", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "TTS synthesis failed")
		return
	}

	if len(audio) == 0 {
		slog.Warn("session: tts returned empty audio", "preview", truncateString(text, 50))
		span.SetAttributes(attribute.Bool("audio.empty", true))
		span.SetStatus(codes.Ok, "no audio generated")
		return
	}

	span.SetAttributes(attribute.Int("audio.bytes", len(audio)))

	if err := s.lk.PlayAudio(audio); err != nil {
		slog.Error("session: play audio error", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "audio playback failed")
		return
	}

	span.SetStatus(codes.Ok, "speech completed")
}

