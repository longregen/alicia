package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

type mockLiveKitService struct {
	createRoomFunc func(ctx context.Context, name string) (*ports.LiveKitRoom, error)
	getRoomFunc    func(ctx context.Context, name string) (*ports.LiveKitRoom, error)
	deleteRoomFunc func(ctx context.Context, name string) error
	rooms          map[string]*ports.LiveKitRoom
}

func newMockLiveKitService() *mockLiveKitService {
	return &mockLiveKitService{
		rooms: make(map[string]*ports.LiveKitRoom),
	}
}

func (m *mockLiveKitService) CreateRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if m.createRoomFunc != nil {
		return m.createRoomFunc(ctx, name)
	}
	if _, exists := m.rooms[name]; exists {
		return nil, errors.New("room already exists")
	}
	room := &ports.LiveKitRoom{Name: name, SID: "room_" + name}
	m.rooms[name] = room
	return room, nil
}

func (m *mockLiveKitService) GetRoom(ctx context.Context, name string) (*ports.LiveKitRoom, error) {
	if m.getRoomFunc != nil {
		return m.getRoomFunc(ctx, name)
	}
	if room, ok := m.rooms[name]; ok {
		return room, nil
	}
	return nil, errors.New("room not found")
}

func (m *mockLiveKitService) DeleteRoom(ctx context.Context, name string) error {
	if m.deleteRoomFunc != nil {
		return m.deleteRoomFunc(ctx, name)
	}
	delete(m.rooms, name)
	return nil
}

func (m *mockLiveKitService) GenerateToken(ctx context.Context, roomName, participantID, participantName string) (*ports.LiveKitToken, error) {
	return &ports.LiveKitToken{Token: "test_token"}, nil
}

func (m *mockLiveKitService) ListParticipants(ctx context.Context, roomName string) ([]*ports.LiveKitParticipant, error) {
	return []*ports.LiveKitParticipant{}, nil
}

func (m *mockLiveKitService) SendData(ctx context.Context, roomName string, data []byte, participantIDs []string) error {
	return nil
}

func TestManageConversation_StartConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &StartConversationInput{
		Title: "Test Conversation",
	}

	output, err := uc.StartConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Conversation == nil {
		t.Fatal("expected conversation to be created")
	}

	if output.Conversation.Title != "Test Conversation" {
		t.Errorf("expected title 'Test Conversation', got %s", output.Conversation.Title)
	}

	if output.Conversation.Status != models.ConversationStatusActive {
		t.Errorf("expected status active, got %s", output.Conversation.Status)
	}

	stored, _ := convRepo.GetByID(context.Background(), output.Conversation.ID)
	if stored == nil {
		t.Error("conversation not stored in repository")
	}
}

func TestManageConversation_StartConversationWithLiveKit(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &StartConversationInput{
		Title:           "LiveKit Conversation",
		LiveKitRoomName: "test_room",
	}

	output, err := uc.StartConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Conversation.LiveKitRoomName != "test_room" {
		t.Errorf("expected LiveKit room 'test_room', got %s", output.Conversation.LiveKitRoomName)
	}

	if _, exists := liveKitService.rooms["test_room"]; !exists {
		t.Error("expected LiveKit room to be created")
	}
}

func TestManageConversation_StartConversationDefaultTitle(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &StartConversationInput{
		Title: "",
	}

	output, err := uc.StartConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Conversation.Title == "" {
		t.Error("expected default title to be generated")
	}
}

func TestManageConversation_ResumeConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	msg := models.NewMessage("msg_1", "conv_1", 0, models.MessageRoleUser, "Hello")
	msgRepo.Create(context.Background(), msg)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &ResumeConversationInput{
		ConversationID: "conv_1",
	}

	output, err := uc.ResumeConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Conversation == nil {
		t.Fatal("expected conversation to be returned")
	}

	if output.Conversation.ID != "conv_1" {
		t.Errorf("expected conversation ID conv_1, got %s", output.Conversation.ID)
	}

	if output.Messages == nil {
		t.Error("expected messages to be returned")
	}
}

func TestManageConversation_ResumeNonexistentConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &ResumeConversationInput{
		ConversationID: "nonexistent",
	}

	_, err := uc.ResumeConversation(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for nonexistent conversation, got nil")
	}
}

