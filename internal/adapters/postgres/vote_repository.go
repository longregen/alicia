package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

type VoteRepository struct {
	BaseRepository
}

func NewVoteRepository(pool *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *VoteRepository) Create(ctx context.Context, vote *models.Vote) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Map Vote.Value to database vote string
	voteStr := "up"
	if vote.Value == models.VoteValueDown {
		voteStr = "down"
	}

	// Handle optional quick_feedback and note
	var quickFeedback, note sql.NullString
	if vote.QuickFeedback != "" {
		quickFeedback = sql.NullString{String: vote.QuickFeedback, Valid: true}
	}
	if vote.Note != "" {
		note = sql.NullString{String: vote.Note, Valid: true}
	}

	// Build query based on target type to get conversation_id from appropriate source
	var query string
	var args []interface{}

	switch vote.TargetType {
	case "message":
		// For message votes, get conversation_id from the message itself
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, $2, $3, $4, $5, $6, $7, $8, $9
			FROM alicia_messages m
			WHERE m.id = $2 AND m.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.MessageID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "tool_use":
		// For tool_use votes, get conversation_id through the tool_use's message
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, tu.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_tool_uses tu
			JOIN alicia_messages m ON m.id = tu.message_id AND m.deleted_at IS NULL
			WHERE tu.id = $3 AND tu.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "reasoning":
		// For reasoning votes, get conversation_id through the reasoning step's message
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, rs.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_reasoning_steps rs
			JOIN alicia_messages m ON m.id = rs.message_id AND m.deleted_at IS NULL
			WHERE rs.id = $3 AND rs.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	case "memory":
		// For memory votes, get conversation_id through memory_used junction table
		// If no memory_used record exists, use NULL for message_id and look up any conversation
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, COALESCE(mu.conversation_id, (SELECT id FROM alicia_conversations ORDER BY created_at DESC LIMIT 1)),
			       mu.message_id, $2, $3, $4, $5, $6, $7, $8
			FROM alicia_memory mem
			LEFT JOIN alicia_memory_used mu ON mu.memory_id = mem.id
			WHERE mem.id = $3 AND mem.deleted_at IS NULL
			LIMIT 1`
		args = []interface{}{
			vote.ID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	default:
		// Fallback to original behavior for unknown types
		query = `
			INSERT INTO alicia_votes (
				id, conversation_id, message_id, target_type, target_id, vote, quick_feedback, note, created_at, updated_at
			)
			SELECT $1, m.conversation_id, $2, $3, $4, $5, $6, $7, $8, $9
			FROM alicia_messages m
			WHERE m.id = $2 AND m.deleted_at IS NULL`
		args = []interface{}{
			vote.ID, vote.MessageID, vote.TargetType, vote.TargetID,
			voteStr, quickFeedback, note, vote.CreatedAt, vote.UpdatedAt,
		}
	}

	_, err := r.conn(ctx).Exec(ctx, query, args...)
	return err
}

func (r *VoteRepository) Delete(ctx context.Context, targetType string, targetID string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_votes
		SET deleted_at = NOW()
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, targetType, targetID)
	return err
}

func (r *VoteRepository) GetByTarget(ctx context.Context, targetType string, targetID string) ([]*models.Vote, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, target_type, target_id, message_id, vote, quick_feedback, note, created_at, updated_at
		FROM alicia_votes
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVotes(rows)
}

func (r *VoteRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.Vote, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, target_type, target_id, message_id, vote, quick_feedback, note, created_at, updated_at
		FROM alicia_votes
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVotes(rows)
}

func (r *VoteRepository) GetAggregates(ctx context.Context, targetType string, targetID string) (*models.VoteAggregates, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			target_type,
			target_id,
			COUNT(*) FILTER (WHERE vote = 'up') as upvotes,
			COUNT(*) FILTER (WHERE vote = 'down') as downvotes
		FROM alicia_votes
		WHERE target_type = $1 AND target_id = $2 AND deleted_at IS NULL
		GROUP BY target_type, target_id`

	var aggregates models.VoteAggregates
	var upvotes, downvotes int

	err := r.conn(ctx).QueryRow(ctx, query, targetType, targetID).Scan(
		&aggregates.TargetType,
		&aggregates.TargetID,
		&upvotes,
		&downvotes,
	)

	if err != nil {
		if checkNoRows(err) {
			// No votes found, return zero aggregates
			return &models.VoteAggregates{
				TargetType: targetType,
				TargetID:   targetID,
				Upvotes:    0,
				Downvotes:  0,
				NetScore:   0,
			}, nil
		}
		return nil, err
	}

	aggregates.Upvotes = upvotes
	aggregates.Downvotes = downvotes
	aggregates.NetScore = upvotes - downvotes

	return &aggregates, nil
}

func (r *VoteRepository) scanVotes(rows pgx.Rows) ([]*models.Vote, error) {
	votes := make([]*models.Vote, 0)

	for rows.Next() {
		var v models.Vote
		var voteStr string
		var messageID, quickFeedback, note sql.NullString

		err := rows.Scan(
			&v.ID,
			&v.TargetType,
			&v.TargetID,
			&messageID,
			&voteStr,
			&quickFeedback,
			&note,
			&v.CreatedAt,
			&v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Map database vote string to Vote.Value
		if voteStr == "up" {
			v.Value = models.VoteValueUp
		} else if voteStr == "down" {
			v.Value = models.VoteValueDown
		}

		v.MessageID = getString(messageID)
		v.QuickFeedback = getString(quickFeedback)
		v.Note = getString(note)

		votes = append(votes, &v)
	}

	return votes, rows.Err()
}
