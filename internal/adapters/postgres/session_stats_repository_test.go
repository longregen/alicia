package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/pashagolub/pgxmock/v4"
)

func TestSessionStatsRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	stats := &models.SessionStats{
		ID:                "stats_1",
		ConversationID:    "conv_1",
		MessageCount:      10,
		ToolCallCount:     5,
		MemoryRetrievals:  3,
		SessionDurationMs: 60000, // 60 seconds in milliseconds
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// SessionDurationMs (60000ms) should be converted to seconds (60s)
	mock.ExpectExec("INSERT INTO alicia_session_stats").
		WithArgs(
			stats.ID, stats.ConversationID, stats.MessageCount, stats.ToolCallCount,
			stats.MemoryRetrievals, int64(60), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, stats)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSessionStatsRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	stats := &models.SessionStats{
		ID:                "stats_1",
		ConversationID:    "conv_1",
		MessageCount:      15,
		ToolCallCount:     8,
		MemoryRetrievals:  5,
		SessionDurationMs: 120000, // 120 seconds in milliseconds
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec("UPDATE alicia_session_stats").
		WithArgs(
			stats.MessageCount, stats.ToolCallCount, stats.MemoryRetrievals,
			int64(120), pgxmock.AnyArg(), stats.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := setupMockContext(mock)
	err = repo.Update(ctx, stats)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSessionStatsRepository_Update_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	stats := &models.SessionStats{
		ID:                "nonexistent",
		ConversationID:    "conv_1",
		MessageCount:      15,
		ToolCallCount:     8,
		MemoryRetrievals:  5,
		SessionDurationMs: 120000,
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec("UPDATE alicia_session_stats").
		WithArgs(
			stats.MessageCount, stats.ToolCallCount, stats.MemoryRetrievals,
			int64(120), pgxmock.AnyArg(), stats.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	ctx := setupMockContext(mock)
	err = repo.Update(ctx, stats)
	if err == nil {
		t.Error("expected error for not found, got nil")
	}

	expectedErr := "session stats not found"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSessionStatsRepository_GetByConversation(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	conversationID := "conv_1"
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "conversation_id", "message_count", "tool_call_count",
		"memories_used", "session_duration_seconds", "created_at", "updated_at",
	}).
		AddRow("stats_1", conversationID, 10, 5, 3, int64(60), now, now)

	mock.ExpectQuery("SELECT (.+) FROM alicia_session_stats").
		WithArgs(conversationID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	stats, err := repo.GetByConversation(ctx, conversationID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.ID != "stats_1" {
		t.Errorf("expected ID stats_1, got %s", stats.ID)
	}

	if stats.ConversationID != conversationID {
		t.Errorf("expected conversation ID %s, got %s", conversationID, stats.ConversationID)
	}

	if stats.MessageCount != 10 {
		t.Errorf("expected message count 10, got %d", stats.MessageCount)
	}

	if stats.ToolCallCount != 5 {
		t.Errorf("expected tool call count 5, got %d", stats.ToolCallCount)
	}

	if stats.MemoryRetrievals != 3 {
		t.Errorf("expected memory retrievals 3, got %d", stats.MemoryRetrievals)
	}

	// Seconds (60) should be converted back to milliseconds (60000)
	if stats.SessionDurationMs != 60000 {
		t.Errorf("expected session duration 60000ms, got %d", stats.SessionDurationMs)
	}

	// Verify initialized fields
	if stats.Meta == nil {
		t.Error("expected Meta to be initialized")
	}

	if stats.UserMessageCount != 0 {
		t.Errorf("expected UserMessageCount to be 0, got %d", stats.UserMessageCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSessionStatsRepository_GetByConversation_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	conversationID := "nonexistent"

	mock.ExpectQuery("SELECT (.+) FROM alicia_session_stats").
		WithArgs(conversationID).
		WillReturnError(pgx.ErrNoRows)

	ctx := setupMockContext(mock)
	_, err = repo.GetByConversation(ctx, conversationID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSessionStatsRepository_DurationConversion(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &SessionStatsRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	// Test that milliseconds are correctly converted to seconds on create
	stats := &models.SessionStats{
		ID:                "stats_conversion",
		ConversationID:    "conv_1",
		MessageCount:      1,
		ToolCallCount:     0,
		MemoryRetrievals:  0,
		SessionDurationMs: 5500, // 5.5 seconds
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 5500ms / 1000 = 5 seconds (integer division)
	mock.ExpectExec("INSERT INTO alicia_session_stats").
		WithArgs(
			stats.ID, stats.ConversationID, stats.MessageCount, stats.ToolCallCount,
			stats.MemoryRetrievals, int64(5), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, stats)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
