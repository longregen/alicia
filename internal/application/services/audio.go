package services

import (
	"context"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type AudioService struct {
	audioRepo   ports.AudioRepository
	messageRepo ports.MessageRepository
	asrService  ports.ASRService
	ttsService  ports.TTSService
	idGenerator ports.IDGenerator
	txManager   ports.TransactionManager
}

func NewAudioService(
	audioRepo ports.AudioRepository,
	messageRepo ports.MessageRepository,
	asrService ports.ASRService,
	ttsService ports.TTSService,
	idGenerator ports.IDGenerator,
	txManager ports.TransactionManager,
) *AudioService {
	return &AudioService{
		audioRepo:   audioRepo,
		messageRepo: messageRepo,
		asrService:  asrService,
		ttsService:  ttsService,
		idGenerator: idGenerator,
		txManager:   txManager,
	}
}

func (s *AudioService) CreateInputAudio(ctx context.Context, format string, data []byte, durationMs int) (*models.Audio, error) {
	if format == "" {
		return nil, domain.NewDomainError(domain.ErrAudioFormatUnsupported, "audio format cannot be empty")
	}

	if len(data) == 0 {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "audio data cannot be empty")
	}

	id := s.idGenerator.GenerateAudioID()
	audio := models.NewInputAudio(id, format)
	audio.SetData(data, durationMs)

	if err := s.audioRepo.Create(ctx, audio); err != nil {
		return nil, domain.NewDomainError(err, "failed to create input audio")
	}

	return audio, nil
}

func (s *AudioService) CreateOutputAudio(ctx context.Context, format string, data []byte, durationMs int) (*models.Audio, error) {
	if format == "" {
		return nil, domain.NewDomainError(domain.ErrAudioFormatUnsupported, "audio format cannot be empty")
	}

	if len(data) == 0 {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "audio data cannot be empty")
	}

	id := s.idGenerator.GenerateAudioID()
	audio := models.NewOutputAudio(id, format)
	audio.SetData(data, durationMs)

	if err := s.audioRepo.Create(ctx, audio); err != nil {
		return nil, domain.NewDomainError(err, "failed to create output audio")
	}

	return audio, nil
}

func (s *AudioService) GetByID(ctx context.Context, id string) (*models.Audio, error) {
	if id == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "audio ID cannot be empty")
	}

	audio, err := s.audioRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio not found")
	}

	if audio.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio has been deleted")
	}

	return audio, nil
}

func (s *AudioService) GetByMessage(ctx context.Context, messageID string) (*models.Audio, error) {
	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	audio, err := s.audioRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio not found for message")
	}

	if audio.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio has been deleted")
	}

	return audio, nil
}

func (s *AudioService) GetByLiveKitTrack(ctx context.Context, trackSID string) (*models.Audio, error) {
	if trackSID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "track SID cannot be empty")
	}

	audio, err := s.audioRepo.GetByLiveKitTrack(ctx, trackSID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio not found for track")
	}

	if audio.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrAudioNotFound, "audio has been deleted")
	}

	return audio, nil
}

func (s *AudioService) Update(ctx context.Context, audio *models.Audio) error {
	if audio == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "audio cannot be nil")
	}

	if audio.ID == "" {
		return domain.NewDomainError(domain.ErrInvalidID, "audio ID cannot be empty")
	}

	// Verify audio exists
	existing, err := s.audioRepo.GetByID(ctx, audio.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrAudioNotFound, "audio not found")
	}

	if existing.DeletedAt != nil {
		return domain.NewDomainError(domain.ErrAudioNotFound, "cannot update deleted audio")
	}

	if err := s.audioRepo.Update(ctx, audio); err != nil {
		return domain.NewDomainError(err, "failed to update audio")
	}

	return nil
}

func (s *AudioService) Delete(ctx context.Context, id string) error {
	audio, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.audioRepo.Delete(ctx, audio.ID); err != nil {
		return domain.NewDomainError(err, "failed to delete audio")
	}

	return nil
}

func (s *AudioService) Transcribe(ctx context.Context, audioID string) (*models.Audio, error) {
	audio, err := s.GetByID(ctx, audioID)
	if err != nil {
		return nil, err
	}

	if len(audio.AudioData) == 0 {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "audio has no data to transcribe")
	}

	// Check if ASR service is available
	if s.asrService == nil {
		return nil, domain.NewDomainError(domain.ErrASRUnavailable, "ASR service not available")
	}

	// Perform transcription
	result, err := s.asrService.Transcribe(ctx, audio.AudioData, audio.AudioFormat)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrTranscriptionFailed, err.Error())
	}

	// Build transcription metadata
	meta := &models.TranscriptionMeta{
		Language:   result.Language,
		Confidence: result.Confidence,
		Duration:   result.Duration,
		Segments:   result.Segments,
	}

	// Update audio with transcription
	audio.SetTranscriptionWithMeta(result.Text, meta)
	if err := s.Update(ctx, audio); err != nil {
		return nil, err
	}

	return audio, nil
}