func TestManageConversation_ResumeArchivedConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	conv.Archive()
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &ResumeConversationInput{
		ConversationID: "conv_1",
	}

	_, err := uc.ResumeConversation(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for archived conversation, got nil")
	}
}

func TestManageConversation_ArchiveConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &ArchiveConversationInput{
		ConversationID: "conv_1",
	}

	err := uc.ArchiveConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	archived, _ := convRepo.GetByID(context.Background(), "conv_1")
	if archived.Status != models.ConversationStatusArchived {
		t.Errorf("expected status archived, got %s", archived.Status)
	}
}

func TestManageConversation_ArchiveConversationWithLiveKit(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	conv.SetLiveKitRoom("test_room")
	convRepo.Create(context.Background(), conv)

	liveKitService.rooms["test_room"] = &ports.LiveKitRoom{Name: "test_room"}

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &ArchiveConversationInput{
		ConversationID: "conv_1",
	}

	err := uc.ArchiveConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := liveKitService.rooms["test_room"]; exists {
		t.Error("expected LiveKit room to be deleted")
	}
}

func TestManageConversation_UnarchiveConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	conv.Archive()
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	err := uc.UnarchiveConversation(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unarchived, _ := convRepo.GetByID(context.Background(), "conv_1")
	if unarchived.Status != models.ConversationStatusActive {
		t.Errorf("expected status active, got %s", unarchived.Status)
	}
}

func TestManageConversation_DeleteConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &DeleteConversationInput{
		ConversationID: "conv_1",
	}

	err := uc.DeleteConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = convRepo.GetByID(context.Background(), "conv_1")
	if err == nil {
		t.Error("expected conversation to be deleted")
	}
}

func TestManageConversation_DeleteConversationWithLiveKit(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	conv.SetLiveKitRoom("test_room")
	convRepo.Create(context.Background(), conv)

	liveKitService.rooms["test_room"] = &ports.LiveKitRoom{Name: "test_room"}

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	input := &DeleteConversationInput{
		ConversationID: "conv_1",
	}

	err := uc.DeleteConversation(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, exists := liveKitService.rooms["test_room"]; exists {
		t.Error("expected LiveKit room to be deleted")
	}
}

func TestManageConversation_ListActiveConversations(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv1 := models.NewConversation("conv_1", "test-user", "Active Conversation 1")
	convRepo.Create(context.Background(), conv1)

	conv2 := models.NewConversation("conv_2", "test-user", "Active Conversation 2")
	convRepo.Create(context.Background(), conv2)

	conv3 := models.NewConversation("conv_3", "test-user", "Archived Conversation")
	conv3.Archive()
	convRepo.Create(context.Background(), conv3)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	conversations, err := uc.ListActiveConversations(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conversations == nil {
		t.Error("expected non-nil conversations list")
	}
}

func TestManageConversation_GetConversation(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	retrieved, err := uc.GetConversation(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != "conv_1" {
		t.Errorf("expected ID conv_1, got %s", retrieved.ID)
	}
}

func TestManageConversation_UpdateConversationPreferences(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	prefs := &models.ConversationPreferences{
		TTSVoice:        "test_voice",
		EnableReasoning: true,
	}

	err := uc.UpdateConversationPreferences(context.Background(), "conv_1", prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := convRepo.GetByID(context.Background(), "conv_1")
	if updated.Preferences == nil {
		t.Fatal("expected preferences to be set")
	}

	if updated.Preferences.TTSVoice != "test_voice" {
		t.Errorf("expected voice 'test_voice', got %s", updated.Preferences.TTSVoice)
	}
}

func TestManageConversation_ResumeConversationByLiveKit(t *testing.T) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	liveKitService := newMockLiveKitService()
	idGen := newMockIDGenerator()

	conv := models.NewConversation("conv_1", "test-user", "Test Conversation")
	conv.SetLiveKitRoom("test_room")
	convRepo.Create(context.Background(), conv)

	uc := NewManageConversation(convRepo, msgRepo, liveKitService, idGen)

	output, err := uc.ResumeConversationByLiveKit(context.Background(), "test_room")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Conversation.ID != "conv_1" {
		t.Errorf("expected conversation ID conv_1, got %s", output.Conversation.ID)
	}
}
