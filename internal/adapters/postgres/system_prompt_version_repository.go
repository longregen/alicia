package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type SystemPromptVersionRepository struct {
	BaseRepository
	idGenerator ports.IDGenerator
}

func NewSystemPromptVersionRepository(pool *pgxpool.Pool, idGenerator ports.IDGenerator) *SystemPromptVersionRepository {
	return &SystemPromptVersionRepository{
		BaseRepository: NewBaseRepository(pool),
		idGenerator:    idGenerator,
	}
}

func (r *SystemPromptVersionRepository) Create(ctx context.Context, version *models.SystemPromptVersion) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO system_prompt_versions (
			id, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.conn(ctx).Exec(ctx, query,
		version.ID,
		version.PromptHash,
		version.PromptContent,
		version.PromptType,
		version.Description,
		version.Active,
		version.CreatedAt,
		nullTime(version.ActivatedAt),
		nullTime(version.DeactivatedAt),
	)

	return err
}

func (r *SystemPromptVersionRepository) GetByID(ctx context.Context, id string) (*models.SystemPromptVersion, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, version_number, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at, deleted_at
		FROM system_prompt_versions
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanVersion(r.conn(ctx).QueryRow(ctx, query, id))
}

func (r *SystemPromptVersionRepository) GetActiveByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, version_number, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at, deleted_at
		FROM system_prompt_versions
		WHERE prompt_type = $1 AND active = true AND deleted_at IS NULL
		LIMIT 1`

	return r.scanVersion(r.conn(ctx).QueryRow(ctx, query, promptType))
}

func (r *SystemPromptVersionRepository) GetByHash(ctx context.Context, promptType, hash string) (*models.SystemPromptVersion, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, version_number, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at, deleted_at
		FROM system_prompt_versions
		WHERE prompt_type = $1 AND prompt_hash = $2 AND deleted_at IS NULL
		LIMIT 1`

	return r.scanVersion(r.conn(ctx).QueryRow(ctx, query, promptType, hash))
}

func (r *SystemPromptVersionRepository) SetActive(ctx context.Context, id string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var promptType string
	getTypeQuery := `SELECT prompt_type FROM system_prompt_versions WHERE id = $1 AND deleted_at IS NULL`
	err := r.conn(ctx).QueryRow(ctx, getTypeQuery, id).Scan(&promptType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("system prompt version not found")
		}
		return err
	}

	deactivateQuery := `
		UPDATE system_prompt_versions
		SET active = false, deactivated_at = NOW()
		WHERE prompt_type = $1 AND active = true AND deleted_at IS NULL`

	_, err = r.conn(ctx).Exec(ctx, deactivateQuery, promptType)
	if err != nil {
		return err
	}

	activateQuery := `
		UPDATE system_prompt_versions
		SET active = true, activated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, activateQuery, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("system prompt version not found")
	}

	return nil
}

func (r *SystemPromptVersionRepository) List(ctx context.Context, promptType string, limit int) ([]*models.SystemPromptVersion, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, version_number, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at, deleted_at
		FROM system_prompt_versions
		WHERE prompt_type = $1 AND deleted_at IS NULL
		ORDER BY version_number DESC
		LIMIT $2`

	rows, err := r.conn(ctx).Query(ctx, query, promptType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanVersions(rows)
}

func (r *SystemPromptVersionRepository) GetLatestByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, version_number, prompt_hash, prompt_content, prompt_type, description, active, created_at, activated_at, deactivated_at, deleted_at
		FROM system_prompt_versions
		WHERE prompt_type = $1 AND deleted_at IS NULL
		ORDER BY version_number DESC
		LIMIT 1`

	return r.scanVersion(r.conn(ctx).QueryRow(ctx, query, promptType))
}

func (r *SystemPromptVersionRepository) scanVersion(row pgx.Row) (*models.SystemPromptVersion, error) {
	var version models.SystemPromptVersion
	var activatedAt, deactivatedAt, deletedAt sql.NullTime

	err := row.Scan(
		&version.ID,
		&version.VersionNumber,
		&version.PromptHash,
		&version.PromptContent,
		&version.PromptType,
		&version.Description,
		&version.Active,
		&version.CreatedAt,
		&activatedAt,
		&deactivatedAt,
		&deletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	version.ActivatedAt = getTimePtr(activatedAt)
	version.DeactivatedAt = getTimePtr(deactivatedAt)
	version.DeletedAt = getTimePtr(deletedAt)

	return &version, nil
}

func (r *SystemPromptVersionRepository) scanVersions(rows pgx.Rows) ([]*models.SystemPromptVersion, error) {
	versions := make([]*models.SystemPromptVersion, 0)

	for rows.Next() {
		var version models.SystemPromptVersion
		var activatedAt, deactivatedAt, deletedAt sql.NullTime

		err := rows.Scan(
			&version.ID,
			&version.VersionNumber,
			&version.PromptHash,
			&version.PromptContent,
			&version.PromptType,
			&version.Description,
			&version.Active,
			&version.CreatedAt,
			&activatedAt,
			&deactivatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, err
		}

		version.ActivatedAt = getTimePtr(activatedAt)
		version.DeactivatedAt = getTimePtr(deactivatedAt)
		version.DeletedAt = getTimePtr(deletedAt)

		versions = append(versions, &version)
	}

	return versions, rows.Err()
}
