package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

type LiveKitClient struct {
	cfg      *Config
	room     *lksdk.Room
	roomName string

	audioTrack  *lksdk.LocalSampleTrack
	opusEncoder *opus.Encoder

	audioBuffer   []byte
	audioBufferMu sync.Mutex
	lastAudioTime time.Time
	isSpeaking    bool
	speakingMu    sync.Mutex

	onUtterance func(audio []byte)
	onJoin      func(identity string)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	connected bool
	mu        sync.RWMutex
}

func NewLiveKitClient(cfg *Config) (*LiveKitClient, error) {
	enc, err := opus.NewEncoder(cfg.TTSSampleRate, cfg.Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %w", err)
	}

	return &LiveKitClient{
		cfg:         cfg,
		opusEncoder: enc,
		audioBuffer: make([]byte, 0, cfg.SampleRate*2*5), // 5 seconds buffer at capture rate
	}, nil
}

func (c *LiveKitClient) SetCallbacks(onUtterance func([]byte), onJoin func(string)) {
	c.onUtterance = onUtterance
	c.onJoin = onJoin
}

func (c *LiveKitClient) Connect(ctx context.Context, roomName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		slog.Info("livekit: already connected", "room", c.roomName)
		return nil
	}

	c.roomName = roomName
	c.ctx, c.cancel = context.WithCancel(ctx)
	slog.Info("livekit: connecting to room", "room", roomName, "url", c.cfg.LiveKitURL)

	c.room = lksdk.NewRoom(&lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackSubscribed:   c.onTrackSubscribed,
			OnTrackUnsubscribed: c.onTrackUnsubscribed,
		},
		OnParticipantConnected:    c.onParticipantConnected,
		OnParticipantDisconnected: c.onParticipantDisconnected,
		OnDisconnected:            c.onDisconnected,
	})

	connectInfo := lksdk.ConnectInfo{
		APIKey:              c.cfg.LiveKitAPIKey,
		APISecret:           c.cfg.LiveKitAPISecret,
		RoomName:            roomName,
		ParticipantIdentity: "voice-helper",
		ParticipantName:     "Alicia Voice",
	}
	slog.Info("livekit: joining room", "room", roomName)
	if err := c.room.Join(c.cfg.LiveKitURL, connectInfo, lksdk.WithAutoSubscribe(true)); err != nil {
		slog.Error("livekit: failed to join room", "room", roomName, "error", err)
		return err
	}
	slog.Info("livekit: joined room", "room", roomName)

	slog.Info("livekit: creating audio track", "sample_rate", c.cfg.TTSSampleRate, "channels", c.cfg.Channels)
	var err error
	c.audioTrack, err = lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeOpus,
		ClockRate: uint32(c.cfg.TTSSampleRate),
		Channels:  uint16(c.cfg.Channels),
	})
	if err != nil {
		slog.Error("livekit: failed to create audio track", "error", err)
	} else {
		slog.Info("livekit: publishing audio track")
		_, err = c.room.LocalParticipant.PublishTrack(c.audioTrack, &lksdk.TrackPublicationOptions{
			Name:   "voice-assistant",
			Source: livekit.TrackSource_MICROPHONE,
		})
		if err != nil {
			slog.Error("livekit: failed to publish audio track", "error", err)
		} else {
			slog.Info("livekit: audio track published")
		}
	}

	c.connected = true
	slog.Info("livekit: connected", "room", roomName, "participants", len(c.room.GetRemoteParticipants()))

	return nil
}

func (c *LiveKitClient) Disconnect() {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return
	}

	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()
	c.wg.Wait()

	c.mu.Lock()
	if c.room != nil {
		c.room.Disconnect()
		c.room = nil
	}

	c.connected = false
	c.mu.Unlock()
	slog.Info("livekit: disconnected", "room", c.roomName)
}

