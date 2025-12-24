package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type StartConversationInput struct {
	UserID          string
	Title           string
	LiveKitRoomName string
	Preferences     *models.ConversationPreferences
}

type StartConversationOutput struct {
	Conversation *models.Conversation
}

type ResumeConversationInput struct {
	ConversationID string
}

type ResumeConversationOutput struct {
	Conversation *models.Conversation
	Messages     []*models.Message
}

type ArchiveConversationInput struct {
	ConversationID string
}

type DeleteConversationInput struct {
	ConversationID string
}

type ManageConversation struct {
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	liveKitService   ports.LiveKitService
	idGenerator      ports.IDGenerator
}

func NewManageConversation(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	liveKitService ports.LiveKitService,
	idGenerator ports.IDGenerator,
) *ManageConversation {
	return &ManageConversation{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		liveKitService:   liveKitService,
		idGenerator:      idGenerator,
	}
}

func (uc *ManageConversation) StartConversation(ctx context.Context, input *StartConversationInput) (*StartConversationOutput, error) {
	conversationID := uc.idGenerator.GenerateConversationID()

	title := input.Title
	if title == "" {
		title = fmt.Sprintf("Conversation %s", time.Now().Format("2006-01-02 15:04"))
	}

	userID := input.UserID
	if userID == "" {
		userID = "default-user"
	}

	conversation := models.NewConversation(conversationID, userID, title)

	if input.Preferences != nil {
		conversation.Preferences = input.Preferences
	}

	if input.LiveKitRoomName != "" && uc.liveKitService != nil {
		// Try to get the room first to avoid race condition
		existingRoom, err := uc.liveKitService.GetRoom(ctx, input.LiveKitRoomName)
		if err != nil {
			// Room doesn't exist, create it
			_, createErr := uc.liveKitService.CreateRoom(ctx, input.LiveKitRoomName)
			if createErr != nil {
				// Creation failed, check if room was created by another request
				var getErr error
				existingRoom, getErr = uc.liveKitService.GetRoom(ctx, input.LiveKitRoomName)
				if getErr != nil {
					return nil, fmt.Errorf("failed to create or get LiveKit room: %w", createErr)
				}
				// Room exists now, use it
			}
		}
		// Room exists (either pre-existing or just created)
		if existingRoom != nil {
			conversation.SetLiveKitRoom(existingRoom.Name)
		} else {
			conversation.SetLiveKitRoom(input.LiveKitRoomName)
		}
	}

	if err := uc.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return &StartConversationOutput{
		Conversation: conversation,
	}, nil
}

func (uc *ManageConversation) ResumeConversation(ctx context.Context, input *ResumeConversationInput) (*ResumeConversationOutput, error) {
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if !conversation.IsActive() {
		return nil, fmt.Errorf("conversation is not active (status: %s)", conversation.Status)
	}

	messages, err := uc.messageRepo.GetLatestByConversation(ctx, input.ConversationID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	return &ResumeConversationOutput{
		Conversation: conversation,
		Messages:     messages,
	}, nil
}

func (uc *ManageConversation) ResumeConversationByLiveKit(ctx context.Context, roomName string) (*ResumeConversationOutput, error) {
	conversation, err := uc.conversationRepo.GetByLiveKitRoom(ctx, roomName)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation by LiveKit room: %w", err)
	}

	if !conversation.IsActive() {
		return nil, fmt.Errorf("conversation is not active (status: %s)", conversation.Status)
	}

	messages, err := uc.messageRepo.GetLatestByConversation(ctx, conversation.ID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	return &ResumeConversationOutput{
		Conversation: conversation,
		Messages:     messages,
	}, nil
}

func (uc *ManageConversation) ArchiveConversation(ctx context.Context, input *ArchiveConversationInput) error {
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Archive now validates state transitions
	if err := conversation.Archive(); err != nil {
		return fmt.Errorf("cannot archive conversation: %w", err)
	}

	if err := uc.conversationRepo.Update(ctx, conversation); err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	if conversation.LiveKitRoomName != "" && uc.liveKitService != nil {
		if err := uc.liveKitService.DeleteRoom(ctx, conversation.LiveKitRoomName); err != nil {
			// Log error but don't fail the operation
			log.Printf("warning: failed to delete LiveKit room: %v\n", err)
		}
	}

	return nil
}

// UnarchiveConversation restores an archived conversation to active status
func (uc *ManageConversation) UnarchiveConversation(ctx context.Context, conversationID string) error {
	conversation, err := uc.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Unarchive validates state transitions
	if err := conversation.Unarchive(); err != nil {
		return fmt.Errorf("cannot unarchive conversation: %w", err)
	}

	if err := uc.conversationRepo.Update(ctx, conversation); err != nil {
		return fmt.Errorf("failed to unarchive conversation: %w", err)
	}

	return nil
}

func (uc *ManageConversation) DeleteConversation(ctx context.Context, input *DeleteConversationInput) error {
	conversation, err := uc.conversationRepo.GetByID(ctx, input.ConversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conversation.LiveKitRoomName != "" && uc.liveKitService != nil {
		if err := uc.liveKitService.DeleteRoom(ctx, conversation.LiveKitRoomName); err != nil {
			// Log error but don't fail the operation
			log.Printf("warning: failed to delete LiveKit room: %v\n", err)
		}
	}

	if err := uc.conversationRepo.Delete(ctx, input.ConversationID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

func (uc *ManageConversation) ListActiveConversations(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	conversations, err := uc.conversationRepo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list active conversations: %w", err)
	}

	return conversations, nil
}

func (uc *ManageConversation) GetConversation(ctx context.Context, conversationID string) (*models.Conversation, error) {
	conversation, err := uc.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conversation, nil
}

func (uc *ManageConversation) UpdateConversationPreferences(ctx context.Context, conversationID string, preferences *models.ConversationPreferences) error {
	conversation, err := uc.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	conversation.Preferences = preferences
	conversation.UpdatedAt = time.Now()

	if err := uc.conversationRepo.Update(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation preferences: %w", err)
	}

	return nil
}
