package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type SessionStatsRepository struct {
	BaseRepository
}

func NewSessionStatsRepository(pool *pgxpool.Pool) *SessionStatsRepository {
	return &SessionStatsRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *SessionStatsRepository) Create(ctx context.Context, stats *models.SessionStats) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO alicia_session_stats (
			id, conversation_id, message_count, tool_call_count,
			memories_used, session_duration_seconds, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err := r.conn(ctx).Exec(ctx, query,
		stats.ID,
		stats.ConversationID,
		stats.MessageCount,
		stats.ToolCallCount,
		stats.MemoryRetrievals,
		stats.SessionDurationMs/1000, // Convert milliseconds to seconds
		stats.CreatedAt,
		stats.UpdatedAt,
	)

	return err
}

func (r *SessionStatsRepository) Update(ctx context.Context, stats *models.SessionStats) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_session_stats
		SET message_count = $1,
		    tool_call_count = $2,
		    memories_used = $3,
		    session_duration_seconds = $4,
		    updated_at = $5
		WHERE id = $6`

	result, err := r.conn(ctx).Exec(ctx, query,
		stats.MessageCount,
		stats.ToolCallCount,
		stats.MemoryRetrievals,
		stats.SessionDurationMs/1000, // Convert milliseconds to seconds
		stats.UpdatedAt,
		stats.ID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("session stats not found")
	}

	return nil
}

func (r *SessionStatsRepository) GetByConversation(ctx context.Context, conversationID string) (*models.SessionStats, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, message_count, tool_call_count,
		       memories_used, session_duration_seconds, created_at, updated_at
		FROM alicia_session_stats
		WHERE conversation_id = $1`

	return r.scanSessionStats(r.conn(ctx).QueryRow(ctx, query, conversationID))
}

func (r *SessionStatsRepository) scanSessionStats(row pgx.Row) (*models.SessionStats, error) {
	var stats models.SessionStats
	var sessionDurationSeconds int64

	err := row.Scan(
		&stats.ID,
		&stats.ConversationID,
		&stats.MessageCount,
		&stats.ToolCallCount,
		&stats.MemoryRetrievals,
		&sessionDurationSeconds,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	// Convert seconds to milliseconds
	stats.SessionDurationMs = sessionDurationSeconds * 1000

	// Initialize fields not stored in database
	stats.UserMessageCount = 0
	stats.TotalTokensUsed = 0
	stats.TotalLatencyMs = 0
	stats.AverageLatencyMs = 0
	stats.ErrorCount = 0
	stats.StartedAt = stats.CreatedAt
	stats.LastActivityAt = stats.UpdatedAt
	stats.Meta = make(map[string]any)

	return &stats, nil
}
