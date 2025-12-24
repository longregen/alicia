package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type ConversationRepository struct {
	BaseRepository
}

func NewConversationRepository(pool *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *ConversationRepository) Create(ctx context.Context, conversation *models.Conversation) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	preferences, err := marshalJSONField(conversation.Preferences)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_conversations (
			id, user_id, title, status, livekit_room_name, preferences, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		conversation.ID,
		conversation.UserID,
		conversation.Title,
		conversation.Status,
		nullString(conversation.LiveKitRoomName),
		preferences,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	)

	return err
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanConversation(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *ConversationRepository) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	return r.scanConversation(r.conn(ctx).QueryRow(ctx, query, id, userID))
}

func (r *ConversationRepository) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE livekit_room_name = $1 AND deleted_at IS NULL`

	return r.scanConversation(r.conn(ctx).QueryRow(ctx, query, roomName))
}

func (r *ConversationRepository) Update(ctx context.Context, conversation *models.Conversation) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	preferences, err := marshalJSONField(conversation.Preferences)
	if err != nil {
		return err
	}

	query := `
		UPDATE alicia_conversations
		SET title = $2,
			status = $3,
			livekit_room_name = $4,
			preferences = $5,
			updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		conversation.ID,
		conversation.Title,
		conversation.Status,
		nullString(conversation.LiveKitRoomName),
		preferences,
		conversation.UpdatedAt,
	)

	return err
}

// UpdateStanzaIDs updates only the stanza ID fields for reconnection tracking
func (r *ConversationRepository) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_conversations
		SET last_client_stanza_id = GREATEST(last_client_stanza_id, $2),
			last_server_stanza_id = LEAST(last_server_stanza_id, $3),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id, clientStanza, serverStanza)
	return err
}

func (r *ConversationRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_conversations
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *ConversationRepository) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_conversations
		SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id, userID)
	return err
}

func (r *ConversationRepository) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanConversations(rows)
}

func (r *ConversationRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.conn(ctx).Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanConversations(rows)
}

func (r *ConversationRepository) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.conn(ctx).Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanConversations(rows)
}

func (r *ConversationRepository) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, title, status, livekit_room_name, preferences,
		       last_client_stanza_id, last_server_stanza_id,
		       created_at, updated_at, deleted_at
		FROM alicia_conversations
		WHERE user_id = $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.conn(ctx).Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanConversations(rows)
}

// scanConversation scans a single conversation row
func (r *ConversationRepository) scanConversation(row pgx.Row) (*models.Conversation, error) {
	var c models.Conversation
	var preferences []byte
	var livekitRoom sql.NullString

	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Title,
		&c.Status,
		&livekitRoom,
		&preferences,
		&c.LastClientStanzaID,
		&c.LastServerStanzaID,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	)

	if err != nil {
		if checkNoRows(err) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	// Use helper to extract nullable string
	c.LiveKitRoomName = getString(livekitRoom)

	// Use generic JSON unmarshaling helper
	c.Preferences, err = unmarshalJSONPointer[models.ConversationPreferences](preferences)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *ConversationRepository) scanConversations(rows pgx.Rows) ([]*models.Conversation, error) {
	conversations := make([]*models.Conversation, 0)

	for rows.Next() {
		var c models.Conversation
		var preferences []byte
		var livekitRoom sql.NullString

		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.Title,
			&c.Status,
			&livekitRoom,
			&preferences,
			&c.LastClientStanzaID,
			&c.LastServerStanzaID,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Use helpers to reduce boilerplate
		c.LiveKitRoomName = getString(livekitRoom)
		c.Preferences, err = unmarshalJSONPointer[models.ConversationPreferences](preferences)
		if err != nil {
			return nil, err
		}

		conversations = append(conversations, &c)
	}

	return conversations, rows.Err()
}
