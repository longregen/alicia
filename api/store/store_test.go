package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/api/domain"
)

var testStore *Store

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Use DATABASE_URL or default to local test database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres@localhost:5555/alicia?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic("failed to ping database: " + err.Error())
	}

	testStore = New(pool)

	os.Exit(m.Run())
}

func TestConversations(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-" + NewID("u")

	// Create
	conv := &domain.Conversation{
		ID:        NewConversationID(),
		UserID:    userID,
		Title:     "Test Conversation",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := testStore.CreateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}
	t.Logf("Created conversation: %s", conv.ID)

	// Get
	got, err := testStore.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if got.Title != conv.Title {
		t.Errorf("Title mismatch: got %q, want %q", got.Title, conv.Title)
	}

	// GetByUser
	got, err = testStore.GetConversationByUser(ctx, conv.ID, userID)
	if err != nil {
		t.Fatalf("GetConversationByUser failed: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("UserID mismatch: got %q, want %q", got.UserID, userID)
	}

	// Update
	conv.Title = "Updated Title"
	err = testStore.UpdateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("UpdateConversation failed: %v", err)
	}

	got, err = testStore.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation after update failed: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title not updated: got %q", got.Title)
	}

	// List
	convs, total, err := testStore.ListConversations(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected at least 1 conversation, got %d", total)
	}
	t.Logf("Listed %d conversations (total: %d)", len(convs), total)

	// Delete
	err = testStore.DeleteConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}

	_, err = testStore.GetConversation(ctx, conv.ID)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestMessages(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-" + NewID("u")

	// Create conversation first
	conv := &domain.Conversation{
		ID:        NewConversationID(),
		UserID:    userID,
		Title:     "Test Conversation for Messages",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := testStore.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}
	t.Logf("Created conversation: %s", conv.ID)

	// Create first message (no previous_id - root message)
	msg1 := &domain.Message{
		ID:             NewMessageID(),
		ConversationID: conv.ID,
		PreviousID:     nil, // Root message
		Role:           "user",
		Content:        "Hello, this is the first message",
		Reasoning:      "",
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err := testStore.CreateMessage(ctx, msg1)
	if err != nil {
		t.Fatalf("CreateMessage (root) failed: %v", err)
	}
	t.Logf("Created root message: %s (branch_index: %d)", msg1.ID, msg1.BranchIndex)

	// Get message
	got, err := testStore.GetMessage(ctx, msg1.ID)
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if got.Content != msg1.Content {
		t.Errorf("Content mismatch: got %q, want %q", got.Content, msg1.Content)
	}

	// Create second message (with previous_id)
	msg2 := &domain.Message{
		ID:             NewMessageID(),
		ConversationID: conv.ID,
		PreviousID:     &msg1.ID,
		Role:           "assistant",
		Content:        "Hello! How can I help you?",
		Reasoning:      "User greeted me",
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err = testStore.CreateMessage(ctx, msg2)
	if err != nil {
		t.Fatalf("CreateMessage (with previous) failed: %v", err)
	}
	t.Logf("Created reply message: %s (branch_index: %d)", msg2.ID, msg2.BranchIndex)

	// Create a sibling message (same previous_id as msg2)
	msg3 := &domain.Message{
		ID:             NewMessageID(),
		ConversationID: conv.ID,
		PreviousID:     &msg1.ID, // Same parent as msg2
		Role:           "assistant",
		Content:        "Hi there! What's up?",
		Reasoning:      "Alternative greeting",
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}

	err = testStore.CreateMessage(ctx, msg3)
	if err != nil {
		t.Fatalf("CreateMessage (sibling) failed: %v", err)
	}
	t.Logf("Created sibling message: %s (branch_index: %d)", msg3.ID, msg3.BranchIndex)

	if msg3.BranchIndex != msg2.BranchIndex+1 {
		t.Errorf("Sibling branch_index wrong: got %d, want %d", msg3.BranchIndex, msg2.BranchIndex+1)
	}

	// Update message
	msg1.Content = "Updated content"
	err = testStore.UpdateMessage(ctx, msg1)
	if err != nil {
		t.Fatalf("UpdateMessage failed: %v", err)
	}

	// List messages
	msgs, err := testStore.ListMessages(ctx, conv.ID, 100)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(msgs))
	}

	// Get message chain
	chain, err := testStore.GetMessageChain(ctx, msg2.ID)
	if err != nil {
		t.Fatalf("GetMessageChain failed: %v", err)
	}
	if len(chain) != 2 {
		t.Errorf("Expected chain of 2, got %d", len(chain))
	}

	// Get siblings
	siblings, err := testStore.GetMessageSiblings(ctx, msg2.ID)
	if err != nil {
		t.Fatalf("GetMessageSiblings failed: %v", err)
	}
	if len(siblings) != 2 {
		t.Errorf("Expected 2 siblings, got %d", len(siblings))
	}

	// Update conversation tip
	err = testStore.UpdateConversationTip(ctx, conv.ID, msg2.ID)
	if err != nil {
		t.Fatalf("UpdateConversationTip failed: %v", err)
	}

	// Delete message
	err = testStore.DeleteMessage(ctx, msg3.ID)
	if err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}

	_, err = testStore.GetMessage(ctx, msg3.ID)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}

	// Cleanup
	testStore.DeleteConversation(ctx, conv.ID)
}

