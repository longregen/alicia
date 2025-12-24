package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type CommentaryRepository struct {
	BaseRepository
}

func NewCommentaryRepository(pool *pgxpool.Pool) *CommentaryRepository {
	return &CommentaryRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *CommentaryRepository) Create(ctx context.Context, commentary *models.Commentary) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	meta, err := json.Marshal(commentary.Meta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_user_conversation_commentaries (
			id, content, conversation_id, message_id, created_by, meta, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		commentary.ID,
		commentary.Content,
		commentary.ConversationID,
		nullString(commentary.MessageID),
		nullString(commentary.CreatedBy),
		meta,
		commentary.CreatedAt,
		commentary.UpdatedAt,
	)

	return err
}

func (r *CommentaryRepository) GetByID(ctx context.Context, id string) (*models.Commentary, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, content, conversation_id, message_id, created_by, meta, created_at, updated_at, deleted_at
		FROM alicia_user_conversation_commentaries
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanCommentary(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *CommentaryRepository) GetByConversation(ctx context.Context, conversationID string) ([]*models.Commentary, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, content, conversation_id, message_id, created_by, meta, created_at, updated_at, deleted_at
		FROM alicia_user_conversation_commentaries
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCommentaries(rows)
}

func (r *CommentaryRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Commentary, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, content, conversation_id, message_id, created_by, meta, created_at, updated_at, deleted_at
		FROM alicia_user_conversation_commentaries
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCommentaries(rows)
}

func (r *CommentaryRepository) scanCommentary(row pgx.Row) (*models.Commentary, error) {
	var c models.Commentary
	var messageID, createdBy sql.NullString
	var meta []byte

	err := row.Scan(
		&c.ID,
		&c.Content,
		&c.ConversationID,
		&messageID,
		&createdBy,
		&meta,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	if messageID.Valid {
		c.MessageID = messageID.String
	}

	if createdBy.Valid {
		c.CreatedBy = createdBy.String
	}

	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &c.Meta); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

func (r *CommentaryRepository) scanCommentaries(rows pgx.Rows) ([]*models.Commentary, error) {
	var commentaries []*models.Commentary

	for rows.Next() {
		var c models.Commentary
		var messageID, createdBy sql.NullString
		var meta []byte

		err := rows.Scan(
			&c.ID,
			&c.Content,
			&c.ConversationID,
			&messageID,
			&createdBy,
			&meta,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if messageID.Valid {
			c.MessageID = messageID.String
		}

		if createdBy.Valid {
			c.CreatedBy = createdBy.String
		}

		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &c.Meta); err != nil {
				return nil, err
			}
		}

		commentaries = append(commentaries, &c)
	}

	return commentaries, rows.Err()
}
