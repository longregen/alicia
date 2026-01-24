package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// CreateMessage inserts a new message with auto-computed branch_index.
// Uses ON CONFLICT to handle cases where the message already exists (e.g., agent created it first).
func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	if msg.Status == "" {
		msg.Status = domain.MessageStatusPending
	}

	var query string
	var args []any

	if msg.PreviousID == nil {
		// Root message (no previous_id)
		// Use separate parameters for subquery to avoid type inference issues
		query = `
			INSERT INTO messages (id, conversation_id, previous_id, branch_index, role, content, reasoning, status, created_at)
			VALUES ($1, $2, NULL,
				COALESCE((
					SELECT MAX(branch_index) + 1
					FROM messages
					WHERE conversation_id = $8
					  AND previous_id IS NULL
					  AND deleted_at IS NULL
				), 0),
				$3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				reasoning = EXCLUDED.reasoning,
				status = EXCLUDED.status
			RETURNING branch_index`
		args = []any{
			msg.ID, msg.ConversationID,
			msg.Role, msg.Content, msg.Reasoning, msg.Status, msg.CreatedAt,
			msg.ConversationID, // $8 - duplicate for subquery
		}
	} else {
		// Reply message (has previous_id)
		// Use separate parameters for subquery to avoid type inference issues
		query = `
			INSERT INTO messages (id, conversation_id, previous_id, branch_index, role, content, reasoning, status, created_at)
			VALUES ($1, $2, $3,
				COALESCE((
					SELECT MAX(branch_index) + 1
					FROM messages
					WHERE conversation_id = $9
					  AND previous_id = $10
					  AND deleted_at IS NULL
				), 0),
				$4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				reasoning = EXCLUDED.reasoning,
				status = EXCLUDED.status
			RETURNING branch_index`
		args = []any{
			msg.ID, msg.ConversationID, *msg.PreviousID,
			msg.Role, msg.Content, msg.Reasoning, msg.Status, msg.CreatedAt,
			msg.ConversationID, *msg.PreviousID, // $9, $10 - duplicates for subquery
		}
	}

	err := s.conn(ctx).QueryRow(ctx, query, args...).Scan(&msg.BranchIndex)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

// GetMessage retrieves a message by ID.
func (s *Store) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	query := `
		SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at
		FROM messages
		WHERE id = $1 AND deleted_at IS NULL`

	msg := &domain.Message{}
	err := s.conn(ctx).QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ConversationID, &msg.PreviousID, &msg.BranchIndex,
		&msg.Role, &msg.Content, &msg.Reasoning, &msg.Status, &msg.TraceID, &msg.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get message: %w", err)
	}
	return msg, nil
}

// UpdateMessage updates a message's content and status.
func (s *Store) UpdateMessage(ctx context.Context, msg *domain.Message) error {
	query := `
		UPDATE messages
		SET content = $2, reasoning = $3, status = $4
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := s.conn(ctx).Exec(ctx, query, msg.ID, msg.Content, msg.Reasoning, msg.Status)
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	return nil
}

// UpdateMessageStatus updates only the message status.
func (s *Store) UpdateMessageStatus(ctx context.Context, id, status string) error {
	query := `UPDATE messages SET status = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.conn(ctx).Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}
	return nil
}

// DeleteMessage soft-deletes a message.
func (s *Store) DeleteMessage(ctx context.Context, id string) error {
	query := `UPDATE messages SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	result, err := s.conn(ctx).Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListMessages returns messages for a conversation ordered by creation time.
func (s *Store) ListMessages(ctx context.Context, conversationID string, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at
		FROM messages
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := s.conn(ctx).Query(ctx, query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetMessageChain returns the chain of messages from root to the given tip.
func (s *Store) GetMessageChain(ctx context.Context, tipID string) ([]*domain.Message, error) {
	query := `
		WITH RECURSIVE chain AS (
			SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at, 0 as depth
			FROM messages
			WHERE id = $1 AND deleted_at IS NULL

			UNION ALL

			SELECT m.id, m.conversation_id, m.previous_id, m.branch_index, m.role, m.content, m.reasoning, m.status, m.trace_id, m.created_at, c.depth + 1
			FROM messages m
			JOIN chain c ON m.id = c.previous_id
			WHERE m.deleted_at IS NULL
		)
		SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at
		FROM chain
		ORDER BY depth DESC`

	rows, err := s.conn(ctx).Query(ctx, query, tipID)
	if err != nil {
		return nil, fmt.Errorf("get message chain: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetMessageSiblings returns all messages with the same previous_id.
func (s *Store) GetMessageSiblings(ctx context.Context, messageID string) ([]*domain.Message, error) {
	msg, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	var query string
	var args []any

	if msg.PreviousID == nil {
		query = `
			SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at
			FROM messages
			WHERE conversation_id = $1 AND previous_id IS NULL AND deleted_at IS NULL
			ORDER BY branch_index ASC`
		args = []any{msg.ConversationID}
	} else {
		query = `
			SELECT id, conversation_id, previous_id, branch_index, role, content, reasoning, status, trace_id, created_at
			FROM messages
			WHERE previous_id = $1 AND deleted_at IS NULL
			ORDER BY branch_index ASC`
		args = []any{*msg.PreviousID}
	}

	rows, err := s.conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get siblings: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func scanMessages(rows pgx.Rows) ([]*domain.Message, error) {
	var msgs []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.PreviousID, &msg.BranchIndex,
			&msg.Role, &msg.Content, &msg.Reasoning, &msg.Status, &msg.TraceID, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// UpdateMessageTraceID updates a message's trace_id for Langfuse correlation.
func (s *Store) UpdateMessageTraceID(ctx context.Context, messageID, traceID string) error {
	query := `UPDATE messages SET trace_id = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := s.conn(ctx).Exec(ctx, query, messageID, traceID)
	if err != nil {
		return fmt.Errorf("update message trace_id: %w", err)
	}
	return nil
}

// GetMessageTraceID retrieves the trace_id for a message.
func (s *Store) GetMessageTraceID(ctx context.Context, messageID string) (string, error) {
	query := `SELECT trace_id FROM messages WHERE id = $1 AND deleted_at IS NULL`
	var traceID *string
	err := s.conn(ctx).QueryRow(ctx, query, messageID).Scan(&traceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("get message trace_id: %w", err)
	}
	if traceID == nil {
		return "", nil
	}
	return *traceID, nil
}
