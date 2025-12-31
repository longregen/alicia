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
)

// OptimizationRepository implements ports.PromptOptimizationRepository
type OptimizationRepository struct {
	BaseRepository
}

// NewOptimizationRepository creates a new optimization repository
func NewOptimizationRepository(pool *pgxpool.Pool) *OptimizationRepository {
	return &OptimizationRepository{
		BaseRepository: NewBaseRepository(pool),
	}
}

// CreateRun creates a new optimization run
func (r *OptimizationRepository) CreateRun(ctx context.Context, run *models.OptimizationRun) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Store both name and prompt_type in config
	configCopy := make(map[string]any)
	for k, v := range run.Config {
		configCopy[k] = v
	}
	configCopy["prompt_type"] = run.PromptType

	config, err := json.Marshal(configCopy)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO prompt_optimization_runs (
			id, signature_name, status, config, best_score, iterations, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		run.ID,
		run.Name,
		run.Status,
		config,
		run.BestScore,
		run.Iterations,
		run.CreatedAt,
		run.CompletedAt,
	)

	return err
}

// GetRun retrieves an optimization run by ID
func (r *OptimizationRepository) GetRun(ctx context.Context, id string) (*models.OptimizationRun, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, signature_name, status, config, best_score, iterations, created_at, completed_at
		FROM prompt_optimization_runs
		WHERE id = $1 AND deleted_at IS NULL`

	return r.scanRun(r.conn(ctx).QueryRow(ctx, query, id))
}

// UpdateRun updates an existing optimization run
func (r *OptimizationRepository) UpdateRun(ctx context.Context, run *models.OptimizationRun) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Store both name and prompt_type in config
	configCopy := make(map[string]any)
	for k, v := range run.Config {
		configCopy[k] = v
	}
	configCopy["prompt_type"] = run.PromptType

	config, err := json.Marshal(configCopy)
	if err != nil {
		return err
	}

	query := `
		UPDATE prompt_optimization_runs
		SET status = $1, config = $2, best_score = $3, iterations = $4, completed_at = $5
		WHERE id = $6 AND deleted_at IS NULL`

	result, err := r.conn(ctx).Exec(ctx, query,
		run.Status,
		config,
		run.BestScore,
		run.Iterations,
		run.CompletedAt,
		run.ID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("optimization run not found")
	}

	return nil
}

// ListRuns retrieves optimization runs with optional filtering and pagination
func (r *OptimizationRepository) ListRuns(ctx context.Context, opts ports.ListOptimizationRunsOptions) ([]*models.OptimizationRun, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	// Set defaults
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200 // Maximum cap
	}

	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Build query with optional status filter
	query := `
		SELECT id, signature_name, status, config, best_score, iterations, created_at, completed_at
		FROM prompt_optimization_runs
		WHERE deleted_at IS NULL`

	args := []interface{}{}
	argPos := 1

	if opts.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, opts.Status)
		argPos++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRuns(rows)
}

func (r *OptimizationRepository) scanRuns(rows pgx.Rows) ([]*models.OptimizationRun, error) {
	runs := make([]*models.OptimizationRun, 0)

	for rows.Next() {
		var run models.OptimizationRun
		var config []byte
		var bestScore sql.NullFloat64
		var completedAt sql.NullTime

		err := rows.Scan(
			&run.ID,
			&run.Name,
			&run.Status,
			&config,
			&bestScore,
			&run.Iterations,
			&run.CreatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(config) > 0 {
			if err := json.Unmarshal(config, &run.Config); err != nil {
				run.Config = make(map[string]any)
			}
		} else {
			run.Config = make(map[string]any)
		}

		// Extract prompt_type from config
		if promptType, ok := run.Config["prompt_type"].(string); ok {
			run.PromptType = promptType
		}

		if bestScore.Valid {
			run.BestScore = bestScore.Float64
		}

		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}

		run.Meta = make(map[string]any)
		run.StartedAt = run.CreatedAt
		run.UpdatedAt = run.CreatedAt

		runs = append(runs, &run)
	}

	return runs, rows.Err()
}

// SaveCandidate saves a prompt candidate
func (r *OptimizationRepository) SaveCandidate(ctx context.Context, runID string, candidate *models.PromptCandidate) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	demos, err := json.Marshal(candidate.PromptVariables)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO prompt_candidates (
			id, run_id, instructions, demos, coverage, avg_score, generation, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (id) DO UPDATE SET
			avg_score = EXCLUDED.avg_score,
			coverage = EXCLUDED.coverage`

	_, err = r.conn(ctx).Exec(ctx, query,
		candidate.ID,
		runID,
		candidate.PromptText,
		demos,
		candidate.EvaluationCount,
		candidate.Score,
		candidate.Iteration,
		candidate.CreatedAt,
	)

	return err
}

// GetCandidates retrieves all candidates for a run
func (r *OptimizationRepository) GetCandidates(ctx context.Context, runID string) ([]*models.PromptCandidate, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, run_id, instructions, demos, coverage, avg_score, generation, created_at
		FROM prompt_candidates
		WHERE run_id = $1 AND deleted_at IS NULL
		ORDER BY generation DESC, avg_score DESC NULLS LAST`

	rows, err := r.conn(ctx).Query(ctx, query, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanCandidates(rows)
}

// GetBestCandidate retrieves the best candidate for a run
func (r *OptimizationRepository) GetBestCandidate(ctx context.Context, runID string) (*models.PromptCandidate, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, run_id, instructions, demos, coverage, avg_score, generation, created_at
		FROM prompt_candidates
		WHERE run_id = $1 AND deleted_at IS NULL
		ORDER BY avg_score DESC NULLS LAST
		LIMIT 1`

	return r.scanCandidate(r.conn(ctx).QueryRow(ctx, query, runID))
}