func (s *AudioService) TranscribeAndCreateMessage(ctx context.Context, audioID, conversationID string) (*models.Message, *models.Audio, error) {
	// Transcribe the audio (external API call, done outside transaction)
	audio, err := s.Transcribe(ctx, audioID)
	if err != nil {
		return nil, nil, err
	}

	if audio.Transcription == "" {
		return nil, nil, domain.NewDomainError(domain.ErrTranscriptionFailed, "transcription produced no text")
	}

	// Verify conversation exists
	if _, err := s.messageRepo.GetByID(ctx, conversationID); err == nil {
		// If we get a result, it's a message ID not a conversation ID - this is an error
		return nil, nil, domain.NewDomainError(domain.ErrInvalidID, "expected conversation ID, got message ID")
	}

	// Get next sequence number
	sequenceNumber, err := s.messageRepo.GetNextSequenceNumber(ctx, conversationID)
	if err != nil {
		return nil, nil, domain.NewDomainError(err, "failed to get next sequence number")
	}

	// Create user message with transcription
	messageID := s.idGenerator.GenerateMessageID()
	message := models.NewUserMessage(messageID, conversationID, sequenceNumber, audio.Transcription)

	// Link audio to message
	audio.MessageID = messageID

	// Wrap message creation and audio linking in a transaction to ensure atomicity
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create message
		if err := s.messageRepo.Create(txCtx, message); err != nil {
			return domain.NewDomainError(err, "failed to create message")
		}

		// Link audio to message
		if err := s.audioRepo.Update(txCtx, audio); err != nil {
			return domain.NewDomainError(err, "failed to link audio to message")
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return message, audio, nil
}

func (s *AudioService) Synthesize(ctx context.Context, text string, options *ports.TTSOptions) (*models.Audio, error) {
	if text == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "text cannot be empty")
	}

	// Check if TTS service is available
	if s.ttsService == nil {
		return nil, domain.NewDomainError(domain.ErrTTSUnavailable, "TTS service not available")
	}

	// Perform synthesis
	result, err := s.ttsService.Synthesize(ctx, text, options)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrTTSFailed, err.Error())
	}

	// Create audio record
	id := s.idGenerator.GenerateAudioID()
	audio := models.NewOutputAudio(id, result.Format)
	audio.SetData(result.Audio, result.DurationMs)

	if err := s.audioRepo.Create(ctx, audio); err != nil {
		return nil, domain.NewDomainError(err, "failed to create synthesized audio")
	}

	return audio, nil
}

func (s *AudioService) SynthesizeForMessage(ctx context.Context, messageID string, options *ports.TTSOptions) (*models.Audio, error) {
	if messageID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "message ID cannot be empty")
	}

	// Get message
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "message not found")
	}

	if message.Contents == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "message has no content to synthesize")
	}

	// Check if TTS service is available
	if s.ttsService == nil {
		return nil, domain.NewDomainError(domain.ErrTTSUnavailable, "TTS service not available")
	}

	// Perform synthesis (external API call, done outside transaction)
	result, err := s.ttsService.Synthesize(ctx, message.Contents, options)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrTTSFailed, err.Error())
	}

	// Create audio record
	id := s.idGenerator.GenerateAudioID()
	audio := models.NewOutputAudio(id, result.Format)
	audio.SetData(result.Audio, result.DurationMs)
	audio.MessageID = messageID

	// Wrap audio creation in a transaction to ensure atomicity
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.audioRepo.Create(txCtx, audio); err != nil {
			return domain.NewDomainError(err, "failed to create synthesized audio")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return audio, nil
}

func (s *AudioService) AssociateLiveKitTrack(ctx context.Context, audioID, trackSID string) (*models.Audio, error) {
	audio, err := s.GetByID(ctx, audioID)
	if err != nil {
		return nil, err
	}

	audio.SetLiveKitTrack(trackSID)
	if err := s.Update(ctx, audio); err != nil {
		return nil, err
	}

	return audio, nil
}

func (s *AudioService) AssociateWithMessage(ctx context.Context, audioID, messageID string) (*models.Audio, error) {
	audio, err := s.GetByID(ctx, audioID)
	if err != nil {
		return nil, err
	}

	// Verify message exists
	if _, err := s.messageRepo.GetByID(ctx, messageID); err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "message not found")
	}

	audio.MessageID = messageID
	if err := s.Update(ctx, audio); err != nil {
		return nil, err
	}

	return audio, nil
}

func (s *AudioService) SetTranscription(ctx context.Context, audioID, transcription string) (*models.Audio, error) {
	audio, err := s.GetByID(ctx, audioID)
	if err != nil {
		return nil, err
	}

	if transcription == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "transcription cannot be empty")
	}

	audio.SetTranscription(transcription)
	if err := s.Update(ctx, audio); err != nil {
		return nil, err
	}

	return audio, nil
}

