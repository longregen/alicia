package postgres

import (
	"context"
	"database/sql"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type MessageRepository struct {
	BaseRepository
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *MessageRepository) Create(ctx context.Context, message *models.Message) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO alicia_messages (
			id, conversation_id, sequence_number, previous_id, message_role, contents,
			local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err := r.conn(ctx).Exec(ctx, query,
		message.ID,
		message.ConversationID,
		message.SequenceNumber,
		nullString(message.PreviousID),
		message.Role,
		message.Contents,
		nullString(message.LocalID),
		nullString(message.ServerID),
		message.SyncStatus,
		nullTime(message.SyncedAt),
		message.CompletionStatus,
		message.CreatedAt,
		message.UpdatedAt,
	)

	return err
}

func (r *MessageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanMessage(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *MessageRepository) Update(ctx context.Context, message *models.Message) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_messages
		SET previous_id = $2,
			contents = $3,
			local_id = $4,
			server_id = $5,
			sync_status = $6,
			synced_at = $7,
			completion_status = $8,
			updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query,
		message.ID,
		nullString(message.PreviousID),
		message.Contents,
		nullString(message.LocalID),
		nullString(message.ServerID),
		message.SyncStatus,
		nullTime(message.SyncedAt),
		message.CompletionStatus,
		message.UpdatedAt,
	)

	return err
}

func (r *MessageRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_messages
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}

func (r *MessageRepository) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

func (r *MessageRepository) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY sequence_number DESC
		LIMIT $2`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages, err := r.scanMessages(rows)
	if err != nil {
		return nil, err
	}

	// Reverse the slice to get ascending order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *MessageRepository) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// If we're in a transaction, use the transaction connection
	if tx := GetTx(ctx); tx != nil {
		// log.Printf("GetNextSequenceNumber: using existing transaction")
		return r.getNextSequenceWithConn(ctx, tx, conversationID)
	}

	// Otherwise, we need to start a transaction to use transaction-scoped advisory locks
	// This ensures the lock is held for the duration of the sequence number generation
	// and automatically released when the transaction ends
	// log.Printf("GetNextSequenceNumber: starting new transaction")
	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) // Rollback is safe to call even after commit

	nextSeq, err := r.getNextSequenceWithConn(ctx, tx, conversationID)
	if err != nil {
		return 0, err
	}

	// Commit the transaction to release the advisory lock
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return nextSeq, nil
}

func (r *MessageRepository) getNextSequenceWithConn(ctx context.Context, conn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, conversationID string) (int, error) {
	// Use PostgreSQL advisory lock with transaction-level scope
	// Hash the conversation ID to a 64-bit integer for the advisory lock
	lockID := hashConversationID(conversationID)

	// Use pg_advisory_xact_lock for transaction-scoped lock
	// This automatically releases when the connection is returned to the pool
	_, err := conn.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID)
	if err != nil {
		return 0, err
	}

	query := `
		SELECT COALESCE(MAX(sequence_number), 0) + 1 as next_sequence
		FROM alicia_messages
		WHERE conversation_id = $1 AND deleted_at IS NULL`

	var nextSeq int
	err = conn.QueryRow(ctx, query, conversationID).Scan(&nextSeq)
	if err != nil {
		return 0, err
	}

	return nextSeq, nil
}

// hashConversationID generates a 64-bit hash from a conversation ID for use with advisory locks
func hashConversationID(conversationID string) int64 {
	h := fnv.New64a()
	h.Write([]byte(conversationID))
	// Convert to int64, preserving all bits
	return int64(h.Sum64())
}

func (r *MessageRepository) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE conversation_id = $1 AND sequence_number > $2 AND deleted_at IS NULL
		ORDER BY sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID, afterSequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetPendingSync retrieves all messages pending synchronization for a conversation
func (r *MessageRepository) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE conversation_id = $1
		  AND sync_status = 'pending'
		  AND deleted_at IS NULL
		ORDER BY sequence_number ASC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetByLocalID retrieves a message by its local ID
func (r *MessageRepository) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE local_id = $1 AND deleted_at IS NULL`

	return r.scanMessage(r.conn(ctx).QueryRow(ctx, query, localID))
}

