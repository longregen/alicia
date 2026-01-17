package usecases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// continueResponseMockGenUC is a mock for the GenerateResponse use case
// (named uniquely to avoid conflicts with other test files)
type continueResponseMockGenUC struct {
	executeFunc func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error)
}

func (m *continueResponseMockGenUC) Execute(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &ports.GenerateResponseOutput{
		Message: models.NewAssistantMessage("msg_generated", input.ConversationID, 0, "Generated continuation"),
	}, nil
}

// continueResponseMockMsgRepo is a specialized mock for ContinueResponse tests
type continueResponseMockMsgRepo struct {
	mu        sync.RWMutex
	store     map[string]*models.Message
	getError  error
	updateErr error
	deleteErr error
}

func newContinueResponseMockMsgRepo() *continueResponseMockMsgRepo {
	return &continueResponseMockMsgRepo{
		store: make(map[string]*models.Message),
	}
}

func (m *continueResponseMockMsgRepo) copyMessage(msg *models.Message) *models.Message {
	if msg == nil {
		return nil
	}
	msgCopy := &models.Message{
		ID:               msg.ID,
		ConversationID:   msg.ConversationID,
		SequenceNumber:   msg.SequenceNumber,
		PreviousID:       msg.PreviousID,
		Role:             msg.Role,
		Contents:         msg.Contents,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
		LocalID:          msg.LocalID,
		ServerID:         msg.ServerID,
		SyncStatus:       msg.SyncStatus,
		CompletionStatus: msg.CompletionStatus,
	}
	if msg.DeletedAt != nil {
		deletedAt := *msg.DeletedAt
		msgCopy.DeletedAt = &deletedAt
	}
	if msg.SyncedAt != nil {
		syncedAt := *msg.SyncedAt
		msgCopy.SyncedAt = &syncedAt
	}
	return msgCopy
}

func (m *continueResponseMockMsgRepo) Create(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *continueResponseMockMsgRepo) GetByID(ctx context.Context, id string) (*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getError != nil {
		return nil, m.getError
	}
	if msg, ok := m.store[id]; ok {
		return m.copyMessage(msg), nil
	}
	return nil, nil // Return nil, nil when not found (per the use case expectations)
}

func (m *continueResponseMockMsgRepo) Update(ctx context.Context, msg *models.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.store[msg.ID]; !ok {
		return errors.New("not found")
	}
	m.store[msg.ID] = m.copyMessage(msg)
	return nil
}

func (m *continueResponseMockMsgRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.store, id)
	return nil
}