func TestMemories(t *testing.T) {
	ctx := context.Background()

	// Create memory
	mem := &domain.Memory{
		ID:         NewMemoryID(),
		Content:    "User prefers dark mode",
		Importance: 0.7,
		Pinned:     false,
		Archived:   false,
		Tags:       []string{"preference", "ui"},
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	err := testStore.CreateMemory(ctx, mem)
	if err != nil {
		t.Fatalf("CreateMemory failed: %v", err)
	}
	t.Logf("Created memory: %s", mem.ID)

	// Get
	got, err := testStore.GetMemory(ctx, mem.ID)
	if err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}
	if got.Content != mem.Content {
		t.Errorf("Content mismatch: got %q, want %q", got.Content, mem.Content)
	}

	// Update
	mem.Importance = 0.9
	mem.Pinned = true
	err = testStore.UpdateMemory(ctx, mem)
	if err != nil {
		t.Fatalf("UpdateMemory failed: %v", err)
	}

	got, err = testStore.GetMemory(ctx, mem.ID)
	if err != nil {
		t.Fatalf("GetMemory after update failed: %v", err)
	}
	if got.Importance != 0.9 {
		t.Errorf("Importance not updated: got %f", got.Importance)
	}
	if !got.Pinned {
		t.Error("Pinned not updated")
	}

	// List
	mems, total, err := testStore.ListMemories(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListMemories failed: %v", err)
	}
	t.Logf("Listed %d memories (total: %d)", len(mems), total)

	// Get by tags
	mems, err = testStore.GetMemoriesByTags(ctx, []string{"preference"}, 10)
	if err != nil {
		t.Fatalf("GetMemoriesByTags failed: %v", err)
	}
	found := false
	for _, m := range mems {
		if m.ID == mem.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Memory not found by tag")
	}

	// Delete
	reason := "test cleanup"
	err = testStore.DeleteMemory(ctx, mem.ID, &reason)
	if err != nil {
		t.Fatalf("DeleteMemory failed: %v", err)
	}

	_, err = testStore.GetMemory(ctx, mem.ID)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestMessageFeedback(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-" + NewID("u")

	// Create conversation and message
	conv := &domain.Conversation{
		ID:        NewConversationID(),
		UserID:    userID,
		Title:     "Test Feedback",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := testStore.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	msg := &domain.Message{
		ID:             NewMessageID(),
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        "Test message for feedback",
		Status:         domain.MessageStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	}
	if err := testStore.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	// Create feedback
	fb := &domain.MessageFeedback{
		ID:        NewMessageFeedbackID(),
		MessageID: msg.ID,
		Rating:    1, // thumbs up
		Note:      "Great response!",
		CreatedAt: time.Now().UTC(),
	}

	err := testStore.CreateMessageFeedback(ctx, fb)
	if err != nil {
		t.Fatalf("CreateMessageFeedback failed: %v", err)
	}
	t.Logf("Created feedback: %s", fb.ID)

	// Get feedback
	got, err := testStore.GetMessageFeedbackByMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetMessageFeedbackByMessage failed: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("Expected 1 feedback, got %d", len(got))
	}
	if got[0].Rating != 1 {
		t.Errorf("Rating mismatch: got %d, want 1", got[0].Rating)
	}

	// Cleanup
	testStore.DeleteConversation(ctx, conv.ID)
}

func TestMCPServers(t *testing.T) {
	ctx := context.Background()

	// Create MCP server
	server := &domain.MCPServer{
		ID:            NewMCPServerID(),
		Name:          "test-server-" + NewID("s"),
		TransportType: "stdio",
		Command:       "test-command",
		Args:          []string{"--arg1", "--arg2"},
		Enabled:       true,
		CreatedAt:     time.Now().UTC(),
	}

	err := testStore.CreateMCPServer(ctx, server)
	if err != nil {
		t.Fatalf("CreateMCPServer failed: %v", err)
	}
	t.Logf("Created MCP server: %s", server.ID)

	// Get by name
	got, err := testStore.GetMCPServerByName(ctx, server.Name)
	if err != nil {
		t.Fatalf("GetMCPServerByName failed: %v", err)
	}
	if got.Command != server.Command {
		t.Errorf("Command mismatch: got %q, want %q", got.Command, server.Command)
	}

	// Update
	server.Enabled = false
	err = testStore.UpdateMCPServer(ctx, server)
	if err != nil {
		t.Fatalf("UpdateMCPServer failed: %v", err)
	}

	// List enabled
	servers, err := testStore.ListEnabledMCPServers(ctx)
	if err != nil {
		t.Fatalf("ListEnabledMCPServers failed: %v", err)
	}
	for _, s := range servers {
		if s.ID == server.ID {
			t.Error("Disabled server should not be in enabled list")
		}
	}

	// Delete by name
	err = testStore.DeleteMCPServerByName(ctx, server.Name)
	if err != nil {
		t.Fatalf("DeleteMCPServerByName failed: %v", err)
	}

	_, err = testStore.GetMCPServerByName(ctx, server.Name)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got: %v", err)
	}
}

