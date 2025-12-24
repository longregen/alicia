package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type SentenceRepository struct {
	BaseRepository
}

func NewSentenceRepository(pool *pgxpool.Pool) *SentenceRepository {
	return &SentenceRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *SentenceRepository) Create(ctx context.Context, sentence *models.Sentence) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	meta, err := json.Marshal(sentence.Meta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_sentences (
			id, message_id, sentence_sequence_number, text, audio_type, audio_format,
			duration_ms, audio_bytesize, audio_data, meta, completion_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		sentence.ID,
		sentence.MessageID,
		sentence.SequenceNumber,
		sentence.Text,
		nullString(string(sentence.AudioType)),
		nullString(sentence.AudioFormat),
		nullInt(sentence.DurationMs),
		nullInt(sentence.AudioBytesize),
		sentence.AudioData,
		meta,
		sentence.CompletionStatus,
		sentence.CreatedAt,
		sentence.UpdatedAt,
	)

	return err
}

func (r *SentenceRepository) GetByID(ctx context.Context, id string) (*models.Sentence, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, sentence_sequence_number, text, audio_type, audio_format,
			   duration_ms, audio_bytesize, audio_data, meta, completion_status, created_at, updated_at, deleted_at
		FROM alicia_sentences
		WHERE id = $1 AND deleted_at IS NULL`

	var s models.Sentence
	var audioType, audioFormat sql.NullString
	var durationMs, audioBytesize sql.NullInt32
	var meta []byte

	err := r.conn(ctx).QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.MessageID,
		&s.SequenceNumber,
		&s.Text,
		&audioType,
		&audioFormat,
		&durationMs,
		&audioBytesize,
		&s.AudioData,
		&meta,
		&s.CompletionStatus,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	if audioType.Valid {
		s.AudioType = models.AudioType(audioType.String)
	}
	if audioFormat.Valid {
		s.AudioFormat = audioFormat.String
	}
	if durationMs.Valid {
		s.DurationMs = int(durationMs.Int32)
	}
	if audioBytesize.Valid {
		s.AudioBytesize = int(audioBytesize.Int32)
	}

	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &s.Meta); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

func (r *SentenceRepository) Update(ctx context.Context, sentence *models.Sentence) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	meta, err := json.Marshal(sentence.Meta)
	if err != nil {
		return err
	}

	query := `
		UPDATE alicia_sentences
		SET text = $2,
			audio_type = $3,
			audio_format = $4,
			duration_ms = $5,
			audio_bytesize = $6,
			audio_data = $7,
			meta = $8,
			completion_status = $9,
			updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		sentence.ID,
		sentence.Text,
		nullString(string(sentence.AudioType)),
		nullString(sentence.AudioFormat),
		nullInt(sentence.DurationMs),
		nullInt(sentence.AudioBytesize),
		sentence.AudioData,
		meta,
		sentence.CompletionStatus,
		sentence.UpdatedAt,
	)

	return err
}

func (r *SentenceRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_sentences
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *SentenceRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, message_id, sentence_sequence_number, text, audio_type, audio_format,
			   duration_ms, audio_bytesize, audio_data, meta, completion_status, created_at, updated_at, deleted_at
		FROM alicia_sentences
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY sentence_sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sentences []*models.Sentence

	for rows.Next() {
		var s models.Sentence
		var audioType, audioFormat sql.NullString
		var durationMs, audioBytesize sql.NullInt32
		var meta []byte

		err := rows.Scan(
			&s.ID,
			&s.MessageID,
			&s.SequenceNumber,
			&s.Text,
			&audioType,
			&audioFormat,
			&durationMs,
			&audioBytesize,
			&s.AudioData,
			&meta,
			&s.CompletionStatus,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if audioType.Valid {
			s.AudioType = models.AudioType(audioType.String)
		}
		if audioFormat.Valid {
			s.AudioFormat = audioFormat.String
		}
		if durationMs.Valid {
			s.DurationMs = int(durationMs.Int32)
		}
		if audioBytesize.Valid {
			s.AudioBytesize = int(audioBytesize.Int32)
		}

		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &s.Meta); err != nil {
				return nil, err
			}
		}

		sentences = append(sentences, &s)
	}

	return sentences, rows.Err()
}

func (r *SentenceRepository) GetNextSequenceNumber(ctx context.Context, messageID string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT COALESCE(MAX(sentence_sequence_number), 0) + 1 as next_sequence
		FROM alicia_sentences
		WHERE message_id = $1 AND deleted_at IS NULL`

	var nextSeq int
	err := r.conn(ctx).QueryRow(ctx, query, messageID).Scan(&nextSeq)
	if err != nil {
		return 0, err
	}

	return nextSeq, nil
}

// GetIncompleteOlderThan retrieves sentences with incomplete status older than the given time
func (r *SentenceRepository) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Sentence, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT s.id, s.message_id, s.sentence_sequence_number, s.text, s.audio_type, s.audio_format,
			   s.duration_ms, s.audio_bytesize, s.audio_data, s.meta, s.completion_status,
			   s.created_at, s.updated_at, s.deleted_at
		FROM alicia_sentences s
		WHERE s.completion_status IN ('pending', 'streaming', 'failed')
		  AND s.created_at < $1
		  AND s.deleted_at IS NULL
		ORDER BY s.created_at ASC`

	rows, err := r.conn(ctx).Query(ctx, query, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSentences(rows)
}

// GetIncompleteByConversation retrieves incomplete sentences for a specific conversation
func (r *SentenceRepository) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Sentence, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT s.id, s.message_id, s.sentence_sequence_number, s.text, s.audio_type, s.audio_format,
			   s.duration_ms, s.audio_bytesize, s.audio_data, s.meta, s.completion_status,
			   s.created_at, s.updated_at, s.deleted_at
		FROM alicia_sentences s
		JOIN alicia_messages m ON s.message_id = m.id
		WHERE m.conversation_id = $1
		  AND s.completion_status IN ('pending', 'streaming', 'failed')
		  AND s.created_at < $2
		  AND s.deleted_at IS NULL
		ORDER BY s.created_at ASC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanSentences(rows)
}

// scanSentences is a helper to scan multiple sentence rows
func (r *SentenceRepository) scanSentences(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*models.Sentence, error) {
	var sentences []*models.Sentence

	for rows.Next() {
		var s models.Sentence
		var audioType, audioFormat sql.NullString
		var durationMs, audioBytesize sql.NullInt32
		var meta []byte

		err := rows.Scan(
			&s.ID,
			&s.MessageID,
			&s.SequenceNumber,
			&s.Text,
			&audioType,
			&audioFormat,
			&durationMs,
			&audioBytesize,
			&s.AudioData,
			&meta,
			&s.CompletionStatus,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if audioType.Valid {
			s.AudioType = models.AudioType(audioType.String)
		}
		if audioFormat.Valid {
			s.AudioFormat = audioFormat.String
		}
		if durationMs.Valid {
			s.DurationMs = int(durationMs.Int32)
		}
		if audioBytesize.Valid {
			s.AudioBytesize = int(audioBytesize.Int32)
		}

		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &s.Meta); err != nil {
				return nil, err
			}
		}

		sentences = append(sentences, &s)
	}

	return sentences, rows.Err()
}