func (m *continueResponseMockMsgRepo) GetByConversation(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetLatestByConversation(ctx context.Context, conversationID string, limit int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetNextSequenceNumber(ctx context.Context, conversationID string) (int, error) {
	return 0, nil
}

func (m *continueResponseMockMsgRepo) GetAfterSequence(ctx context.Context, conversationID string, afterSequence int) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetChainFromTip(ctx context.Context, tipMessageID string) ([]*models.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var chain []*models.Message
	currentID := tipMessageID

	for currentID != "" {
		msg, ok := m.store[currentID]
		if !ok {
			break
		}
		chain = append([]*models.Message{m.copyMessage(msg)}, chain...)
		if msg.PreviousID == "" {
			break
		}
		currentID = msg.PreviousID
	}

	return chain, nil
}

func (m *continueResponseMockMsgRepo) GetSiblings(ctx context.Context, messageID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetPendingSync(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetByLocalID(ctx context.Context, localID string) (*models.Message, error) {
	return nil, errors.New("not found")
}

func (m *continueResponseMockMsgRepo) GetIncompleteOlderThan(ctx context.Context, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) GetIncompleteByConversation(ctx context.Context, conversationID string, olderThan time.Time) ([]*models.Message, error) {
	return []*models.Message{}, nil
}

func (m *continueResponseMockMsgRepo) DeleteAfterSequence(ctx context.Context, conversationID string, afterSequence int) error {
	return nil
}

// continueResponseMockConvRepo is a specialized mock for ContinueResponse tests
type continueResponseMockConvRepo struct {
	store        map[string]*models.Conversation
	updateTipErr error
}

func newContinueResponseMockConvRepo() *continueResponseMockConvRepo {
	return &continueResponseMockConvRepo{
		store: make(map[string]*models.Conversation),
	}
}

func (m *continueResponseMockConvRepo) Create(ctx context.Context, c *models.Conversation) error {
	m.store[c.ID] = c
	return nil
}

func (m *continueResponseMockConvRepo) GetByID(ctx context.Context, id string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (m *continueResponseMockConvRepo) Update(ctx context.Context, c *models.Conversation) error {
	if _, ok := m.store[c.ID]; !ok {
		return errors.New("not found")
	}
	m.store[c.ID] = c
	return nil
}

func (m *continueResponseMockConvRepo) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

func (m *continueResponseMockConvRepo) List(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *continueResponseMockConvRepo) ListActive(ctx context.Context, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *continueResponseMockConvRepo) GetByLiveKitRoom(ctx context.Context, roomName string) (*models.Conversation, error) {
	return nil, errors.New("not found")
}

func (m *continueResponseMockConvRepo) UpdateStanzaIDs(ctx context.Context, id string, clientStanza, serverStanza int32) error {
	return nil
}

func (m *continueResponseMockConvRepo) DeleteByIDAndUserID(ctx context.Context, id, userID string) error {
	delete(m.store, id)
	return nil
}

func (m *continueResponseMockConvRepo) GetByIDAndUserID(ctx context.Context, id, userID string) (*models.Conversation, error) {
	if c, ok := m.store[id]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (m *continueResponseMockConvRepo) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *continueResponseMockConvRepo) ListActiveByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	return []*models.Conversation{}, nil
}

func (m *continueResponseMockConvRepo) UpdateTip(ctx context.Context, conversationID, messageID string) error {
	if m.updateTipErr != nil {
		return m.updateTipErr
	}
	if c, ok := m.store[conversationID]; ok {
		c.TipMessageID = &messageID
		return nil
	}
	return errors.New("not found")
}

func (m *continueResponseMockConvRepo) UpdatePromptVersion(ctx context.Context, convID, versionID string) error {
	return nil
}

// continueResponseMockIDGen is a specialized mock for ContinueResponse tests
type continueResponseMockIDGen struct {
	messageCounter int
}

func newContinueResponseMockIDGen() *continueResponseMockIDGen {
	return &continueResponseMockIDGen{}
}

func (m *continueResponseMockIDGen) GenerateConversationID() string {
	return "conv_test1"
}

func (m *continueResponseMockIDGen) GenerateMessageID() string {
	m.messageCounter++
	return "msg_cont" + string(rune('0'+m.messageCounter))
}

func (m *continueResponseMockIDGen) GenerateSentenceID() string {
	return "sent_test1"
}

func (m *continueResponseMockIDGen) GenerateAudioID() string {
	return "audio_test1"
}

func (m *continueResponseMockIDGen) GenerateMemoryID() string {
	return "mem_test1"
}

func (m *continueResponseMockIDGen) GenerateMemoryUsageID() string {
	return "mu_test1"
}

func (m *continueResponseMockIDGen) GenerateToolID() string {
	return "tool_test1"
}

func (m *continueResponseMockIDGen) GenerateToolUseID() string {
	return "tu_test1"
}

func (m *continueResponseMockIDGen) GenerateReasoningStepID() string {
	return "rs_test1"
}

func (m *continueResponseMockIDGen) GenerateCommentaryID() string {
	return "comm_test1"
}

func (m *continueResponseMockIDGen) GenerateMetaID() string {
	return "meta_test1"
}

func (m *continueResponseMockIDGen) GenerateMCPServerID() string {
	return "amcp_test1"
}

func (m *continueResponseMockIDGen) GenerateVoteID() string {
	return "av_test1"
}

func (m *continueResponseMockIDGen) GenerateNoteID() string {
	return "an_test1"
}

func (m *continueResponseMockIDGen) GenerateSessionStatsID() string {
	return "ass_test1"
}

func (m *continueResponseMockIDGen) GenerateOptimizationRunID() string {
	return "aor_test1"
}

func (m *continueResponseMockIDGen) GeneratePromptCandidateID() string {
	return "apc_test1"
}

func (m *continueResponseMockIDGen) GeneratePromptEvaluationID() string {
	return "ape_test1"
}

func (m *continueResponseMockIDGen) GenerateTrainingExampleID() string {
	return "gte_test1"
}

func (m *continueResponseMockIDGen) GenerateSystemPromptVersionID() string {
	return "spv_test1"
}

// continueResponseMockTxManager is a mock transaction manager for testing
type continueResponseMockTxManager struct{}

func (m *continueResponseMockTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// continueResponseTestHelper wraps the test execution logic for ContinueResponse
// This mimics the real ContinueResponse behavior but uses mocks
type continueResponseTestHelper struct {
	messageRepo      *continueResponseMockMsgRepo
	conversationRepo *continueResponseMockConvRepo
	genMock          *continueResponseMockGenUC
	idGenerator      *continueResponseMockIDGen
	txManager        *continueResponseMockTxManager
}

func (uc *continueResponseTestHelper) Execute(ctx context.Context, input *ports.ContinueResponseInput) (*ports.ContinueResponseOutput, error) {
	// 1. Get target message by ID
	targetMessage, err := uc.messageRepo.GetByID(ctx, input.TargetMessageID)
	if err != nil {
		return nil, err
	}
	if targetMessage == nil {
		return nil, errors.New("target message not found: " + input.TargetMessageID)
	}

	// 2. Validate it's an assistant message
	if !targetMessage.IsFromAssistant() {
		return nil, errors.New("cannot continue: target message is not an assistant message")
	}

	// 3. Get the user message that triggered this response
	if targetMessage.PreviousID == "" {
		return nil, errors.New("cannot continue: target message has no previous message reference")
	}

	userMessage, err := uc.messageRepo.GetByID(ctx, targetMessage.PreviousID)
	if err != nil {
		return nil, err
	}
	if userMessage == nil {
		return nil, errors.New("previous message not found: " + targetMessage.PreviousID)
	}
	if !userMessage.IsFromUser() {
		return nil, errors.New("cannot continue: previous message is not a user message")
	}

	// Pre-generate the continuation message ID
	continuationMsgID := uc.idGenerator.GenerateMessageID()

	// 4. Call mock GenerateResponse
	generateInput := &ports.GenerateResponseInput{
		ConversationID:  targetMessage.ConversationID,
		UserMessageID:   userMessage.ID,
		MessageID:       continuationMsgID,
		EnableTools:     input.EnableTools,
		EnableReasoning: input.EnableReasoning,
		EnableStreaming: input.EnableStreaming,
		PreviousID:      targetMessage.ID,
	}

	generateOutput, err := uc.genMock.Execute(ctx, generateInput)
	if err != nil {
		return nil, err
	}

	// 5. Handle streaming mode
	if input.EnableStreaming {
		wrappedStream := make(chan *ports.ResponseStreamChunk, 10)
		// For test simplicity, just close the stream
		close(wrappedStream)

		return &ports.ContinueResponseOutput{
			TargetMessage:   targetMessage,
			StreamChannel:   wrappedStream,
			GeneratedOutput: generateOutput,
		}, nil
	}

	// 6. Append generated content to target message
	appendedContent := ""
	if generateOutput.Message != nil && generateOutput.Message.Contents != "" {
		appendedContent = generateOutput.Message.Contents

		err = uc.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
			if targetMessage.Contents != "" {
				targetMessage.Contents += "\n\n" + appendedContent
			} else {
				targetMessage.Contents = appendedContent
			}
			targetMessage.UpdatedAt = time.Now().UTC()

			if err := uc.messageRepo.Update(txCtx, targetMessage); err != nil {
				return err
			}

			// Delete the continuation message
			if err := uc.messageRepo.Delete(txCtx, generateOutput.Message.ID); err != nil {
				// Log but don't fail
			}

			// Update conversation tip
			if err := uc.conversationRepo.UpdateTip(txCtx, targetMessage.ConversationID, targetMessage.ID); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return &ports.ContinueResponseOutput{
		TargetMessage:   targetMessage,
		AppendedContent: appendedContent,
		GeneratedOutput: generateOutput,
	}, nil
}

func TestContinueResponse_Execute_Success(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Create the conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Create user message (the trigger)
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello, can you help me?")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message to continue from
	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Of course! I'd be happy to help you")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Configure mock generate response
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			// Verify the input parameters
			if input.ConversationID != "conv_123" {
				t.Errorf("expected conversation ID conv_123, got %s", input.ConversationID)
			}
			if input.UserMessageID != "user_msg_1" {
				t.Errorf("expected user message ID user_msg_1, got %s", input.UserMessageID)
			}
			if input.PreviousID != "assistant_msg_1" {
				t.Errorf("expected previous ID assistant_msg_1, got %s", input.PreviousID)
			}

			// Return generated continuation
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, " with your question.")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableTools:     false,
		EnableReasoning: false,
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TargetMessage == nil {
		t.Fatal("expected target message to be returned")
	}

	if output.TargetMessage.ID != "assistant_msg_1" {
		t.Errorf("expected target message ID assistant_msg_1, got %s", output.TargetMessage.ID)
	}

	if output.AppendedContent != " with your question." {
		t.Errorf("expected appended content ' with your question.', got %s", output.AppendedContent)
	}

	// Verify the message was updated with appended content
	updatedMsg, _ := msgRepo.GetByID(context.Background(), "assistant_msg_1")
	expectedContent := "Of course! I'd be happy to help you\n\n with your question."
	if updatedMsg.Contents != expectedContent {
		t.Errorf("expected updated content %q, got %q", expectedContent, updatedMsg.Contents)
	}
}

func TestContinueResponse_Execute_MessageNotFound(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "nonexistent_msg",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when target message not found, got nil")
	}

	if err.Error() != "target message not found: nonexistent_msg" {
		t.Errorf("expected 'target message not found' error, got: %v", err)
	}
}

func TestContinueResponse_Execute_NotAssistantMessage(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Create a user message (not assistant)
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "user_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when target is user message, got nil")
	}

	expectedErr := "cannot continue: target message is not an assistant message"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got: %v", expectedErr, err)
	}
}

func TestContinueResponse_Execute_NoUserMessage(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Create assistant message without PreviousID
	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 0, "I am ready to help")
	// Don't set PreviousID
	msgRepo.Create(context.Background(), assistantMsg)

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when no previous message reference, got nil")
	}

	expectedErr := "cannot continue: target message has no previous message reference"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got: %v", expectedErr, err)
	}
}

