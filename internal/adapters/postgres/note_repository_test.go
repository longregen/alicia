package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/pashagolub/pgxmock/v4"
)

func TestNoteRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	note := &models.Note{
		ID:        "note_1",
		MessageID: "msg_1",
		Content:   "Test note",
		Category:  models.NoteCategoryGeneral,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO alicia_notes").
		WithArgs(note.ID, note.MessageID, note.Content, note.Category, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, note)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_Create_EmptyCategory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	note := &models.Note{
		ID:        "note_2",
		MessageID: "msg_2",
		Content:   "Test note",
		Category:  "", // Empty category
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Should default to general category
	mock.ExpectExec("INSERT INTO alicia_notes").
		WithArgs(note.ID, note.MessageID, note.Content, models.NoteCategoryGeneral, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := setupMockContext(mock)
	err = repo.Create(ctx, note)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	noteID := "note_1"
	newContent := "Updated content"

	mock.ExpectExec("UPDATE alicia_notes").
		WithArgs(newContent, noteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := setupMockContext(mock)
	err = repo.Update(ctx, noteID, newContent)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_Update_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	noteID := "nonexistent"
	newContent := "Updated content"

	mock.ExpectExec("UPDATE alicia_notes").
		WithArgs(newContent, noteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	ctx := setupMockContext(mock)
	err = repo.Update(ctx, noteID, newContent)
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	noteID := "note_1"

	mock.ExpectExec("UPDATE alicia_notes").
		WithArgs(noteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	ctx := setupMockContext(mock)
	err = repo.Delete(ctx, noteID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_GetByMessage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	messageID := "msg_1"
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "message_id", "content", "category", "created_at", "updated_at"}).
		AddRow("note_1", messageID, "Note 1", models.NoteCategoryGeneral, now, now).
		AddRow("note_2", messageID, "Note 2", models.NoteCategoryImprovement, now, now)

	mock.ExpectQuery("SELECT (.+) FROM alicia_notes").
		WithArgs(messageID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	notes, err := repo.GetByMessage(ctx, messageID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}

	if notes[0].ID != "note_1" {
		t.Errorf("expected note ID note_1, got %s", notes[0].ID)
	}

	if notes[1].Category != models.NoteCategoryImprovement {
		t.Errorf("expected category improvement, got %s", notes[1].Category)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_GetByMessage_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	messageID := "msg_empty"

	rows := pgxmock.NewRows([]string{"id", "message_id", "content", "category", "created_at", "updated_at"})

	mock.ExpectQuery("SELECT (.+) FROM alicia_notes").
		WithArgs(messageID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	notes, err := repo.GetByMessage(ctx, messageID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	noteID := "note_1"
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "message_id", "content", "category", "created_at", "updated_at"}).
		AddRow(noteID, "msg_1", "Test note", models.NoteCategoryGeneral, now, now)

	mock.ExpectQuery("SELECT (.+) FROM alicia_notes").
		WithArgs(noteID).
		WillReturnRows(rows)

	ctx := setupMockContext(mock)
	note, err := repo.GetByID(ctx, noteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if note.ID != noteID {
		t.Errorf("expected note ID %s, got %s", noteID, note.ID)
	}

	if note.Content != "Test note" {
		t.Errorf("expected content 'Test note', got %s", note.Content)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestNoteRepository_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	repo := &NoteRepository{
		BaseRepository: BaseRepository{pool: nil},
	}

	noteID := "nonexistent"

	mock.ExpectQuery("SELECT (.+) FROM alicia_notes").
		WithArgs(noteID).
		WillReturnError(pgx.ErrNoRows)

	ctx := setupMockContext(mock)
	_, err = repo.GetByID(ctx, noteID)
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
