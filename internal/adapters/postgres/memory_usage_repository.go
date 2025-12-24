package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type MemoryUsageRepository struct {
	BaseRepository
}

func NewMemoryUsageRepository(pool *pgxpool.Pool) *MemoryUsageRepository {
	return &MemoryUsageRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *MemoryUsageRepository) Create(ctx context.Context, usage *models.MemoryUsage) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	queryPromptMeta, err := json.Marshal(usage.QueryPromptMeta)
	if err != nil {
		return err
	}

	meta, err := json.Marshal(usage.Meta)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO alicia_memory_used (
			id, conversation_id, message_id, memory_id, query_prompt, query_prompt_meta,
			similarity_score, meta, position_in_results, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		usage.ID,
		usage.ConversationID,
		usage.MessageID,
		usage.MemoryID,
		nullString(usage.QueryPrompt),
		queryPromptMeta,
		nullFloat32(usage.SimilarityScore),
		meta,
		nullInt(usage.PositionInResults),
		usage.CreatedAt,
		usage.UpdatedAt,
	)

	return err
}

func (r *MemoryUsageRepository) GetByMessage(ctx context.Context, messageID string) ([]*models.MemoryUsage, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, message_id, memory_id, query_prompt, query_prompt_meta,
			   similarity_score, meta, position_in_results, created_at, updated_at, deleted_at
		FROM alicia_memory_used
		WHERE message_id = $1 AND deleted_at IS NULL
		ORDER BY position_in_results ASC`

	rows, err := r.conn(ctx).Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemoryUsages(rows)
}

func (r *MemoryUsageRepository) GetByConversation(ctx context.Context, conversationID string) ([]*models.MemoryUsage, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, message_id, memory_id, query_prompt, query_prompt_meta,
			   similarity_score, meta, position_in_results, created_at, updated_at, deleted_at
		FROM alicia_memory_used
		WHERE conversation_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemoryUsages(rows)
}

func (r *MemoryUsageRepository) GetByMemory(ctx context.Context, memoryID string) ([]*models.MemoryUsage, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, conversation_id, message_id, memory_id, query_prompt, query_prompt_meta,
			   similarity_score, meta, position_in_results, created_at, updated_at, deleted_at
		FROM alicia_memory_used
		WHERE memory_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemoryUsages(rows)
}

func (r *MemoryUsageRepository) GetUsageStats(ctx context.Context, memoryID string) (*ports.MemoryUsageStats, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT
			memory_id,
			COUNT(*) as total_usage_count,
			AVG(similarity_score) as average_similarity,
			MAX(created_at) as last_used_at
		FROM alicia_memory_used
		WHERE memory_id = $1 AND deleted_at IS NULL
		GROUP BY memory_id`

	var stats ports.MemoryUsageStats
	var avgSimilarity sql.NullFloat64
	var lastUsedAt sql.NullTime

	err := r.conn(ctx).QueryRow(ctx, query, memoryID).Scan(
		&stats.MemoryID,
		&stats.TotalUsageCount,
		&avgSimilarity,
		&lastUsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return zero stats if memory has never been used
			return &ports.MemoryUsageStats{
				MemoryID:          memoryID,
				TotalUsageCount:   0,
				AverageSimilarity: 0,
				LastUsedAt:        nil,
			}, nil
		}
		return nil, err
	}

	if avgSimilarity.Valid {
		stats.AverageSimilarity = float32(avgSimilarity.Float64)
	}

	if lastUsedAt.Valid {
		stats.LastUsedAt = &lastUsedAt.Time
	}

	return &stats, nil
}

func (r *MemoryUsageRepository) scanMemoryUsages(rows pgx.Rows) ([]*models.MemoryUsage, error) {
	var usages []*models.MemoryUsage

	for rows.Next() {
		var mu models.MemoryUsage
		var queryPrompt sql.NullString
		var similarityScore sql.NullFloat64
		var positionInResults sql.NullInt32
		var queryPromptMeta, meta []byte

		err := rows.Scan(
			&mu.ID,
			&mu.ConversationID,
			&mu.MessageID,
			&mu.MemoryID,
			&queryPrompt,
			&queryPromptMeta,
			&similarityScore,
			&meta,
			&positionInResults,
			&mu.CreatedAt,
			&mu.UpdatedAt,
			&mu.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if queryPrompt.Valid {
			mu.QueryPrompt = queryPrompt.String
		}

		if similarityScore.Valid {
			mu.SimilarityScore = float32(similarityScore.Float64)
		}

		if positionInResults.Valid {
			mu.PositionInResults = int(positionInResults.Int32)
		}

		if len(queryPromptMeta) > 0 {
			if err := json.Unmarshal(queryPromptMeta, &mu.QueryPromptMeta); err != nil {
				return nil, err
			}
		}

		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &mu.Meta); err != nil {
				return nil, err
			}
		}

		usages = append(usages, &mu)
	}

	return usages, rows.Err()
}