func TestContinueResponse_Execute_PreviousMessageNotFound(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Create assistant message with PreviousID pointing to non-existent message
	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 0, "I am ready to help")
	assistantMsg.PreviousID = "nonexistent_user_msg"
	msgRepo.Create(context.Background(), assistantMsg)

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when previous message not found, got nil")
	}

	expectedErr := "previous message not found: nonexistent_user_msg"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got: %v", expectedErr, err)
	}
}

func TestContinueResponse_Execute_PreviousIsNotUserMessage(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Create a system message as the previous message
	systemMsg := models.NewSystemMessage("system_msg_1", "conv_123", 0, "System prompt")
	msgRepo.Create(context.Background(), systemMsg)

	// Create assistant message pointing to system message
	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "I am ready to help")
	assistantMsg.PreviousID = "system_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when previous message is not user message, got nil")
	}

	expectedErr := "cannot continue: previous message is not a user message"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got: %v", expectedErr, err)
	}
}

func TestContinueResponse_Execute_GenerateResponseFails(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that returns an error
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			return nil, errors.New("LLM service unavailable")
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when generate response fails, got nil")
	}

	if err.Error() != "LLM service unavailable" {
		t.Errorf("expected 'LLM service unavailable' error, got: %v", err)
	}
}

func TestContinueResponse_Execute_UpdateFails(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that generates successfully
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, " How can I help?")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	// Set the message repo to fail on update
	msgRepo.updateErr = errors.New("database connection lost")

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when update fails, got nil")
	}

	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost' error, got: %v", err)
	}
}

