package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
	"github.com/pashagolub/pgxmock/v4"
)

func TestOptimizationRepository_CreateRun(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	now := time.Now()
	run := &models.OptimizationRun{
		ID:         "run_1",
		Name:       "test_signature",
		PromptType: "system",
		Status:     models.OptimizationStatusRunning,
		Config: map[string]any{
			"max_iterations": float64(10), // JSON unmarshals numbers as float64
		},
		BestScore:  0.0,
		Iterations: 0,
		CreatedAt:  now,
	}

	mock.ExpectExec("INSERT INTO prompt_optimization_runs").
		WithArgs(
			run.ID, run.Name, run.Status, pgxmock.AnyArg(),
			run.BestScore, run.Iterations, pgxmock.AnyArg(), run.CompletedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.CreateRun(ctx, run)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetRun(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	runID := "run_1"
	now := time.Now()
	config := map[string]any{
		"max_iterations": float64(10),
		"prompt_type":    "system",
	}
	configJSON, _ := json.Marshal(config)

	rows := pgxmock.NewRows([]string{
		"id", "signature_name", "status", "config", "best_score", "iterations", "created_at", "completed_at",
	}).
		AddRow(runID, "test_sig", models.OptimizationStatusCompleted, configJSON,
			sql.NullFloat64{Float64: 0.95, Valid: true}, 5, now, sql.NullTime{Time: now, Valid: true})

	mock.ExpectQuery("SELECT (.+) FROM prompt_optimization_runs").
		WithArgs(runID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	run, err := repo.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run.ID != runID {
		t.Errorf("expected ID %s, got %s", runID, run.ID)
	}

	if run.Name != "test_sig" {
		t.Errorf("expected name test_sig, got %s", run.Name)
	}

	if run.PromptType != "system" {
		t.Errorf("expected prompt type system, got %s", run.PromptType)
	}

	if run.BestScore != 0.95 {
		t.Errorf("expected best score 0.95, got %f", run.BestScore)
	}

	if run.Iterations != 5 {
		t.Errorf("expected 5 iterations, got %d", run.Iterations)
	}

	if run.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	if run.Meta == nil {
		t.Error("expected Meta to be initialized")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetRun_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	mock.ExpectQuery("SELECT (.+) FROM prompt_optimization_runs").
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	ctx := setupMockContext(mock)
	_, err = repo.GetRun(ctx, "nonexistent")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_UpdateRun(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	now := time.Now()
	run := &models.OptimizationRun{
		ID:         "run_1",
		Name:       "test_sig",
		PromptType: "user",
		Status:     models.OptimizationStatusCompleted,
		Config: map[string]any{
			"max_iterations": float64(20),
		},
		BestScore:   0.92,
		Iterations:  10,
		CompletedAt: &now,
	}

	mock.ExpectExec("UPDATE prompt_optimization_runs").
		WithArgs(run.Status, pgxmock.AnyArg(), run.BestScore, run.Iterations, run.CompletedAt, run.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := setupMockContext(mock)
	err = repo.UpdateRun(ctx, run)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_UpdateRun_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	run := &models.OptimizationRun{
		ID:     "nonexistent",
		Status: models.OptimizationStatusCompleted,
		Config: map[string]any{},
	}

	mock.ExpectExec("UPDATE prompt_optimization_runs").
		WithArgs(run.Status, pgxmock.AnyArg(), run.BestScore, run.Iterations, run.CompletedAt, run.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	ctx := setupMockContext(mock)
	err = repo.UpdateRun(ctx, run)
	if err == nil {
		t.Error("expected error for not found, got nil")
	}

	expectedErr := "optimization run not found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_ListRuns(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	now := time.Now()
	config1 := map[string]any{"max_iterations": float64(10), "prompt_type": "system"}
	config2 := map[string]any{"max_iterations": float64(20), "prompt_type": "user"}
	configJSON1, _ := json.Marshal(config1)
	configJSON2, _ := json.Marshal(config2)

	rows := pgxmock.NewRows([]string{
		"id", "signature_name", "status", "config", "best_score", "iterations", "created_at", "completed_at",
	}).
		AddRow("run_1", "sig1", models.OptimizationStatusCompleted, configJSON1,
			sql.NullFloat64{Float64: 0.95, Valid: true}, 5, now, sql.NullTime{Time: now, Valid: true}).
		AddRow("run_2", "sig2", models.OptimizationStatusRunning, configJSON2,
			sql.NullFloat64{}, 2, now, sql.NullTime{})

	opts := ports.ListOptimizationRunsOptions{
		Limit:  50,
		Offset: 0,
	}

	mock.ExpectQuery("SELECT (.+) FROM prompt_optimization_runs").
		WithArgs(opts.Limit, opts.Offset).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	runs, err := repo.ListRuns(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}

	if runs[0].ID != "run_1" {
		t.Errorf("expected first run ID run_1, got %s", runs[0].ID)
	}

	if runs[0].PromptType != "system" {
		t.Errorf("expected prompt type system, got %s", runs[0].PromptType)
	}

	if runs[1].Status != models.OptimizationStatusRunning {
		t.Errorf("expected status running, got %s", runs[1].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_ListRuns_WithStatusFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	now := time.Now()
	config := map[string]any{"max_iterations": float64(10), "prompt_type": "system"}
	configJSON, _ := json.Marshal(config)

	rows := pgxmock.NewRows([]string{
		"id", "signature_name", "status", "config", "best_score", "iterations", "created_at", "completed_at",
	}).
		AddRow("run_1", "sig1", models.OptimizationStatusCompleted, configJSON,
			sql.NullFloat64{Float64: 0.95, Valid: true}, 5, now, sql.NullTime{Time: now, Valid: true})

	opts := ports.ListOptimizationRunsOptions{
		Status: models.OptimizationStatusCompleted,
		Limit:  50,
		Offset: 0,
	}

	mock.ExpectQuery("SELECT (.+) FROM prompt_optimization_runs").
		WithArgs(opts.Status, opts.Limit, opts.Offset).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	runs, err := repo.ListRuns(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_SaveCandidate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	candidate := &models.PromptCandidate{
		ID:         "cand_1",
		RunID:      "run_1",
		PromptText: "You are a helpful assistant",
		PromptVariables: map[string]any{
			"examples": []string{"demo1", "demo2"},
		},
		Score:           0.9,
		EvaluationCount: 5,
		Iteration:       2,
		CreatedAt:       time.Now(),
	}

	mock.ExpectExec("INSERT INTO prompt_candidates").
		WithArgs(
			candidate.ID, candidate.RunID, candidate.PromptText, pgxmock.AnyArg(),
			candidate.EvaluationCount, candidate.Score, candidate.Iteration, pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.SaveCandidate(ctx, candidate.RunID, candidate)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetCandidates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	runID := "run_1"
	now := time.Now()

	demos1 := map[string]any{"examples": []any{"demo1"}}
	demos2 := map[string]any{"examples": []any{"demo2"}}
	demosJSON1, _ := json.Marshal(demos1)
	demosJSON2, _ := json.Marshal(demos2)

	rows := pgxmock.NewRows([]string{
		"id", "run_id", "instructions", "demos", "coverage", "avg_score", "generation", "created_at",
	}).
		AddRow("cand_1", runID, "Prompt 1", demosJSON1, 5, sql.NullFloat64{Float64: 0.9, Valid: true}, 2, now).
		AddRow("cand_2", runID, "Prompt 2", demosJSON2, 3, sql.NullFloat64{Float64: 0.85, Valid: true}, 1, now)

	mock.ExpectQuery("SELECT (.+) FROM prompt_candidates").
		WithArgs(runID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	candidates, err := repo.GetCandidates(ctx, runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(candidates))
	}

	if candidates[0].ID != "cand_1" {
		t.Errorf("expected first candidate ID cand_1, got %s", candidates[0].ID)
	}

	if candidates[0].Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", candidates[0].Score)
	}

	if candidates[1].Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", candidates[1].Iteration)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetBestCandidate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	runID := "run_1"
	now := time.Now()
	demos := map[string]any{"examples": []any{"demo1"}}
	demosJSON, _ := json.Marshal(demos)

	rows := pgxmock.NewRows([]string{
		"id", "run_id", "instructions", "demos", "coverage", "avg_score", "generation", "created_at",
	}).
		AddRow("cand_best", runID, "Best prompt", demosJSON, 10, sql.NullFloat64{Float64: 0.98, Valid: true}, 5, now)

	mock.ExpectQuery("SELECT (.+) FROM prompt_candidates").
		WithArgs(runID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	candidate, err := repo.GetBestCandidate(ctx, runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if candidate.ID != "cand_best" {
		t.Errorf("expected candidate ID cand_best, got %s", candidate.ID)
	}

	if candidate.Score != 0.98 {
		t.Errorf("expected score 0.98, got %f", candidate.Score)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetBestCandidate_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	mock.ExpectQuery("SELECT (.+) FROM prompt_candidates").
		WithArgs("run_empty").
		WillReturnError(pgx.ErrNoRows)

	ctx := setupMockContext(mock)
	_, err = repo.GetBestCandidate(ctx, "run_empty")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_SaveEvaluation(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	eval := &models.PromptEvaluation{
		ID:          "eval_1",
		CandidateID: "cand_1",
		Input:       "example_1",
		Score:       0.95,
		Error:       "Some feedback",
		Metrics: map[string]any{
			"accuracy": 0.95,
			"latency":  float64(100),
		},
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO prompt_evaluations").
		WithArgs(
			eval.ID, eval.CandidateID, eval.Input, eval.Score, eval.Error, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.SaveEvaluation(ctx, eval)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestOptimizationRepository_GetEvaluations(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &OptimizationRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	candidateID := "cand_1"
	now := time.Now()

	metrics1 := map[string]any{"accuracy": 0.9}
	metrics2 := map[string]any{"accuracy": 0.85}
	metricsJSON1, _ := json.Marshal(metrics1)
	metricsJSON2, _ := json.Marshal(metrics2)

	rows := pgxmock.NewRows([]string{
		"id", "candidate_id", "example_id", "score", "feedback", "trace", "created_at",
	}).
		AddRow("eval_1", candidateID, "ex_1", 0.9, sql.NullString{String: "Good", Valid: true}, metricsJSON1, now).
		AddRow("eval_2", candidateID, "ex_2", 0.85, sql.NullString{}, metricsJSON2, now)

	mock.ExpectQuery("SELECT (.+) FROM prompt_evaluations").
		WithArgs(candidateID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	evaluations, err := repo.GetEvaluations(ctx, candidateID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(evaluations) != 2 {
		t.Errorf("expected 2 evaluations, got %d", len(evaluations))
	}

	if evaluations[0].ID != "eval_1" {
		t.Errorf("expected first evaluation ID eval_1, got %s", evaluations[0].ID)
	}

	if evaluations[0].Error != "Good" {
		t.Errorf("expected feedback 'Good', got %s", evaluations[0].Error)
	}

	if evaluations[1].Error != "" {
		t.Errorf("expected empty feedback, got %s", evaluations[1].Error)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
