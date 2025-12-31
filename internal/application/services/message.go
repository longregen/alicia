package services

import (
	"context"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type MessageService struct {
	messageRepo  ports.MessageRepository
	sentenceRepo ports.SentenceRepository
	convRepo     ports.ConversationRepository
	idGenerator  ports.IDGenerator
}

func NewMessageService(
	messageRepo ports.MessageRepository,
	sentenceRepo ports.SentenceRepository,
	convRepo ports.ConversationRepository,
	idGenerator ports.IDGenerator,
) *MessageService {
	return &MessageService{
		messageRepo:  messageRepo,
		sentenceRepo: sentenceRepo,
		convRepo:     convRepo,
		idGenerator:  idGenerator,
	}
}

func (s *MessageService) Create(ctx context.Context, conversationID string, role models.MessageRole, contents string) (*models.Message, error) {
	if err := ValidateID(conversationID, "conversation"); err != nil {
		return nil, err
	}

	if err := ValidateRequired(contents, "message contents"); err != nil {
		return nil, err
	}

	conversation, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	if !conversation.IsActive() {
		return nil, domain.NewDomainError(domain.ErrConversationArchived, "cannot add messages to archived conversation")
	}

	if !isValidRole(role) {
		return nil, domain.NewDomainError(domain.ErrInvalidRole, "invalid message role")
	}

	sequenceNumber, err := s.messageRepo.GetNextSequenceNumber(ctx, conversationID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get next sequence number")
	}

	id := s.idGenerator.GenerateMessageID()
	message := models.NewMessage(id, conversationID, sequenceNumber, role, contents)

	// Set previous_id to the current conversation tip for message branching
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		message.SetPreviousMessage(*conversation.TipMessageID)
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, domain.NewDomainError(err, "failed to create message")
	}

	// Update conversation tip to point to the new message
	if err := s.convRepo.UpdateTip(ctx, conversationID, message.ID); err != nil {
		// Log but don't fail - this is a non-critical operation
		// The message is already created successfully
		return message, nil
	}

	return message, nil
}

func (s *MessageService) CreateUserMessage(ctx context.Context, conversationID, contents string) (*models.Message, error) {
	return s.Create(ctx, conversationID, models.MessageRoleUser, contents)
}

func (s *MessageService) CreateAssistantMessage(ctx context.Context, conversationID, contents string) (*models.Message, error) {
	return s.Create(ctx, conversationID, models.MessageRoleAssistant, contents)
}

func (s *MessageService) CreateSystemMessage(ctx context.Context, conversationID, contents string) (*models.Message, error) {
	return s.Create(ctx, conversationID, models.MessageRoleSystem, contents)
}

func (s *MessageService) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if err := ValidateID(id, "message"); err != nil {
		return nil, err
	}

	message, err := s.messageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "message not found")
	}

	if err := ValidateNotDeleted(message.DeletedAt, "message"); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	if err := ValidateID(conversationID, "conversation"); err != nil {
		return nil, err
	}

	conversation, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	var messages []*models.Message

	// If conversation has a tip, get the chain from the tip
	// Otherwise fall back to getting all messages (for backwards compatibility)
	if conversation.TipMessageID != nil && *conversation.TipMessageID != "" {
		messages, err = s.messageRepo.GetChainFromTip(ctx, *conversation.TipMessageID)
	} else {
		messages, err = s.messageRepo.GetByConversation(ctx, conversationID)
	}

	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get messages")
	}

	return messages, nil
}

func (s *MessageService) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	if err := ValidateID(conversationID, "conversation"); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	messages, err := s.messageRepo.GetLatestByConversation(ctx, conversationID, limit)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get latest messages")
	}

	return messages, nil
}

