package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/longregen/alicia/api/domain"
	"github.com/longregen/alicia/api/store"
)

const maxEmbeddingChars = 32000

func noteEmbeddingText(title, content string) string {
	text := title + "\n" + content
	if len(text) > maxEmbeddingChars {
		text = text[:maxEmbeddingChars]
	}
	return text
}

type NoteService struct {
	store    *store.Store
	embedder EmbeddingService
}

func NewNoteService(s *store.Store, embedder EmbeddingService) *NoteService {
	return &NoteService{store: s, embedder: embedder}
}

func (svc *NoteService) generateEmbedding(ctx context.Context, noteID, title, content string) []float32 {
	if svc.embedder == nil || content == "" {
		return nil
	}
	embedding, err := svc.embedder.Embed(ctx, noteEmbeddingText(title, content))
	if err != nil {
		slog.Warn("embedding generation failed for note", "note_id", noteID, "error", err)
		return nil
	}
	return embedding
}

func (svc *NoteService) Create(ctx context.Context, id, userID, title, content string) (*domain.Note, error) {
	now := time.Now().UTC()
	note := &domain.Note{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Content:   content,
		Embedding: svc.generateEmbedding(ctx, id, title, content),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := svc.store.CreateNote(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (svc *NoteService) Get(ctx context.Context, id string) (*domain.Note, error) {
	return svc.store.GetNote(ctx, id)
}

func (svc *NoteService) List(ctx context.Context, userID string) ([]*domain.Note, error) {
	return svc.store.ListNotesByUser(ctx, userID)
}

func (svc *NoteService) Update(ctx context.Context, note *domain.Note) error {
	note.UpdatedAt = time.Now().UTC()
	note.Embedding = svc.generateEmbedding(ctx, note.ID, note.Title, note.Content)
	return svc.store.UpdateNote(ctx, note)
}

func (svc *NoteService) Delete(ctx context.Context, id string) error {
	return svc.store.DeleteNote(ctx, id)
}

