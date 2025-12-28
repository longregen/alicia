package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/pgvector/pgvector-go"
)

type MemoryRepository struct {
	BaseRepository
}

func NewMemoryRepository(pool *pgxpool.Pool) *MemoryRepository {
	return &MemoryRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, memory *models.Memory) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var embeddingsInfo, sourceInfo []byte
	var err error

	if memory.EmbeddingsInfo != nil {
		embeddingsInfo, err = json.Marshal(memory.EmbeddingsInfo)
		if err != nil {
			return err
		}
	}

	if memory.SourceInfo != nil {
		sourceInfo, err = json.Marshal(memory.SourceInfo)
		if err != nil {
			return err
		}
	}

	var embeddings *pgvector.Vector
	if len(memory.Embeddings) > 0 {
		v := pgvector.NewVector(memory.Embeddings)
		embeddings = &v
	}

	query := `
		INSERT INTO alicia_memory (
			id, content, embeddings, embeddings_info, importance, confidence,
			user_rating, created_by, source_type, source_info, tags, pinned, archived, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		memory.ID,
		memory.Content,
		embeddings,
		embeddingsInfo,
		memory.Importance,
		memory.Confidence,
		nullInt(ptrIntToInt(memory.UserRating)),
		nullString(memory.CreatedBy),
		nullString(memory.SourceType),
		sourceInfo,
		memory.Tags,
		memory.Pinned,
		memory.Archived,
		memory.CreatedAt,
		memory.UpdatedAt,
	)

	return err
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, content, embeddings, embeddings_info, importance, confidence,
			   user_rating, created_by, source_type, source_info, tags, pinned, archived, created_at, updated_at, deleted_at
		FROM alicia_memory
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanMemory(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *MemoryRepository) Update(ctx context.Context, memory *models.Memory) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var embeddingsInfo, sourceInfo []byte
	var err error

	if memory.EmbeddingsInfo != nil {
		embeddingsInfo, err = json.Marshal(memory.EmbeddingsInfo)
		if err != nil {
			return err
		}
	}

	if memory.SourceInfo != nil {
		sourceInfo, err = json.Marshal(memory.SourceInfo)
		if err != nil {
			return err
		}
	}

	var embeddings *pgvector.Vector
	if len(memory.Embeddings) > 0 {
		v := pgvector.NewVector(memory.Embeddings)
		embeddings = &v
	}

	query := `
		UPDATE alicia_memory
		SET content = $2,
			embeddings = $3,
			embeddings_info = $4,
			importance = $5,
			confidence = $6,
			user_rating = $7,
			created_by = $8,
			source_type = $9,
			source_info = $10,
			tags = $11,
			pinned = $12,
			archived = $13,
			updated_at = $14
		WHERE id = $1 AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, query,
		memory.ID,
		memory.Content,
		embeddings,
		embeddingsInfo,
		memory.Importance,
		memory.Confidence,
		nullInt(ptrIntToInt(memory.UserRating)),
		nullString(memory.CreatedBy),
		nullString(memory.SourceType),
		sourceInfo,
		memory.Tags,
		memory.Pinned,
		memory.Archived,
		memory.UpdatedAt,
	)

	return err
}

func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_memory
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	conn := GetConn(ctx, r.pool)
	_, err := conn.Exec(ctx, query, id)
	return err
}

// SearchMemories performs a unified search with configurable options.
// This is the recommended method for memory searches, as it consolidates all search variants.
func (r *MemoryRepository) SearchMemories(ctx context.Context, opts ports.MemorySearchOptions) ([]*ports.MemorySearchResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if len(opts.Embedding) == 0 {
		return nil, errors.New("embedding cannot be empty")
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	vector := pgvector.NewVector(opts.Embedding)

	// Build query based on whether we need scores and/or threshold
	var query string
	var args []interface{}

	if opts.IncludeScores {
		query = `
			SELECT id, content, embeddings, embeddings_info, importance, confidence,
				   user_rating, created_by, source_type, source_info, tags, pinned, archived, created_at, updated_at, deleted_at,
				   1 - (embeddings <=> $1) as similarity
			FROM alicia_memory
			WHERE deleted_at IS NULL AND embeddings IS NOT NULL`
		args = []interface{}{vector}

		if opts.Threshold != nil {
			query += ` AND 1 - (embeddings <=> $1) >= $2`
			args = append(args, *opts.Threshold)
		}

		query += ` ORDER BY embeddings <=> $1 LIMIT $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, opts.Limit)
	} else {
		query = `
			SELECT id, content, embeddings, embeddings_info, importance, confidence,
				   user_rating, created_by, source_type, source_info, tags, pinned, archived, created_at, updated_at, deleted_at
			FROM alicia_memory
			WHERE deleted_at IS NULL AND embeddings IS NOT NULL`
		args = []interface{}{vector}

		if opts.Threshold != nil {
			query += ` AND 1 - (embeddings <=> $1) >= $2`
			args = append(args, *opts.Threshold)
		}

		query += ` ORDER BY embeddings <=> $1 LIMIT $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, opts.Limit)
	}

	rows, err := r.conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ports.MemorySearchResult
	for rows.Next() {
		var m models.Memory
		var embeddings *pgvector.Vector
		var embeddingsInfo, sourceInfo []byte
		var userRating sql.NullInt32
		var createdBy, sourceType sql.NullString
		var similarity float32

		var scanArgs []interface{}
		if opts.IncludeScores {
			scanArgs = []interface{}{
				&m.ID,
				&m.Content,
				&embeddings,
				&embeddingsInfo,
				&m.Importance,
				&m.Confidence,
				&userRating,
				&createdBy,
				&sourceType,
				&sourceInfo,
				&m.Tags,
				&m.Pinned,
				&m.Archived,
				&m.CreatedAt,
				&m.UpdatedAt,
				&m.DeletedAt,
				&similarity,
			}
		} else {
			scanArgs = []interface{}{
				&m.ID,
				&m.Content,
				&embeddings,
				&embeddingsInfo,
				&m.Importance,
				&m.Confidence,
				&userRating,
				&createdBy,
				&sourceType,
				&sourceInfo,
				&m.Tags,
				&m.Pinned,
				&m.Archived,
				&m.CreatedAt,
				&m.UpdatedAt,
				&m.DeletedAt,
			}
			similarity = 0
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		if embeddings != nil {
			m.Embeddings = embeddings.Slice()
		}

		if len(embeddingsInfo) > 0 {
			var info models.EmbeddingsInfo
			if err := json.Unmarshal(embeddingsInfo, &info); err != nil {
				return nil, fmt.Errorf("failed to unmarshal embeddings info: %w", err)
			}
			m.EmbeddingsInfo = &info
		}

		if userRating.Valid {
			rating := int(userRating.Int32)
			m.UserRating = &rating
		}

		if createdBy.Valid {
			m.CreatedBy = createdBy.String
		}

		if sourceType.Valid {
			m.SourceType = sourceType.String
		}

		if len(sourceInfo) > 0 {
			var info models.SourceInfo
			if err := json.Unmarshal(sourceInfo, &info); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source info: %w", err)
			}
			m.SourceInfo = &info
		}

		results = append(results, &ports.MemorySearchResult{
			Memory:     &m,
			Similarity: similarity,
		})
	}

	return results, rows.Err()
}