func (c *LiveKitClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *LiveKitClient) PlayAudio(samples []byte) error {
	c.mu.RLock()
	track := c.audioTrack
	encoder := c.opusEncoder
	c.mu.RUnlock()

	if track == nil || encoder == nil {
		return nil
	}

	if len(samples)%2 != 0 {
		samples = samples[:len(samples)-1]
	}
	if len(samples) == 0 {
		return nil
	}

	numSamples := len(samples) / 2
	audioDurationMs := numSamples * 1000 / c.cfg.TTSSampleRate
	slog.Debug("livekit: playing audio", "bytes", len(samples), "duration_ms", audioDurationMs, "room", c.roomName)

	pcm := make([]int16, numSamples)
	for i := 0; i < numSamples; i++ {
		pcm[i] = int16(binary.LittleEndian.Uint16(samples[i*2:]))
	}

	frameSize := c.cfg.TTSSampleRate * 20 / 1000 // 20ms frame
	opusBuffer := make([]byte, 4096)

	for offset := 0; offset+frameSize <= numSamples; offset += frameSize {
		frame := pcm[offset : offset+frameSize]

		n, err := encoder.Encode(frame, opusBuffer)
		if err != nil {
			slog.Error("livekit: opus encode error", "error", err)
			continue
		}

		data := make([]byte, n)
		copy(data, opusBuffer[:n])

		if err := track.WriteSample(media.Sample{
			Data:     data,
			Duration: 20 * time.Millisecond,
		}, nil); err != nil {
			return err
		}
	}

	slog.Debug("livekit: finished playing audio", "duration_ms", audioDurationMs, "room", c.roomName)
	return nil
}

func (c *LiveKitClient) onTrackSubscribed(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, participant *lksdk.RemoteParticipant) {
	slog.Info("livekit: track subscribed", "kind", track.Kind().String(), "codec", track.Codec().MimeType, "participant", participant.Identity(), "track_id", track.ID())

	if track.Kind() != webrtc.RTPCodecTypeAudio {
		slog.Info("livekit: ignoring non-audio track", "participant", participant.Identity())
		return
	}

	slog.Debug("livekit: starting audio reader", "participant", participant.Identity())
	c.wg.Add(1)
	go c.readAudioTrack(track, participant.Identity())
}

func (c *LiveKitClient) onTrackUnsubscribed(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, participant *lksdk.RemoteParticipant) {
	slog.Info("livekit: track unsubscribed", "kind", track.Kind().String(), "participant", participant.Identity(), "track_id", track.ID())
}

func (c *LiveKitClient) onParticipantConnected(participant *lksdk.RemoteParticipant) {
	slog.Info("livekit: participant connected", "identity", participant.Identity(), "name", participant.Name(), "sid", participant.SID())

	for _, track := range participant.TrackPublications() {
		slog.Debug("livekit: participant track", "track_name", track.Name(), "kind", track.Kind().String(), "source", track.Source().String(), "subscribed", track.IsSubscribed())
	}

	if c.onJoin != nil {
		c.onJoin(participant.Identity())
	}
}

func (c *LiveKitClient) onParticipantDisconnected(participant *lksdk.RemoteParticipant) {
	slog.Info("livekit: participant disconnected", "identity", participant.Identity(), "sid", participant.SID())
}

func (c *LiveKitClient) onDisconnected() {
	c.mu.Lock()
	c.connected = false
	roomName := c.roomName
	c.mu.Unlock()
	slog.Warn("livekit: room disconnected", "room", roomName)
}