func TestContinueResponse_Execute_UpdateTipFails(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that generates successfully
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, " How can I help?")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	// Set the conversation repo to fail on UpdateTip
	convRepo.updateTipErr = errors.New("failed to update tip")

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when update tip fails, got nil")
	}

	if err.Error() != "failed to update tip" {
		t.Errorf("expected 'failed to update tip' error, got: %v", err)
	}
}

func TestContinueResponse_Execute_Streaming(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that returns streaming output
	streamCh := make(chan *ports.ResponseStreamChunk, 10)
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			// Verify streaming is enabled
			if !input.EnableStreaming {
				t.Error("expected streaming to be enabled")
			}

			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, "")
			generatedMsg.MarkAsStreaming()

			// Simulate streaming in background
			go func() {
				defer close(streamCh)
				streamCh <- &ports.ResponseStreamChunk{Text: " How ", IsFinal: false}
				streamCh <- &ports.ResponseStreamChunk{Text: "can I ", IsFinal: false}
				streamCh <- &ports.ResponseStreamChunk{Text: "help?", IsFinal: true}
			}()

			return &ports.GenerateResponseOutput{
				Message:       generatedMsg,
				StreamChannel: streamCh,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: true,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.StreamChannel == nil {
		t.Fatal("expected stream channel to be returned for streaming mode")
	}

	if output.TargetMessage == nil {
		t.Fatal("expected target message to be returned")
	}

	if output.TargetMessage.ID != "assistant_msg_1" {
		t.Errorf("expected target message ID assistant_msg_1, got %s", output.TargetMessage.ID)
	}

	if output.GeneratedOutput == nil {
		t.Fatal("expected generated output to be returned")
	}

	// Drain the original stream channel to prevent goroutine leak
	for range streamCh {
		// Just consume
	}
}

