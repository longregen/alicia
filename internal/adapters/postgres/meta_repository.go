package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaRepository struct {
	BaseRepository
}

func NewMetaRepository(pool *pgxpool.Pool) *MetaRepository {
	return &MetaRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

func (r *MetaRepository) Set(ctx context.Context, ref, key, value string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	now := time.Now()

	query := `
		INSERT INTO alicia_meta (id, ref, key, value, created_at, updated_at)
		VALUES (generate_random_id('amt'), $1, $2, $3, $4, $5)
		ON CONFLICT (ref, key) WHERE deleted_at IS NULL
		DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`

	_, err := r.conn(ctx).Exec(ctx, query, ref, key, value, now, now)
	return err
}

func (r *MetaRepository) Get(ctx context.Context, ref, key string) (string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT value
		FROM alicia_meta
		WHERE ref = $1 AND key = $2 AND deleted_at IS NULL`

	var value sql.NullString
	err := r.conn(ctx).QueryRow(ctx, query, ref, key).Scan(&value)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// For meta repository, returning empty string with nil error is appropriate
			// as it represents "key not found" which is a valid state for metadata lookups
			return "", nil
		}
		return "", err
	}

	if value.Valid {
		return value.String, nil
	}

	return "", nil
}

func (r *MetaRepository) GetAll(ctx context.Context, ref string) (map[string]string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT key, value
		FROM alicia_meta
		WHERE ref = $1 AND deleted_at IS NULL
		ORDER BY key ASC`

	rows, err := r.conn(ctx).Query(ctx, query, ref)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)

	for rows.Next() {
		var key string
		var value sql.NullString

		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}

		if value.Valid {
			result[key] = value.String
		} else {
			result[key] = ""
		}
	}

	return result, rows.Err()
}

func (r *MetaRepository) Delete(ctx context.Context, ref, key string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_meta
		SET deleted_at = NOW()
		WHERE ref = $1 AND key = $2 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, ref, key)
	return err
}

func (r *MetaRepository) DeleteAll(ctx context.Context, ref string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		UPDATE alicia_meta
		SET deleted_at = NOW()
		WHERE ref = $1 AND deleted_at IS NULL`

	_, err := r.conn(ctx).Exec(ctx, query, ref)
	return err
}