func TestTools(t *testing.T) {
	ctx := context.Background()

	// Create tool
	tool := &domain.Tool{
		ID:          NewToolID(),
		Name:        "test-tool-" + NewID("t"),
		Description: "A test tool",
		Schema:      map[string]any{"type": "object"},
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err := testStore.CreateTool(ctx, tool)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}
	t.Logf("Created tool: %s", tool.ID)

	// Get by name
	got, err := testStore.GetToolByName(ctx, tool.Name)
	if err != nil {
		t.Fatalf("GetToolByName failed: %v", err)
	}
	if got.Description != tool.Description {
		t.Errorf("Description mismatch: got %q, want %q", got.Description, tool.Description)
	}

	// List enabled
	tools, err := testStore.ListEnabledTools(ctx)
	if err != nil {
		t.Fatalf("ListEnabledTools failed: %v", err)
	}
	found := false
	for _, tl := range tools {
		if tl.ID == tool.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Tool not found in enabled list")
	}

	// Update - disable
	tool.Enabled = false
	err = testStore.UpdateTool(ctx, tool)
	if err != nil {
		t.Fatalf("UpdateTool failed: %v", err)
	}

	// Verify disabled tool not in enabled list
	tools, err = testStore.ListEnabledTools(ctx)
	if err != nil {
		t.Fatalf("ListEnabledTools after disable failed: %v", err)
	}
	for _, tl := range tools {
		if tl.ID == tool.ID {
			t.Error("Disabled tool should not be in enabled list")
		}
	}
}

func TestTransactions(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-" + NewID("u")

	convID := NewConversationID()

	// Test successful transaction
	err := testStore.WithTx(ctx, func(txCtx context.Context) error {
		conv := &domain.Conversation{
			ID:        convID,
			UserID:    userID,
			Title:     "Transaction Test",
			Status:    "active",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := testStore.CreateConversation(txCtx, conv); err != nil {
			return err
		}

		msg := &domain.Message{
			ID:             NewMessageID(),
			ConversationID: conv.ID,
			Role:           "user",
			Content:        "Test in transaction",
			Status:         domain.MessageStatusCompleted,
			CreatedAt:      time.Now().UTC(),
		}
		return testStore.CreateMessage(txCtx, msg)
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify data persisted
	_, err = testStore.GetConversation(ctx, convID)
	if err != nil {
		t.Fatalf("GetConversation after tx failed: %v", err)
	}

	// Cleanup
	testStore.DeleteConversation(ctx, convID)
}
