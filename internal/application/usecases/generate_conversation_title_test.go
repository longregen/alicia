package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// Mock implementations for testing
type mockConversationRepoForTitle struct {
	conversation *models.Conversation
	updateCalled bool
	updatedTitle string
}

func (m *mockConversationRepoForTitle) Create(ctx context.Context, c *models.Conversation) error {
	return nil
}

func (m *mockConversationRepoForTitle) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	return m.conversation, nil
}

func (m *mockConversationRepoForTitle) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	return m.conversation, nil
}

func (m *mockConversationRepoForTitle) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return m.conversation, nil
}

func (m *mockConversationRepoForTitle) Update(ctx context.Context, c *models.Conversation) error {
	m.updateCalled = true
	m.updatedTitle = c.Title
	return nil
}

func (m *mockConversationRepoForTitle) UpdateStanzaIDs(ctx context.Context, conversationID string, clientStanzaID, serverStanzaID int32) error {
	return nil
}

func (m *mockConversationRepoForTitle) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	return nil
}

func (m *mockConversationRepoForTitle) UpdatePromptVersion(ctx context.Context, conversationID, versionID string) error {
	return nil
}

func (m *mockConversationRepoForTitle) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockConversationRepoForTitle) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	return nil
}

func (m *mockConversationRepoForTitle) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepoForTitle) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepoForTitle) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

func (m *mockConversationRepoForTitle) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return nil, nil
}

type mockMessageRepoForTitle struct {
	messages []*models.Message
}

func (m *mockMessageRepoForTitle) Create(ctx context.Context, msg *models.Message) error {
	return nil
}

func (m *mockMessageRepoForTitle) GetByID(ctx context.Context, id string) (*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) Update(ctx context.Context, msg *models.Message) error {
	return nil
}

func (m *mockMessageRepoForTitle) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMessageRepoForTitle) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return m.messages, nil
}

func (m *mockMessageRepoForTitle) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	if limit > len(m.messages) {
		return m.messages, nil
	}
	return m.messages[:limit], nil
}

func (m *mockMessageRepoForTitle) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return len(m.messages) + 1, nil
}

func (m *mockMessageRepoForTitle) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

func (m *mockMessageRepoForTitle) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return nil, nil
}

type mockLLMServiceForTitle struct {
	response string
}

func (m *mockLLMServiceForTitle) Chat(ctx context.Context, messages []ports.LLMMessage) (*ports.LLMResponse, error) {
	return &ports.LLMResponse{Content: m.response}, nil
}

func (m *mockLLMServiceForTitle) ChatWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (*ports.LLMResponse, error) {
	return &ports.LLMResponse{Content: m.response}, nil
}

func (m *mockLLMServiceForTitle) ChatStream(ctx context.Context, messages []ports.LLMMessage) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

func (m *mockLLMServiceForTitle) ChatStreamWithTools(ctx context.Context, messages []ports.LLMMessage, tools []*models.Tool) (<-chan ports.LLMStreamChunk, error) {
	return nil, nil
}

type mockBroadcasterForTitle struct {
	broadcastCalled bool
}

func (m *mockBroadcasterForTitle) BroadcastConversationUpdate(conversation *models.Conversation) {
	m.broadcastCalled = true
}

func TestNeedsTitleGeneration(t *testing.T) {
	uc := &GenerateConversationTitle{}

	tests := []struct {
		name     string
		title    string
		expected bool
	}{
		{"empty title", "", true},
		{"auto-generated title", "Conversation 2024-01-15 14:30", true},
		{"new chat title", "New Chat", true},
		{"custom title", "My Custom Title", false},
		{"another auto-generated", "Conversation 2023-12-01 09:00", true},
		{"partial match", "Conversation about Python", false},
		{"new chat lowercase", "new chat", false},
		{"new chat with suffix", "New Chat 1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.NeedsTitleGeneration(tt.title)
			if result != tt.expected {
				t.Errorf("NeedsTitleGeneration(%q) = %v, want %v", tt.title, result, tt.expected)
			}
		})
	}
}

func TestGenerateConversationTitle_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		conversation   *models.Conversation
		messages       []*models.Message
		llmResponse    string
		expectUpdate   bool
		expectedTitle  string
	}{
		{
			name: "generates title for auto-generated title",
			conversation: &models.Conversation{
				ID:    "conv-1",
				Title: "Conversation 2024-01-15 14:30",
			},
			messages: []*models.Message{
				{ID: "msg-1", Role: "user", Contents: "How do I sort a list in Python?"},
				{ID: "msg-2", Role: "assistant", Contents: "You can use the sorted() function or the .sort() method."},
			},
			llmResponse:   "Python List Sorting Help",
			expectUpdate:  true,
			expectedTitle: "Python List Sorting Help",
		},
		{
			name: "generates title for New Chat",
			conversation: &models.Conversation{
				ID:    "conv-new-chat",
				Title: "New Chat",
			},
			messages: []*models.Message{
				{ID: "msg-1", Role: "user", Contents: "What is the weather like today?"},
				{ID: "msg-2", Role: "assistant", Contents: "I don't have access to real-time weather data."},
			},
			llmResponse:   "Weather Inquiry",
			expectUpdate:  true,
			expectedTitle: "Weather Inquiry",
		},
		{
			name: "skips if custom title already set",
			conversation: &models.Conversation{
				ID:    "conv-2",
				Title: "My Custom Title",
			},
			messages:     []*models.Message{},
			llmResponse:  "Some Title",
			expectUpdate: false,
		},
		{
			name: "skips if not enough messages",
			conversation: &models.Conversation{
				ID:    "conv-3",
				Title: "Conversation 2024-01-15 14:30",
			},
			messages: []*models.Message{
				{ID: "msg-1", Role: "user", Contents: "Hello"},
			},
			llmResponse:  "Greeting",
			expectUpdate: false,
		},
		{
			name: "truncates long titles",
			conversation: &models.Conversation{
				ID:    "conv-4",
				Title: "Conversation 2024-01-15 14:30",
			},
			messages: []*models.Message{
				{ID: "msg-1", Role: "user", Contents: "Tell me about quantum physics"},
				{ID: "msg-2", Role: "assistant", Contents: "Quantum physics is fascinating..."},
			},
			llmResponse:  "A Very Long Title About Quantum Physics That Exceeds The Maximum Character Limit",
			expectUpdate: true,
			// Title should be truncated to ~50 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convRepo := &mockConversationRepoForTitle{conversation: tt.conversation}
			msgRepo := &mockMessageRepoForTitle{messages: tt.messages}
			llmService := &mockLLMServiceForTitle{response: tt.llmResponse}
			broadcaster := &mockBroadcasterForTitle{}

			uc := NewGenerateConversationTitle(convRepo, msgRepo, llmService, broadcaster)

			err := uc.Execute(ctx, tt.conversation.ID)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if convRepo.updateCalled != tt.expectUpdate {
				t.Errorf("Update called = %v, want %v", convRepo.updateCalled, tt.expectUpdate)
			}

			if tt.expectUpdate && tt.expectedTitle != "" {
				if convRepo.updatedTitle != tt.expectedTitle {
					t.Errorf("Updated title = %q, want %q", convRepo.updatedTitle, tt.expectedTitle)
				}
			}

			// Verify title length constraint
			if convRepo.updateCalled && len(convRepo.updatedTitle) > 50 {
				t.Errorf("Updated title too long: %d chars (max 50)", len(convRepo.updatedTitle))
			}
		})
	}
}