// SaveEvaluation saves a prompt evaluation
func (r *OptimizationRepository) SaveEvaluation(ctx context.Context, eval *models.PromptEvaluation) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	trace, err := json.Marshal(eval.Metrics)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO prompt_evaluations (
			id, candidate_id, example_id, score, feedback, trace, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err = r.conn(ctx).Exec(ctx, query,
		eval.ID,
		eval.CandidateID,
		eval.Input, // Using input as example_id
		eval.Score,
		eval.Error, // Using error field as feedback
		trace,
		eval.CreatedAt,
	)

	return err
}

// GetEvaluations retrieves all evaluations for a candidate
func (r *OptimizationRepository) GetEvaluations(ctx context.Context, candidateID string) ([]*models.PromptEvaluation, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, candidate_id, example_id, score, feedback, trace, created_at
		FROM prompt_evaluations
		WHERE candidate_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.conn(ctx).Query(ctx, query, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvaluations(rows)
}

func (r *OptimizationRepository) scanRun(row pgx.Row) (*models.OptimizationRun, error) {
	var run models.OptimizationRun
	var config []byte
	var bestScore sql.NullFloat64
	var completedAt sql.NullTime

	err := row.Scan(
		&run.ID,
		&run.Name,
		&run.Status,
		&config,
		&bestScore,
		&run.Iterations,
		&run.CreatedAt,
		&completedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	if len(config) > 0 {
		if err := json.Unmarshal(config, &run.Config); err != nil {
			run.Config = make(map[string]any)
		}
	} else {
		run.Config = make(map[string]any)
	}

	// Extract prompt_type from config
	if promptType, ok := run.Config["prompt_type"].(string); ok {
		run.PromptType = promptType
	}

	if bestScore.Valid {
		run.BestScore = bestScore.Float64
	}

	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	run.Meta = make(map[string]any)
	run.StartedAt = run.CreatedAt
	run.UpdatedAt = run.CreatedAt

	return &run, nil
}

func (r *OptimizationRepository) scanCandidate(row pgx.Row) (*models.PromptCandidate, error) {
	var candidate models.PromptCandidate
	var demos []byte
	var avgScore sql.NullFloat64

	err := row.Scan(
		&candidate.ID,
		&candidate.RunID,
		&candidate.PromptText,
		&demos,
		&candidate.EvaluationCount,
		&avgScore,
		&candidate.Iteration,
		&candidate.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	if len(demos) > 0 {
		if err := json.Unmarshal(demos, &candidate.PromptVariables); err != nil {
			candidate.PromptVariables = make(map[string]any)
		}
	} else {
		candidate.PromptVariables = make(map[string]any)
	}

	if avgScore.Valid {
		candidate.Score = avgScore.Float64
	}

	candidate.Meta = make(map[string]any)
	candidate.UpdatedAt = candidate.CreatedAt

	return &candidate, nil
}

func (r *OptimizationRepository) scanCandidates(rows pgx.Rows) ([]*models.PromptCandidate, error) {
	candidates := make([]*models.PromptCandidate, 0)

	for rows.Next() {
		var candidate models.PromptCandidate
		var demos []byte
		var avgScore sql.NullFloat64

		err := rows.Scan(
			&candidate.ID,
			&candidate.RunID,
			&candidate.PromptText,
			&demos,
			&candidate.EvaluationCount,
			&avgScore,
			&candidate.Iteration,
			&candidate.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(demos) > 0 {
			if err := json.Unmarshal(demos, &candidate.PromptVariables); err != nil {
				candidate.PromptVariables = make(map[string]any)
			}
		} else {
			candidate.PromptVariables = make(map[string]any)
		}

		if avgScore.Valid {
			candidate.Score = avgScore.Float64
		}

		candidate.Meta = make(map[string]any)
		candidate.UpdatedAt = candidate.CreatedAt

		candidates = append(candidates, &candidate)
	}

	return candidates, rows.Err()
}

func (r *OptimizationRepository) scanEvaluations(rows pgx.Rows) ([]*models.PromptEvaluation, error) {
	evaluations := make([]*models.PromptEvaluation, 0)

	for rows.Next() {
		var eval models.PromptEvaluation
		var feedback sql.NullString
		var trace []byte

		err := rows.Scan(
			&eval.ID,
			&eval.CandidateID,
			&eval.Input, // example_id -> Input
			&eval.Score,
			&feedback,
			&trace,
			&eval.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if feedback.Valid {
			eval.Error = feedback.String
		}

		if len(trace) > 0 {
			if err := json.Unmarshal(trace, &eval.Metrics); err != nil {
				eval.Metrics = make(map[string]any)
			}
		} else {
			eval.Metrics = make(map[string]any)
		}

		evaluations = append(evaluations, &eval)
	}

	return evaluations, rows.Err()
}
