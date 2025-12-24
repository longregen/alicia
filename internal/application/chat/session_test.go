package chat_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/application/chat"
	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
)

// Mock implementations for testing
type mockConversationRepo struct {
	conversations map[string]*models.Conversation
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		conversations: make(map[string]*models.Conversation),
	}
}

func (m *mockConversationRepo) Create(ctx context.Context, conv *models.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if conv, ok := m.conversations[id]; ok {
		return conv, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConversationRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, domain.ErrConversationNotFound
}

func (m *mockConversationRepo) Update(ctx context.Context, conv *models.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockConversationRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	if conv, ok := m.conversations[id]; ok {
		conv.LastClientStanzaID = clientStanza
		conv.LastServerStanzaID = serverStanza
	}
	return nil
}

func (m *mockConversationRepo) Delete(ctx context.Context, id string) error {
	delete(m.conversations, id)
	return nil
}

func (m *mockConversationRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	delete(m.conversations, id)
	return nil
}

func (m *mockConversationRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if conv, ok := m.conversations[id]; ok {
		return conv, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConversationRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

type mockMessageRepo struct {
	messages       map[string]*models.Message
	sequenceNumber int
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		messages:       make(map[string]*models.Message),
		sequenceNumber: 1,
	}
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	if msg, ok := m.messages[id]; ok {
		return msg, nil
	}
	return nil, domain.ErrMessageNotFound
}

func (m *mockMessageRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	var msgs []*models.Message
	for _, msg := range m.messages {
		if msg.ConversationID == conversationID {
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

func (m *mockMessageRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	return m.GetByConversation(ctx, conversationID)
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id string) error {
	delete(m.messages, id)
	return nil
}

func (m *mockMessageRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	seq := m.sequenceNumber
	m.sequenceNumber++
	return seq, nil
}

func (m *mockMessageRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	var msgs []*models.Message
	for _, msg := range m.messages {
		if msg.ConversationID == conversationID && msg.SequenceNumber > afterSequence {
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

func (m *mockMessageRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, domain.ErrMessageNotFound
}

func (m *mockMessageRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

type mockIDGenerator struct {
	counter int
}

func newMockIDGenerator() *mockIDGenerator {
	return &mockIDGenerator{counter: 1}
}

func (m *mockIDGenerator) GenerateConversationID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("ac_%d", id)
}

func (m *mockIDGenerator) GenerateMessageID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("am_%d", id)
}

func (m *mockIDGenerator) GenerateSentenceID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("ams_%d", id)
}

func (m *mockIDGenerator) GenerateToolID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("at_%d", id)
}

func (m *mockIDGenerator) GenerateMemoryID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("amem_%d", id)
}

func (m *mockIDGenerator) GenerateAudioID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("aa_%d", id)
}

func (m *mockIDGenerator) GenerateMemoryUsageID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("amu_%d", id)
}

func (m *mockIDGenerator) GenerateToolUseID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("atu_%d", id)
}

func (m *mockIDGenerator) GenerateReasoningStepID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("ar_%d", id)
}

func (m *mockIDGenerator) GenerateCommentaryID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("aucc_%d", id)
}

func (m *mockIDGenerator) GenerateMetaID() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("amt_%d", id)
}

func (m *mockIDGenerator) GenerateLiveKitRoomName() string {
	id := m.counter
	m.counter++
	return fmt.Sprintf("room_%d", id)
}

func (m *mockIDGenerator) GenerateMCPServerID() string {
	return "amcp_test"
}

// TestSessionStartNew tests creating a new conversation session
func TestSessionStartNew(t *testing.T) {
	ctx := context.Background()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	session := chat.NewSession(convRepo, msgRepo, idGen)

	// Start a new conversation
	_, err := session.StartNew(ctx, "Test Conversation")
	if err != nil {
		t.Fatalf("Failed to start new session: %v", err)
	}

	// Verify conversation was created
	conv, err := session.GetConversation()
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if conv.Title != "Test Conversation" {
		t.Errorf("Expected title 'Test Conversation', got %s", conv.Title)
	}

	if !session.IsActive() {
		t.Error("Expected session to be active")
	}
}

// TestSessionResume tests resuming an existing conversation
func TestSessionResume(t *testing.T) {
	ctx := context.Background()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	// Create a conversation first
	session1 := chat.NewSession(convRepo, msgRepo, idGen)
	_, err := session1.StartNew(ctx, "Existing Conversation")
	if err != nil {
		t.Fatalf("Failed to start new session: %v", err)
	}

	convID, err := session1.ConversationID()
	if err != nil {
		t.Fatalf("Failed to get conversation ID: %v", err)
	}

	// Resume the conversation in a new session
	session2 := chat.NewSession(convRepo, msgRepo, idGen)
	_, err = session2.Resume(ctx, convID)
	if err != nil {
		t.Fatalf("Failed to resume session: %v", err)
	}

	// Verify conversation was loaded
	conv, err := session2.GetConversation()
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if conv.Title != "Existing Conversation" {
		t.Errorf("Expected title 'Existing Conversation', got %s", conv.Title)
	}
}

// TestSessionSend tests sending a message in a session
func TestSessionSend(t *testing.T) {
	ctx := context.Background()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	session := chat.NewSession(convRepo, msgRepo, idGen)

	// Start a new conversation
	_, err := session.StartNew(ctx, "Test Conversation")
	if err != nil {
		t.Fatalf("Failed to start new session: %v", err)
	}

	// Send a message
	msg, err := session.Send(ctx, "Hello, Alicia!")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if msg.Contents != "Hello, Alicia!" {
		t.Errorf("Expected content 'Hello, Alicia!', got %s", msg.Contents)
	}

	if msg.Role != models.MessageRoleUser {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}

	// Verify message history
	history, err := session.GetMessageHistory()
	if err != nil {
		t.Fatalf("Failed to get message history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 message in history, got %d", len(history))
	}
}

// TestSessionSendWithoutConversation tests error when sending without a conversation
func TestSessionSendWithoutConversation(t *testing.T) {
	ctx := context.Background()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	session := chat.NewSession(convRepo, msgRepo, idGen)

	// Try to send without starting a conversation
	_, err := session.Send(ctx, "Hello")
	if err == nil {
		t.Error("Expected error when sending without a conversation, got nil")
	}
}

// TestSessionMultipleMessages tests sending multiple messages
func TestSessionMultipleMessages(t *testing.T) {
	ctx := context.Background()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	idGen := newMockIDGenerator()

	session := chat.NewSession(convRepo, msgRepo, idGen)

	// Start a new conversation
	_, err := session.StartNew(ctx, "Multi-message Test")
	if err != nil {
		t.Fatalf("Failed to start new session: %v", err)
	}

	// Send multiple messages
	messages := []string{"First message", "Second message", "Third message"}
	for _, content := range messages {
		_, err := session.Send(ctx, content)
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}
	}

	// Verify message history
	history, err := session.GetMessageHistory()
	if err != nil {
		t.Fatalf("Failed to get message history: %v", err)
	}

	if len(history) != len(messages) {
		t.Errorf("Expected %d messages in history, got %d", len(messages), len(history))
	}

	// Verify message linking
	for i := 1; i < len(history); i++ {
		if history[i].PreviousID != history[i-1].ID {
			t.Errorf("Message %d not properly linked to previous message", i)
		}
	}
}