func (s *MessageService) Update(ctx context.Context, message *models.Message) error {
	if message == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "message cannot be nil")
	}

	if err := ValidateID(message.ID, "message"); err != nil {
		return err
	}

	existing, err := s.messageRepo.GetByID(ctx, message.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrMessageNotFound, "message not found")
	}

	if err := ValidateNotDeleted(existing.DeletedAt, "message"); err != nil {
		return err
	}

	if err := s.messageRepo.Update(ctx, message); err != nil {
		return domain.NewDomainError(err, "failed to update message")
	}

	return nil
}

func (s *MessageService) AppendContent(ctx context.Context, id, content string) (*models.Message, error) {
	message, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	message.AppendContent(content)
	if err := s.Update(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) LinkToPrevious(ctx context.Context, messageID, previousID string) (*models.Message, error) {
	message, err := s.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if _, err := s.GetByID(ctx, previousID); err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "previous message not found")
	}

	message.SetPreviousMessage(previousID)
	if err := s.Update(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) Delete(ctx context.Context, id string) error {
	message, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.messageRepo.Delete(ctx, message.ID); err != nil {
		return domain.NewDomainError(err, "failed to delete message")
	}

	return nil
}

func (s *MessageService) CreateSentence(ctx context.Context, messageID, text string) (*models.Sentence, error) {
	if err := ValidateID(messageID, "message"); err != nil {
		return nil, err
	}

	if err := ValidateRequired(text, "sentence text"); err != nil {
		return nil, err
	}

	if _, err := s.GetByID(ctx, messageID); err != nil {
		return nil, err
	}

	sequenceNumber, err := s.sentenceRepo.GetNextSequenceNumber(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get next sequence number")
	}

	id := s.idGenerator.GenerateSentenceID()
	sentence := models.NewSentence(id, messageID, sequenceNumber, text)

	if err := s.sentenceRepo.Create(ctx, sentence); err != nil {
		return nil, domain.NewDomainError(err, "failed to create sentence")
	}

	return sentence, nil
}

func (s *MessageService) GetSentencesByMessage(ctx context.Context, messageID string) ([]*models.Sentence, error) {
	if err := ValidateID(messageID, "message"); err != nil {
		return nil, err
	}

	sentences, err := s.sentenceRepo.GetByMessage(ctx, messageID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get sentences")
	}

	return sentences, nil
}

func (s *MessageService) UpdateSentence(ctx context.Context, sentence *models.Sentence) error {
	if sentence == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "sentence cannot be nil")
	}

	if err := ValidateID(sentence.ID, "sentence"); err != nil {
		return err
	}

	existing, err := s.sentenceRepo.GetByID(ctx, sentence.ID)
	if err != nil {
		return domain.NewDomainError(domain.ErrMessageNotFound, "sentence not found")
	}

	if err := ValidateNotDeleted(existing.DeletedAt, "sentence"); err != nil {
		return err
	}

	if err := s.sentenceRepo.Update(ctx, sentence); err != nil {
		return domain.NewDomainError(err, "failed to update sentence")
	}

	return nil
}

func (s *MessageService) AttachAudioToSentence(ctx context.Context, sentenceID string, audioType models.AudioType, format string, data []byte, durationMs int) (*models.Sentence, error) {
	sentence, err := s.sentenceRepo.GetByID(ctx, sentenceID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrMessageNotFound, "sentence not found")
	}

	if err := ValidateNotDeleted(sentence.DeletedAt, "sentence"); err != nil {
		return nil, err
	}

	sentence.SetAudio(audioType, format, data, durationMs)
	if err := s.UpdateSentence(ctx, sentence); err != nil {
		return nil, err
	}

	return sentence, nil
}

func (s *MessageService) DeleteSentence(ctx context.Context, id string) error {
	if err := ValidateID(id, "sentence"); err != nil {
		return err
	}

	if err := s.sentenceRepo.Delete(ctx, id); err != nil {
		return domain.NewDomainError(err, "failed to delete sentence")
	}

	return nil
}

func isValidRole(role models.MessageRole) bool {
	return role == models.MessageRoleUser ||
		role == models.MessageRoleAssistant ||
		role == models.MessageRoleSystem
}
