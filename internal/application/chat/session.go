package chat

import (
	"context"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Session represents a chat session that manages a conversation and its messages.
// It provides a stateful interface for interacting with a single conversation.
type Session struct {
	// Current conversation being tracked by this session
	conversation *models.Conversation

	// Services for managing conversations and messages
	conversationRepo ports.ConversationRepository
	messageRepo      ports.MessageRepository
	idGenerator      ports.IDGenerator

	// Message history cache (optional optimization)
	messageHistory []*models.Message
}

// NewSession creates a new session with the required dependencies
func NewSession(
	conversationRepo ports.ConversationRepository,
	messageRepo ports.MessageRepository,
	idGenerator ports.IDGenerator,
) *Session {
	return &Session{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		idGenerator:      idGenerator,
		messageHistory:   make([]*models.Message, 0),
	}
}

// StartNew creates a new conversation and initializes the session with it
func (s *Session) StartNew(ctx context.Context, title string) (*Session, error) {
	// Validate title
	if title == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidInput, "conversation title is required")
	}

	// Generate a new conversation ID
	conversationID := s.idGenerator.GenerateConversationID()

	// Create the conversation model
	conversation := models.NewConversation(conversationID, "default-user", title)

	// Persist the conversation
	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, domain.NewDomainError(err, "failed to create conversation")
	}

	// Update session state
	s.conversation = conversation
	s.messageHistory = make([]*models.Message, 0)

	return s, nil
}

// Resume loads an existing conversation and initializes the session with it
func (s *Session) Resume(ctx context.Context, conversationID string) (*Session, error) {
	// Validate conversation ID
	if conversationID == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidID, "conversation ID cannot be empty")
	}

	// Retrieve the conversation
	conversation, err := s.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	// Validate conversation state
	if conversation.DeletedAt != nil {
		return nil, domain.NewDomainError(domain.ErrConversationDeleted, "cannot resume a deleted conversation")
	}

	if !conversation.IsActive() {
		return nil, domain.NewDomainError(domain.ErrConversationArchived, "cannot resume an archived conversation")
	}

	// Load message history (last 50 messages for context)
	messages, err := s.messageRepo.GetLatestByConversation(ctx, conversationID, 50)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to load conversation history")
	}

	// Update session state
	s.conversation = conversation
	s.messageHistory = messages

	return s, nil
}

// Send creates a new message in the current conversation session
func (s *Session) Send(ctx context.Context, content string) (*models.Message, error) {
	// Validate session state
	if s.conversation == nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "no active conversation in session - call StartNew or Resume first")
	}

	// Validate content
	if content == "" {
		return nil, domain.NewDomainError(domain.ErrEmptyContent, "message content cannot be empty")
	}

	// Re-check conversation state (it may have been modified externally)
	conversation, err := s.conversationRepo.GetByID(ctx, s.conversation.ID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrConversationNotFound, "conversation not found")
	}

	if !conversation.IsActive() {
		return nil, domain.NewDomainError(domain.ErrConversationArchived, "cannot send messages to an inactive conversation")
	}

	// Get the next sequence number for this message
	sequenceNumber, err := s.messageRepo.GetNextSequenceNumber(ctx, s.conversation.ID)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get next sequence number")
	}

	// Generate message ID
	messageID := s.idGenerator.GenerateMessageID()

	// Create the message (assuming user role for sent messages)
	message := models.NewUserMessage(messageID, s.conversation.ID, sequenceNumber, content)

	// Link to previous message if exists
	if len(s.messageHistory) > 0 {
		lastMessage := s.messageHistory[len(s.messageHistory)-1]
		message.SetPreviousMessage(lastMessage.ID)
	}

	// Persist the message
	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, domain.NewDomainError(err, "failed to create message")
	}

	// Update local cache
	s.messageHistory = append(s.messageHistory, message)
	s.conversation = conversation // Update to latest state

	return message, nil
}

// GetConversation returns the current conversation in the session
func (s *Session) GetConversation() (*models.Conversation, error) {
	if s.conversation == nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "no active conversation in session")
	}
	return s.conversation, nil
}

// GetMessageHistory returns the cached message history
func (s *Session) GetMessageHistory() ([]*models.Message, error) {
	if s.conversation == nil {
		return nil, domain.NewDomainError(domain.ErrInvalidState, "no active conversation in session")
	}
	return s.messageHistory, nil
}

// RefreshHistory reloads the message history from the repository
func (s *Session) RefreshHistory(ctx context.Context) error {
	if s.conversation == nil {
		return domain.NewDomainError(domain.ErrInvalidState, "no active conversation in session")
	}

	messages, err := s.messageRepo.GetLatestByConversation(ctx, s.conversation.ID, 50)
	if err != nil {
		return domain.NewDomainError(err, "failed to refresh message history")
	}

	s.messageHistory = messages
	return nil
}

// ConversationID returns the ID of the current conversation
func (s *Session) ConversationID() (string, error) {
	if s.conversation == nil {
		return "", domain.NewDomainError(domain.ErrInvalidState, "no active conversation in session")
	}
	return s.conversation.ID, nil
}

// IsActive returns whether the session has an active conversation
func (s *Session) IsActive() bool {
	return s.conversation != nil && s.conversation.IsActive()
}

// Close clears the session state
func (s *Session) Close() {
	s.conversation = nil
	s.messageHistory = nil
}
