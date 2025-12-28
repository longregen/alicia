package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/pashagolub/pgxmock/v4"
)

func TestVoteRepository_Create_Message(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	vote := &models.Vote{
		ID:            "vote_1",
		MessageID:     "msg_1",
		TargetType:    "message",
		TargetID:      "msg_1",
		Value:         models.VoteValueUp,
		QuickFeedback: "helpful",
		Note:          "Great response",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mock.ExpectExec("INSERT INTO alicia_votes").
		WithArgs(
			vote.ID, vote.MessageID, vote.TargetType, vote.TargetID,
			"up", sql.NullString{String: vote.QuickFeedback, Valid: true},
			sql.NullString{String: vote.Note, Valid: true}, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, vote)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_Create_ToolUse(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	vote := &models.Vote{
		ID:         "vote_2",
		TargetType: "tool_use",
		TargetID:   "tool_1",
		Value:      models.VoteValueDown,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO alicia_votes").
		WithArgs(
			vote.ID, vote.TargetType, vote.TargetID,
			"down", sql.NullString{}, sql.NullString{}, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, vote)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_Create_Reasoning(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	vote := &models.Vote{
		ID:         "vote_3",
		TargetType: "reasoning",
		TargetID:   "reasoning_1",
		Value:      models.VoteValueUp,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO alicia_votes").
		WithArgs(
			vote.ID, vote.TargetType, vote.TargetID,
			"up", sql.NullString{}, sql.NullString{}, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, vote)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_Create_Memory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	vote := &models.Vote{
		ID:         "vote_4",
		TargetType: "memory",
		TargetID:   "mem_1",
		Value:      models.VoteValueUp,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO alicia_votes").
		WithArgs(
			vote.ID, vote.TargetType, vote.TargetID,
			"up", sql.NullString{}, sql.NullString{}, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, vote)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	targetType := "message"
	targetID := "msg_1"

	mock.ExpectExec("UPDATE alicia_votes").
		WithArgs(targetType, targetID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := setupMockContext(mock)
	err = repo.Delete(ctx, targetType, targetID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_GetByTarget(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	targetType := "message"
	targetID := "msg_1"
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "target_type", "target_id", "message_id", "vote", "quick_feedback", "note", "created_at", "updated_at",
	}).
		AddRow("vote_1", targetType, targetID, sql.NullString{String: "msg_1", Valid: true}, "up",
			sql.NullString{String: "helpful", Valid: true}, sql.NullString{}, now, now).
		AddRow("vote_2", targetType, targetID, sql.NullString{String: "msg_1", Valid: true}, "down",
			sql.NullString{}, sql.NullString{String: "needs work", Valid: true}, now, now)

	mock.ExpectQuery("SELECT (.+) FROM alicia_votes").
		WithArgs(targetType, targetID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	votes, err := repo.GetByTarget(ctx, targetType, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(votes) != 2 {
		t.Errorf("expected 2 votes, got %d", len(votes))
	}

	if votes[0].Value != models.VoteValueUp {
		t.Errorf("expected vote value up, got %d", votes[0].Value)
	}

	if votes[0].QuickFeedback != "helpful" {
		t.Errorf("expected quick feedback 'helpful', got %s", votes[0].QuickFeedback)
	}

	if votes[1].Value != models.VoteValueDown {
		t.Errorf("expected vote value down, got %d", votes[1].Value)
	}

	if votes[1].Note != "needs work" {
		t.Errorf("expected note 'needs work', got %s", votes[1].Note)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_GetByMessage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	messageID := "msg_1"
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "target_type", "target_id", "message_id", "vote", "quick_feedback", "note", "created_at", "updated_at",
	}).
		AddRow("vote_1", "message", "msg_1", sql.NullString{String: messageID, Valid: true}, "up",
			sql.NullString{}, sql.NullString{}, now, now)

	mock.ExpectQuery("SELECT (.+) FROM alicia_votes").
		WithArgs(messageID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	votes, err := repo.GetByMessage(ctx, messageID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(votes) != 1 {
		t.Errorf("expected 1 vote, got %d", len(votes))
	}

	if votes[0].MessageID != messageID {
		t.Errorf("expected message ID %s, got %s", messageID, votes[0].MessageID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_GetAggregates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	targetType := "message"
	targetID := "msg_1"

	rows := pgxmock.NewRows([]string{"target_type", "target_id", "upvotes", "downvotes"}).
		AddRow(targetType, targetID, 5, 2)

	mock.ExpectQuery("SELECT (.+) FROM alicia_votes").
		WithArgs(targetType, targetID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	aggregates, err := repo.GetAggregates(ctx, targetType, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aggregates.Upvotes != 5 {
		t.Errorf("expected 5 upvotes, got %d", aggregates.Upvotes)
	}

	if aggregates.Downvotes != 2 {
		t.Errorf("expected 2 downvotes, got %d", aggregates.Downvotes)
	}

	if aggregates.NetScore != 3 {
		t.Errorf("expected net score 3, got %d", aggregates.NetScore)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVoteRepository_GetAggregates_NoVotes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &VoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	targetType := "message"
	targetID := "msg_empty"

	mock.ExpectQuery("SELECT (.+) FROM alicia_votes").
		WithArgs(targetType, targetID).
		WillReturnError(pgx.ErrNoRows)

	ctx := setupMockContext(mock)
	aggregates, err := repo.GetAggregates(ctx, targetType, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if aggregates.Upvotes != 0 {
		t.Errorf("expected 0 upvotes, got %d", aggregates.Upvotes)
	}

	if aggregates.Downvotes != 0 {
		t.Errorf("expected 0 downvotes, got %d", aggregates.Downvotes)
	}

	if aggregates.NetScore != 0 {
		t.Errorf("expected net score 0, got %d", aggregates.NetScore)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
