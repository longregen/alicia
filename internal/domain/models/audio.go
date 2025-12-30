package models

import (
	"time"
)

type TranscriptionMeta struct {
	Model      string    `json:"model,omitempty"`
	Language   string    `json:"language,omitempty"`
	Confidence float32   `json:"confidence,omitempty"`
	Duration   float32   `json:"duration,omitempty"`
	Segments   []Segment `json:"segments,omitempty"`
}

type Segment struct {
	ID         int     `json:"id"`
	Start      float32 `json:"start"`
	End        float32 `json:"end"`
	Text       string  `json:"text"`
	Confidence float32 `json:"confidence,omitempty"`
}

type Audio struct {
	ID                string             `json:"id"`
	MessageID         string             `json:"message_id,omitempty"`
	AudioType         AudioType          `json:"audio_type"`
	AudioFormat       string             `json:"audio_format"`
	AudioData         []byte             `json:"audio_data,omitempty"`
	DurationMs        int                `json:"duration_ms,omitempty"`
	Transcription     string             `json:"transcription,omitempty"`
	LiveKitTrackSID   string             `json:"livekit_track_sid,omitempty"`
	TranscriptionMeta *TranscriptionMeta `json:"transcription_meta,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	DeletedAt         *time.Time         `json:"deleted_at,omitempty"`
}

func NewAudio(id string, audioType AudioType, format string) *Audio {
	now := time.Now()
	return &Audio{
		ID:          id,
		AudioType:   audioType,
		AudioFormat: format,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewInputAudio creates a new input audio record (user speech)
func NewInputAudio(id, format string) *Audio {
	return NewAudio(id, AudioTypeInput, format)
}

// NewOutputAudio creates a new output audio record (TTS)
func NewOutputAudio(id, format string) *Audio {
	return NewAudio(id, AudioTypeOutput, format)
}

func (a *Audio) SetData(data []byte, durationMs int) {
	a.AudioData = data
	a.DurationMs = durationMs
	a.UpdatedAt = time.Now()
}

func (a *Audio) SetTranscription(text string) {
	a.Transcription = text
	a.UpdatedAt = time.Now()
}

func (a *Audio) SetTranscriptionWithMeta(text string, meta *TranscriptionMeta) {
	a.Transcription = text
	a.TranscriptionMeta = meta
	a.UpdatedAt = time.Now()
}

// SetLiveKitTrack associates a LiveKit track with the audio
func (a *Audio) SetLiveKitTrack(trackSID string) {
	a.LiveKitTrackSID = trackSID
	a.UpdatedAt = time.Now()
}