func (r *MessageRepository) scanMessage(row pgx.Row) (*models.Message, error) {
	var m models.Message
	var previousID, localID, serverID sql.NullString
	var syncedAt sql.NullTime

	err := row.Scan(
		&m.ID,
		&m.ConversationID,
		&m.SequenceNumber,
		&previousID,
		&m.Role,
		&m.Contents,
		&localID,
		&serverID,
		&m.SyncStatus,
		&syncedAt,
		&m.CompletionStatus,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.DeletedAt,
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
	m.PreviousID = getString(previousID)
	m.LocalID = getString(localID)
	m.ServerID = getString(serverID)
	m.SyncedAt = getTimePtr(syncedAt)

	return &m, nil
}

func (r *MessageRepository) scanMessages(rows pgx.Rows) ([]*models.Message, error) {
	var messages []*models.Message

	for rows.Next() {
		var m models.Message
		var previousID, localID, serverID sql.NullString
		var syncedAt sql.NullTime

		err := rows.Scan(
			&m.ID,
			&m.ConversationID,
			&m.SequenceNumber,
			&previousID,
			&m.Role,
			&m.Contents,
			&localID,
			&serverID,
			&m.SyncStatus,
			&syncedAt,
			&m.CompletionStatus,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Use helpers to reduce boilerplate
		m.PreviousID = getString(previousID)
		m.LocalID = getString(localID)
		m.ServerID = getString(serverID)
		m.SyncedAt = getTimePtr(syncedAt)

		messages = append(messages, &m)
	}

	return messages, rows.Err()
}

// GetIncompleteOlderThan retrieves messages with incomplete status older than the given time
func (r *MessageRepository) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE completion_status IN ('pending', 'streaming', 'failed')
		  AND created_at < $1
		  AND deleted_at IS NULL
		ORDER BY created_at ASC`

	rows, err := r.conn(ctx).Query(ctx, query, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetIncompleteByConversation retrieves incomplete messages for a specific conversation
func (r *MessageRepository) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE conversation_id = $1
		  AND completion_status IN ('pending', 'streaming', 'failed')
		  AND created_at < $2
		  AND deleted_at IS NULL
		ORDER BY created_at ASC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetChainFromTip walks backwards from the tip message via previous_id and returns messages in chronological order
func (r *MessageRepository) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Use recursive CTE to walk the chain backwards, then reverse the order
	query := `
		WITH RECURSIVE message_chain AS (
			-- Base case: start with the tip message
			SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
			       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at,
			       1 as depth
			FROM alicia_messages
			WHERE id = $1 AND deleted_at IS NULL

			UNION ALL

			-- Recursive case: follow previous_id backwards
			SELECT m.id, m.conversation_id, m.sequence_number, m.previous_id, m.message_role, m.contents,
			       m.local_id, m.server_id, m.sync_status, m.synced_at, m.completion_status, m.created_at, m.updated_at, m.deleted_at,
			       mc.depth + 1
			FROM alicia_messages m
			INNER JOIN message_chain mc ON m.id = mc.previous_id
			WHERE m.deleted_at IS NULL
		)
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM message_chain
		ORDER BY depth DESC`

	rows, err := r.conn(ctx).Query(ctx, query, tipMessageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}

// GetSiblings returns all messages that share the same previous_id (i.e., branches from the same parent)
func (r *MessageRepository) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// First get the previous_id of the given message
	var previousID sql.NullString
	queryPrevious := `SELECT previous_id FROM alicia_messages WHERE id = $1 AND deleted_at IS NULL`
	err := r.conn(ctx).QueryRow(ctx, queryPrevious, messageID).Scan(&previousID)
	if err != nil {
		return nil, err
	}

	// If no previous_id, there are no siblings (this is a root message)
	if !previousID.Valid {
		return []*models.Message{}, nil
	}

	// Get all messages with the same previous_id
	query := `
		SELECT id, conversation_id, sequence_number, previous_id, message_role, contents,
		       local_id, server_id, sync_status, synced_at, completion_status, created_at, updated_at, deleted_at
		FROM alicia_messages
		WHERE previous_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`

	rows, err := r.conn(ctx).Query(ctx, query, previousID.String)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMessages(rows)
}