func (r *MemoryRepository) GetByTags(ctx context.Context, tags []string, limit int) ([]*models.Memory, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, content, embeddings, embeddings_info, importance, confidence,
			   user_rating, created_by, source_type, source_info, tags, pinned, archived, created_at, updated_at, deleted_at
		FROM alicia_memory
		WHERE deleted_at IS NULL AND tags && $1
		ORDER BY importance DESC, created_at DESC
		LIMIT $2`

	rows, err := r.conn(ctx).Query(ctx, query, tags, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemories(rows)
}

func (r *MemoryRepository) scanMemory(row pgx.Row) (*models.Memory, error) {
	var m models.Memory
	var embeddings *pgvector.Vector
	var embeddingsInfo, sourceInfo []byte
	var userRating sql.NullInt32
	var createdBy, sourceType sql.NullString

	err := row.Scan(
		&m.ID,
		&m.Content,
		&embeddings,
		&embeddingsInfo,
		&m.Importance,
		&m.Confidence,
		&userRating,
		&createdBy,
		&sourceType,
		&sourceInfo,
		&m.Tags,
		&m.Pinned,
		&m.Archived,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a domain error for not found instead of nil, nil to prevent nil pointer issues
			// in calling code that checks err != nil and then accesses the result
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}

	if embeddings != nil {
		m.Embeddings = embeddings.Slice()
	}

	if len(embeddingsInfo) > 0 {
		var info models.EmbeddingsInfo
		if err := json.Unmarshal(embeddingsInfo, &info); err != nil {
			return nil, err
		}
		m.EmbeddingsInfo = &info
	}

	if userRating.Valid {
		rating := int(userRating.Int32)
		m.UserRating = &rating
	}

	if createdBy.Valid {
		m.CreatedBy = createdBy.String
	}

	if sourceType.Valid {
		m.SourceType = sourceType.String
	}

	if len(sourceInfo) > 0 {
		var info models.SourceInfo
		if err := json.Unmarshal(sourceInfo, &info); err != nil {
			return nil, err
		}
		m.SourceInfo = &info
	}

	return &m, nil
}

func (r *MemoryRepository) scanMemories(rows pgx.Rows) ([]*models.Memory, error) {
	var memories []*models.Memory

	for rows.Next() {
		var m models.Memory
		var embeddings *pgvector.Vector
		var embeddingsInfo, sourceInfo []byte
		var userRating sql.NullInt32
		var createdBy, sourceType sql.NullString

		err := rows.Scan(
			&m.ID,
			&m.Content,
			&embeddings,
			&embeddingsInfo,
			&m.Importance,
			&m.Confidence,
			&userRating,
			&createdBy,
			&sourceType,
			&sourceInfo,
			&m.Tags,
			&m.Pinned,
			&m.Archived,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.DeletedAt,
		)
		if err != nil {
			return nil, err
		}

		if embeddings != nil {
			m.Embeddings = embeddings.Slice()
		}

		if len(embeddingsInfo) > 0 {
			var info models.EmbeddingsInfo
			if err := json.Unmarshal(embeddingsInfo, &info); err != nil {
				return nil, fmt.Errorf("failed to unmarshal embeddings info: %w", err)
			}
			m.EmbeddingsInfo = &info
		}

		if userRating.Valid {
			rating := int(userRating.Int32)
			m.UserRating = &rating
		}

		if createdBy.Valid {
			m.CreatedBy = createdBy.String
		}

		if sourceType.Valid {
			m.SourceType = sourceType.String
		}

		if len(sourceInfo) > 0 {
			var info models.SourceInfo
			if err := json.Unmarshal(sourceInfo, &info); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source info: %w", err)
			}
			m.SourceInfo = &info
		}

		memories = append(memories, &m)
	}

	return memories, rows.Err()
}

// Pin sets the pinned status of a memory
func (r *MemoryRepository) Pin(ctx context.Context, id string, pinned bool) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_memory
		SET pinned = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id, pinned)
	return err
}

// Archive sets the archived status of a memory
func (r *MemoryRepository) Archive(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_memory
		SET archived = TRUE,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, id)
	return err
}