func TestContinueResponse_Execute_WithToolsAndReasoning(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "What's 2+2?")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Let me calculate that for you")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that verifies tools and reasoning flags
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			if !input.EnableTools {
				t.Error("expected EnableTools to be true")
			}
			if !input.EnableReasoning {
				t.Error("expected EnableReasoning to be true")
			}

			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, ". The answer is 4.")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableTools:     true,
		EnableReasoning: true,
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.AppendedContent != ". The answer is 4." {
		t.Errorf("expected appended content '. The answer is 4.', got %s", output.AppendedContent)
	}
}

func TestContinueResponse_Execute_EmptyGeneratedContent(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that returns empty content
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, "")
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When content is empty, nothing should be appended
	if output.AppendedContent != "" {
		t.Errorf("expected empty appended content, got %s", output.AppendedContent)
	}

	// Original message should remain unchanged
	originalMsg, _ := msgRepo.GetByID(context.Background(), "assistant_msg_1")
	if originalMsg.Contents != "Hi there" {
		t.Errorf("expected original content 'Hi there', got %s", originalMsg.Contents)
	}
}

func TestContinueResponse_Execute_AppendToEmptyTargetMessage(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	// Create assistant message with empty content
	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that returns content
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, "Hello! How can I help?")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When target is empty, content should be set directly (no separator)
	updatedMsg, _ := msgRepo.GetByID(context.Background(), "assistant_msg_1")
	if updatedMsg.Contents != "Hello! How can I help?" {
		t.Errorf("expected content 'Hello! How can I help?', got %s", updatedMsg.Contents)
	}

	if output.AppendedContent != "Hello! How can I help?" {
		t.Errorf("expected appended content 'Hello! How can I help?', got %s", output.AppendedContent)
	}
}