func (c *LiveKitClient) readAudioTrack(track *webrtc.TrackRemote, identity string) {
	defer c.wg.Done()
	decoder, err := opus.NewDecoder(c.cfg.SampleRate, c.cfg.Channels)
	if err != nil {
		slog.Error("livekit: failed to create opus decoder", "error", err)
		return
	}

	rtpBuf := make([]byte, 4096)
	// Max Opus frame is 120ms at 48kHz = 5760 samples per channel
	pcmBuf := make([]int16, 5760*c.cfg.Channels)

	var totalBytesRead int64
	var packetCount int64
	slog.Debug("livekit: started reading audio", "participant", identity, "track_id", track.ID())

	for {
		select {
		case <-c.ctx.Done():
			slog.Info("livekit: audio reader stopped", "participant", identity, "packets", packetCount, "total_bytes", totalBytesRead)
			return
		default:
		}

		n, _, err := track.Read(rtpBuf)
		if err != nil {
			slog.Error("livekit: audio read error", "participant", identity, "packets", packetCount, "total_bytes", totalBytesRead, "error", err)
			return
		}

		if n == 0 {
			continue
		}

		// RTP header is 12 bytes
		if n <= 12 {
			continue
		}
		opusData := rtpBuf[12:n]

		numSamples, err := decoder.Decode(opusData, pcmBuf)
		if err != nil {
			slog.Error("livekit: opus decode error", "error", err)
			continue
		}

		if numSamples == 0 {
			continue
		}

		pcmBytes := make([]byte, numSamples*2*c.cfg.Channels)
		for i := 0; i < numSamples*c.cfg.Channels; i++ {
			binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(pcmBuf[i]))
		}

		totalBytesRead += int64(len(pcmBytes))
		packetCount++

		c.processAudioData(pcmBytes, identity)
	}
}

func (c *LiveKitClient) processAudioData(data []byte, identity string) {
	// Skip our own audio
	if identity == "voice-helper" {
		return
	}

	energy := c.calculateEnergy(data)
	isSpeaking := energy > c.cfg.VADThreshold

	c.speakingMu.Lock()
	wasSpeaking := c.isSpeaking
	if isSpeaking {
		c.isSpeaking = true
	}
	// Don't set c.isSpeaking = false here; it stays true until the buffer is flushed.
	c.speakingMu.Unlock()

	if isSpeaking && !wasSpeaking {
		slog.Info("livekit: speech started", "participant", identity, "energy", energy, "threshold", c.cfg.VADThreshold)
	}

	if isSpeaking {
		c.audioBufferMu.Lock()
		c.audioBuffer = append(c.audioBuffer, data...)
		c.lastAudioTime = time.Now()
		c.audioBufferMu.Unlock()
	} else if wasSpeaking {
		// Speech has stopped but buffer hasn't been flushed yet.
		// Re-check silence timer on every non-speaking packet until dispatch.
		c.audioBufferMu.Lock()
		timeSinceAudio := time.Since(c.lastAudioTime)
		if timeSinceAudio >= c.cfg.SilenceDuration && len(c.audioBuffer) > 0 {
			audio := make([]byte, len(c.audioBuffer))
			copy(audio, c.audioBuffer)
			c.audioBuffer = c.audioBuffer[:0]

			c.audioBufferMu.Unlock()

			// Buffer dispatched; now clear the speaking flag.
			c.speakingMu.Lock()
			c.isSpeaking = false
			c.speakingMu.Unlock()

			slog.Info("livekit: speech ended", "participant", identity, "bytes", len(audio), "duration_ms", len(audio)/(c.cfg.SampleRate*c.cfg.Channels*2/1000))

			if c.onUtterance != nil {
				go c.onUtterance(audio)
			}
		} else if len(c.audioBuffer) == 0 {
			c.audioBufferMu.Unlock()

			// No audio in buffer; clear the speaking flag.
			c.speakingMu.Lock()
			c.isSpeaking = false
			c.speakingMu.Unlock()
		} else {
			c.audioBufferMu.Unlock()
		}
	}
}

func (c *LiveKitClient) calculateEnergy(data []byte) float64 {
	if len(data) < 2 {
		return 0
	}

	var sum float64
	numSamples := len(data) / 2

	for i := 0; i < numSamples; i++ {
		sample := int16(binary.LittleEndian.Uint16(data[i*2:]))
		normalized := float64(sample) / 32768.0
		sum += normalized * normalized
	}

	return math.Sqrt(sum / float64(numSamples))
}

func (c *LiveKitClient) GetParticipantCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.room == nil {
		return 0
	}

	return len(c.room.GetRemoteParticipants())
}
