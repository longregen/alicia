package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type AudioRepository struct {
	BaseRepository
}

func NewAudioRepository(pool *pgxpool.Pool) *AudioRepository {
	return &AudioRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *AudioRepository) Create(ctx context.Context, audio *models.Audio) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	transcriptionMeta, err := marshalJSONField(audio.TranscriptionMeta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_audio (
			id, message_id, audio_type, audio_format, audio_data, duration_ms,
			transcription, livekit_track_sid, transcription_meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		audio.ID,
		nullString(audio.MessageID),
		audio.AudioType,
		audio.AudioFormat,
		audio.AudioData,
		nullInt(audio.DurationMs),
		nullString(audio.Transcription),
		nullString(audio.LiveKitTrackSID),
		transcriptionMeta,
		audio.CreatedAt,
		audio.UpdatedAt,
	)

	return err
}

func (r *AudioRepository) GetByID(ctx context.Context, id string) (*models.Audio, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, audio_type, audio_format, audio_data, duration_ms,
			   transcription, livekit_track_sid, transcription_meta, created_at, updated_at, deleted_at
		FROM alicia_audio
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanAudio(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *AudioRepository) GetByMessage(ctx context.Context, messageID string) (*models.Audio, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, audio_type, audio_format, audio_data, duration_ms,
			   transcription, livekit_track_sid, transcription_meta, created_at, updated_at, deleted_at
		FROM alicia_audio
		WHERE message_id = $1 AND deleted_at IS NULL
		LIMIT 1`

	return r.scanAudio(r.conn(ctx).QueryRow(ctx, query, messageID))
}

func (r *AudioRepository) GetByLiveKitTrack(ctx context.Context, trackSID string) (*models.Audio, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, audio_type, audio_format, audio_data, duration_ms,
			   transcription, livekit_track_sid, transcription_meta, created_at, updated_at, deleted_at
		FROM alicia_audio
		WHERE livekit_track_sid = $1 AND deleted_at IS NULL`

	return r.scanAudio(r.conn(ctx).QueryRow(ctx, query, trackSID))
}

func (r *AudioRepository) Update(ctx context.Context, audio *models.Audio) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	transcriptionMeta, err := marshalJSONField(audio.TranscriptionMeta)
	if err != nil {
		return err
	}

	query := `
		UPDATE alicia_audio
		SET message_id = $2,
			audio_data = $3,
			duration_ms = $4,
			transcription = $5,
			livekit_track_sid = $6,
			transcription_meta = $7,
			updated_at = $8
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		audio.ID,
		nullString(audio.MessageID),
		audio.AudioData,
		nullInt(audio.DurationMs),
		nullString(audio.Transcription),
		nullString(audio.LiveKitTrackSID),
		transcriptionMeta,
		audio.UpdatedAt,
	)

	return err
}

func (r *AudioRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_audio
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *AudioRepository) scanAudio(row pgx.Row) (*models.Audio, error) {
	var a models.Audio
	var messageID, transcription, livekitTrackSID sql.NullString
	var durationMs sql.NullInt32
	var transcriptionMeta []byte

	err := row.Scan(
		&a.ID,
		&messageID,
		&a.AudioType,
		&a.AudioFormat,
		&a.AudioData,
		&durationMs,
		&transcription,
		&livekitTrackSID,
		&transcriptionMeta,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.DeletedAt,
	)

	if err != nil {
		if checkNoRows(err) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	// Use helpers to extract nullable fields
	a.MessageID = getString(messageID)
	a.DurationMs = getInt(durationMs)
	a.Transcription = getString(transcription)
	a.LiveKitTrackSID = getString(livekitTrackSID)

	// Use generic JSON unmarshaling helper
	a.TranscriptionMeta, err = unmarshalJSONPointer[models.TranscriptionMeta](transcriptionMeta)
	if err != nil {
		return nil, err
	}

	return &a, nil
}
