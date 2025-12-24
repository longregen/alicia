package services

import (
	"context"
	"fmt"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type ConversationService struct {
	repo        ports.ConversationRepository
	livekit     ports.LiveKitService
	idGenerator ports.IDGenerator
}

func NewConversationService(
	repo ports.ConversationRepository,
	livekit ports.LiveKitService,
	idGenerator ports.IDGenerator,
) *ConversationService {
	return &ConversationService{
		repo:        repo,
		livekit:     livekit,
		idGenerator: idGenerator,
	}
}

func (s *ConversationService) Create(ctx context.Context, userID, title string) (*models.Conversation, error) {
	if err := ValidateRequired(title, "conversation title"); err != nil {
		return nil, err
	}

	id := s.idGenerator.GenerateConversationID()
	conversation := models.NewConversation(id, userID, title)

	if err := s.repo.Create(ctx, conversation); err != nil {
		return nil, domain.NewDomainError(err, "failed to create conversation")
	}

	return conversation, nil
}

func (s *ConversationService) CreateWithLiveKit(ctx context.Context, userID, title string) (*models.Conversation, error) {
	if err := ValidateRequired(title, "conversation title"); err != nil {
		return nil, err
	}

	id := s.idGenerator.GenerateConversationID()
	conversation := models.NewConversation(id, userID, title)

	roomName := fmt.Sprintf("conv_%s", id)

	// Try to create room, but handle case where it might already exist
	room, err := s.livekit.CreateRoom(ctx, roomName)
	if err != nil {
		// Room creation failed - check if room already exists from previous conversation
		existingRoom, getErr := s.livekit.GetRoom(ctx, roomName)
		if getErr != nil {
			// Both create and get failed - LiveKit unavailable
			return nil, domain.NewDomainError(domain.ErrLiveKitUnavailable, "failed to create LiveKit room")
		}
		// Room already exists (possibly from cleanup failure), reuse it
		room = existingRoom
	}

	conversation.SetLiveKitRoom(room.Name)

	if err := s.repo.Create(ctx, conversation); err != nil {
		// INTENTIONAL ERROR SWALLOWING: Best-effort cleanup during error handling.
		// We're already in a failure path (conversation creation failed), so we attempt
		// to delete the LiveKit room to avoid orphaned resources. If room deletion fails,
		// we don't want to mask the original error with a cleanup error. The orphaned room
		// can be cleaned up by periodic maintenance tasks.
		_ = s.livekit.DeleteRoom(ctx, room.Name)
		return nil, domain.NewDomainError(err, "failed to create conversation")
	}

	return conversation, nil
}

func (s *ConversationService) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if err := ValidateID(id, "conversation"); err != nil {
		return nil, err
	}

	conversation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	if err := ValidateNotDeleted(conversation.DeletedAt, "conversation"); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	if err := ValidateRequired(roomName, "room name"); err != nil {
		return nil, err
	}

	conversation, err := s.repo.GetByLiveKitRoom(ctx, roomName)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found for room")
	}

	if err := ValidateNotDeleted(conversation.DeletedAt, "conversation"); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) Update(ctx context.Context, conversation *models.Conversation) error {
	if conversation == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "conversation cannot be nil")
	}

	if err := ValidateID(conversation.ID, "conversation"); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, conversation.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	if err := ValidateNotDeleted(existing.DeletedAt, "conversation"); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, conversation); err != nil {
		return domain.NewDomainError(err, "failed to update conversation")
	}

	return nil
}

func (s *ConversationService) UpdateTitle(ctx context.Context, id, title string) (*models.Conversation, error) {
	if err := ValidateRequired(title, "conversation title"); err != nil {
		return nil, err
	}

	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	conversation.Title = title
	if err := s.Update(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) UpdatePreferences(ctx context.Context, id string, preferences *models.ConversationPreferences) (*models.Conversation, error) {
	if preferences == nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "preferences cannot be nil")
	}

	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	conversation.Preferences = preferences
	if err := s.Update(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) Archive(ctx context.Context, id string) (*models.Conversation, error) {
	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Archive() now validates state transitions
	if err := conversation.Archive(); err != nil {
		return nil, domain.NewDomainError(err, "cannot archive conversation")
	}

	if err := s.Update(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) Unarchive(ctx context.Context, id string) (*models.Conversation, error) {
	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Unarchive() validates state transitions
	if err := conversation.Unarchive(); err != nil {
		return nil, domain.NewDomainError(err, "cannot unarchive conversation")
	}

	if err := s.Update(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) Delete(ctx context.Context, id string) error {
	conversation, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate the state transition before deletion
	if err := conversation.MarkAsDeleted(); err != nil {
		return domain.NewDomainError(err, "cannot delete conversation")
	}

	// Use repository's Delete method for soft delete
	if err := s.repo.Delete(ctx, conversation.ID); err != nil {
		return domain.NewDomainError(err, "failed to delete conversation")
	}

	return nil
}

func (s *ConversationService) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	conversations, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list conversations")
	}

	return conversations, nil
}

func (s *ConversationService) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	conversations, err := s.repo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list active conversations")
	}

	return conversations, nil
}

func (s *ConversationService) AssociateLiveKitRoom(ctx context.Context, conversationID, roomName string) (*models.Conversation, error) {
	conversation, err := s.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	_, err = s.livekit.GetRoom(ctx, roomName)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrRoomNotFound, "LiveKit room not found")
	}

	conversation.SetLiveKitRoom(roomName)
	if err := s.Update(ctx, conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

func (s *ConversationService) GenerateLiveKitToken(ctx context.Context, conversationID, participantID, participantName string) (*ports.LiveKitToken, error) {
	conversation, err := s.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation.LiveKitRoomName == "" {
		return nil, domain.NewDomainError(domain.ErrRoomNotFound, "conversation has no associated LiveKit room")
	}

	token, err := s.livekit.GenerateToken(ctx, conversation.LiveKitRoomName, participantID, participantName)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrLiveKitUnavailable, "failed to generate LiveKit token")
	}

	return token, nil
}