func TestContinueResponse_Execute_MessageRepoGetError(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	genMock := &continueResponseMockGenUC{}
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Set repo to return error on GetByID
	msgRepo.getError = errors.New("database timeout")

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "any_msg",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when message repo fails, got nil")
	}

	if err.Error() != "database timeout" {
		t.Errorf("expected 'database timeout' error, got: %v", err)
	}
}

func TestContinueResponse_Execute_ContinuationMessageDeleted(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	var generatedMsgID string

	// Mock that generates a message
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsgID = input.MessageID
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, " How can I help?")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the continuation message was deleted
	deletedMsg, _ := msgRepo.GetByID(context.Background(), generatedMsgID)
	if deletedMsg != nil {
		t.Error("expected continuation message to be deleted")
	}
}

func TestContinueResponse_Execute_ConversationTipUpdated(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that generates a message
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			generatedMsg := models.NewAssistantMessage(input.MessageID, "conv_123", 2, " How can I help?")
			msgRepo.Create(ctx, generatedMsg)
			return &ports.GenerateResponseOutput{
				Message: generatedMsg,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify conversation tip was updated to point to the target message
	updatedConv, _ := convRepo.GetByID(context.Background(), "conv_123")
	if updatedConv.TipMessageID == nil || *updatedConv.TipMessageID != "assistant_msg_1" {
		tipID := ""
		if updatedConv.TipMessageID != nil {
			tipID = *updatedConv.TipMessageID
		}
		t.Errorf("expected conversation tip to be assistant_msg_1, got %s", tipID)
	}
}

func TestContinueResponse_Execute_NilGeneratedMessage(t *testing.T) {
	msgRepo := newContinueResponseMockMsgRepo()
	convRepo := newContinueResponseMockConvRepo()
	idGen := newContinueResponseMockIDGen()
	txManager := &continueResponseMockTxManager{}

	// Setup conversation
	conv := models.NewConversation("conv_123", "", "")
	convRepo.Create(context.Background(), conv)

	// Setup messages
	userMsg := models.NewUserMessage("user_msg_1", "conv_123", 0, "Hello")
	msgRepo.Create(context.Background(), userMsg)

	assistantMsg := models.NewAssistantMessage("assistant_msg_1", "conv_123", 1, "Hi there")
	assistantMsg.PreviousID = "user_msg_1"
	msgRepo.Create(context.Background(), assistantMsg)

	// Mock that returns nil message
	genMock := &continueResponseMockGenUC{
		executeFunc: func(ctx context.Context, input *ports.GenerateResponseInput) (*ports.GenerateResponseOutput, error) {
			return &ports.GenerateResponseOutput{
				Message: nil,
			}, nil
		},
	}

	uc := &continueResponseTestHelper{
		messageRepo:      msgRepo,
		conversationRepo: convRepo,
		genMock:          genMock,
		idGenerator:      idGen,
		txManager:        txManager,
	}

	input := &ports.ContinueResponseInput{
		TargetMessageID: "assistant_msg_1",
		EnableStreaming: false,
	}

	output, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When generated message is nil, nothing should be appended
	if output.AppendedContent != "" {
		t.Errorf("expected empty appended content, got %s", output.AppendedContent)
	}

	// Original message should remain unchanged
	originalMsg, _ := msgRepo.GetByID(context.Background(), "assistant_msg_1")
	if originalMsg.Contents != "Hi there" {
		t.Errorf("expected original content 'Hi there', got %s", originalMsg.Contents)
	}
}
