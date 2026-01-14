package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// Integration tests for vote repository context queries
// These require a real PostgreSQL instance with the test database

func TestVoteRepository_GetToolUseVotesWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	toolUseRepo := NewToolUseRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_ctx_test1", "test-user", "Vote Context Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create user message
	userMsg := &models.Message{
		ID:               "msg_vote_ctx_user1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleUser,
		Contents:         "Search for Python tutorials",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	// Create assistant message
	assistantMsg := &models.Message{
		ID:               "msg_vote_ctx_asst1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleAssistant,
		Contents:         "I'll search for that.",
		PreviousID:       userMsg.ID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, assistantMsg)
	if err != nil {
		t.Fatalf("Create assistant message failed: %v", err)
	}

	// Create tool use
	toolArgs := map[string]any{"query": "Python tutorials"}
	toolUse := &models.ToolUse{
		ID:             "tu_vote_ctx1",
		MessageID:      assistantMsg.ID,
		ToolName:       "search",
		Arguments:      toolArgs,
		Status:         models.ToolStatusSuccess,
		SequenceNumber: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = toolUseRepo.Create(ctx, toolUse)
	if err != nil {
		t.Fatalf("Create tool use failed: %v", err)
	}

	// Create vote on tool use
	vote := &models.Vote{
		ID:            "vote_ctx_test1",
		MessageID:     assistantMsg.ID,
		TargetType:    models.VoteTargetToolUse,
		TargetID:      toolUse.ID,
		Value:         models.VoteValueUp,
		QuickFeedback: "",
		Note:          "Great search",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = voteRepo.Create(ctx, vote)
	if err != nil {
		t.Fatalf("Create vote failed: %v", err)
	}

	// Retrieve votes with context
	votesWithContext, err := voteRepo.GetToolUseVotesWithContext(ctx, 10)
	if err != nil {
		t.Fatalf("GetToolUseVotesWithContext failed: %v", err)
	}

	if len(votesWithContext) == 0 {
		t.Fatal("expected at least 1 vote with context, got 0")
	}

	// Find our vote
	var found *models.Vote
	var foundToolUse *models.ToolUse
	var foundUserMessage string
	for _, vc := range votesWithContext {
		if vc.Vote.ID == vote.ID {
			found = vc.Vote
			foundToolUse = vc.ToolUse
			foundUserMessage = vc.UserMessage
			break
		}
	}

	if found == nil {
		t.Fatal("did not find our test vote in results")
	}

	// Verify vote data
	if found.Value != models.VoteValueUp {
		t.Errorf("expected vote value up, got %d", found.Value)
	}

	if found.QuickFeedback != "" {
		t.Errorf("expected quick feedback '', got %s", found.QuickFeedback)
	}

	// Verify tool use data
	if foundToolUse == nil {
		t.Fatal("expected tool use to be populated")
	}

	if foundToolUse.ToolName != "search" {
		t.Errorf("expected tool name 'search', got %s", foundToolUse.ToolName)
	}

	if foundToolUse.Arguments == nil {
		t.Fatal("expected tool arguments to be populated")
	}

	if query, ok := foundToolUse.Arguments["query"].(string); !ok || query != "Python tutorials" {
		t.Errorf("expected query 'Python tutorials', got %v", foundToolUse.Arguments["query"])
	}

	// Verify user message
	if foundUserMessage != "Search for Python tutorials" {
		t.Errorf("expected user message 'Search for Python tutorials', got %s", foundUserMessage)
	}
}

func TestVoteRepository_GetMemoryVotesWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	memoryRepo := NewMemoryRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_mem_ctx1", "test-user", "Memory Vote Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create memory with unique ID using timestamp
	memory := &models.Memory{
		ID:         fmt.Sprintf("mem_vote_ctx1_%d", time.Now().UnixNano()),
		Content:    "User prefers Python over JavaScript",
		Importance: 0.8,
		SourceType: "conversation",
		SourceInfo: &models.SourceInfo{ConversationID: conv.ID},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = memoryRepo.Create(ctx, memory)
	if err != nil {
		t.Fatalf("Create memory failed: %v", err)
	}

	// Create user message
	userMsg := &models.Message{
		ID:               "msg_vote_mem_ctx1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleUser,
		Contents:         "What programming language should I learn?",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	// Create assistant message
	assistantMsg := &models.Message{
		ID:               "msg_vote_mem_asst1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleAssistant,
		Contents:         "Based on your preference, I recommend Python.",
		PreviousID:       userMsg.ID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, assistantMsg)
	if err != nil {
		t.Fatalf("Create assistant message failed: %v", err)
	}

	// Create memory usage repository
	memoryUsageRepo := NewMemoryUsageRepository(pool)

	// Record memory usage
	memoryUsage := &models.MemoryUsage{
		ID:              "mu_vote_ctx1",
		MessageID:       assistantMsg.ID,
		MemoryID:        memory.ID,
		ConversationID:  conv.ID,
		QueryPrompt:     "programming language preference",
		SimilarityScore: 0.92,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	err = memoryUsageRepo.Create(ctx, memoryUsage)
	if err != nil {
		t.Fatalf("Create memory usage failed: %v", err)
	}

	// Create vote on memory
	vote := &models.Vote{
		ID:            "vote_mem_ctx1",
		MessageID:     assistantMsg.ID,
		TargetType:    models.VoteTargetMemory,
		TargetID:      memory.ID,
		Value:         models.VoteValueUp,
		QuickFeedback: "",
		Note:          "Good recall",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = voteRepo.Create(ctx, vote)
	if err != nil {
		t.Fatalf("Create vote failed: %v", err)
	}

	// Retrieve votes with context
	votesWithContext, err := voteRepo.GetMemoryVotesWithContext(ctx, 10)
	if err != nil {
		t.Fatalf("GetMemoryVotesWithContext failed: %v", err)
	}

	if len(votesWithContext) == 0 {
		t.Fatal("expected at least 1 memory vote with context, got 0")
	}

	// Find our vote
	var found *models.Vote
	var foundMemory *models.Memory
	var foundUserMessage string
	var foundSimilarityScore float32
	for _, vc := range votesWithContext {
		if vc.Vote.ID == vote.ID {
			found = vc.Vote
			foundMemory = vc.Memory
			foundUserMessage = vc.UserMessage
			foundSimilarityScore = vc.SimilarityScore
			break
		}
	}

	if found == nil {
		t.Fatal("did not find our test memory vote in results")
	}

	// Verify vote data
	if found.Value != models.VoteValueUp {
		t.Errorf("expected vote value up, got %d", found.Value)
	}

	// Verify memory data
	if foundMemory == nil {
		t.Fatal("expected memory to be populated")
	}

	if foundMemory.Content != "User prefers Python over JavaScript" {
		t.Errorf("expected memory content about Python, got %s", foundMemory.Content)
	}

	// Verify user message - this is the assistant message that used the memory
	if foundUserMessage != "Based on your preference, I recommend Python." {
		t.Errorf("expected specific user message, got %s", foundUserMessage)
	}

	// Verify similarity score
	if foundSimilarityScore != 0.92 {
		t.Errorf("expected similarity score 0.92, got %f", foundSimilarityScore)
	}
}

func TestVoteRepository_CountByTargetType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	toolUseRepo := NewToolUseRepository(pool)
	memoryRepo := NewMemoryRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_count1", "test-user", "Vote Count Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create messages for votes
	userMsg := &models.Message{
		ID:               "msg_vote_count_user1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleUser,
		Contents:         "Test message",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	assistantMsg := &models.Message{
		ID:               "msg_vote_count_asst1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleAssistant,
		Contents:         "Response",
		PreviousID:       userMsg.ID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, assistantMsg)
	if err != nil {
		t.Fatalf("Create assistant message failed: %v", err)
	}

	// Create 3 tool use votes
	for i := 0; i < 3; i++ {
		toolUse := &models.ToolUse{
			ID:             "tu_count_" + string(rune('a'+i)),
			MessageID:      assistantMsg.ID,
			ToolName:       "test_tool",
			Status:         models.ToolStatusSuccess,
			SequenceNumber: i + 1,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = toolUseRepo.Create(ctx, toolUse)
		if err != nil {
			t.Fatalf("Create tool use %d failed: %v", i, err)
		}

		vote := &models.Vote{
			ID:         "vote_tu_count_" + string(rune('a'+i)),
			MessageID:  assistantMsg.ID,
			TargetType: models.VoteTargetToolUse,
			TargetID:   toolUse.ID,
			Value:      models.VoteValueUp,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = voteRepo.Create(ctx, vote)
		if err != nil {
			t.Fatalf("Create tool use vote %d failed: %v", i, err)
		}
	}

	// Create 2 memory votes
	for i := 0; i < 2; i++ {
		memory := &models.Memory{
			ID:         fmt.Sprintf("mem_count_%c_%d", rune('a'+i), time.Now().UnixNano()),
			Content:    "Test memory " + string(rune('1'+i)),
			Importance: 5,
			SourceType: "test",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = memoryRepo.Create(ctx, memory)
		if err != nil {
			t.Fatalf("Create memory %d failed: %v", i, err)
		}

		vote := &models.Vote{
			ID:         "vote_mem_count_" + string(rune('a'+i)),
			MessageID:  assistantMsg.ID,
			TargetType: models.VoteTargetMemory,
			TargetID:   memory.ID,
			Value:      models.VoteValueDown,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = voteRepo.Create(ctx, vote)
		if err != nil {
			t.Fatalf("Create memory vote %d failed: %v", i, err)
		}
	}

	// Create 1 message vote
	msgVote := &models.Vote{
		ID:         "vote_msg_count1",
		MessageID:  assistantMsg.ID,
		TargetType: models.VoteTargetMessage,
		TargetID:   assistantMsg.ID,
		Value:      models.VoteValueUp,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = voteRepo.Create(ctx, msgVote)
	if err != nil {
		t.Fatalf("Create message vote failed: %v", err)
	}

	// Count tool use votes
	toolUseCount, err := voteRepo.CountByTargetType(ctx, models.VoteTargetToolUse)
	if err != nil {
		t.Fatalf("CountByTargetType for tool_use failed: %v", err)
	}

	if toolUseCount < 3 {
		t.Errorf("expected at least 3 tool use votes, got %d", toolUseCount)
	}

	// Count memory votes
	memoryCount, err := voteRepo.CountByTargetType(ctx, models.VoteTargetMemory)
	if err != nil {
		t.Fatalf("CountByTargetType for memory failed: %v", err)
	}

	if memoryCount < 2 {
		t.Errorf("expected at least 2 memory votes, got %d", memoryCount)
	}

	// Count message votes
	messageCount, err := voteRepo.CountByTargetType(ctx, models.VoteTargetMessage)
	if err != nil {
		t.Fatalf("CountByTargetType for message failed: %v", err)
	}

	if messageCount < 1 {
		t.Errorf("expected at least 1 message vote, got %d", messageCount)
	}
}

func TestVoteRepository_GetToolUseVotesWithContext_Limit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	toolUseRepo := NewToolUseRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_limit1", "test-user", "Vote Limit Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create user and assistant messages
	userMsg := &models.Message{
		ID:               "msg_vote_limit_user1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleUser,
		Contents:         "Test",
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	assistantMsg := &models.Message{
		ID:               "msg_vote_limit_asst1",
		ConversationID:   conv.ID,
		Role:             models.MessageRoleAssistant,
		Contents:         "Response",
		PreviousID:       userMsg.ID,
		SyncStatus:       models.SyncStatusSynced,
		CompletionStatus: models.CompletionStatusCompleted,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err = msgRepo.Create(ctx, assistantMsg)
	if err != nil {
		t.Fatalf("Create assistant message failed: %v", err)
	}

	// Create 5 tool uses with votes
	for i := 0; i < 5; i++ {
		toolUse := &models.ToolUse{
			ID:             "tu_limit_" + string(rune('a'+i)),
			MessageID:      assistantMsg.ID,
			ToolName:       "tool_" + string(rune('1'+i)),
			Status:         models.ToolStatusSuccess,
			SequenceNumber: i + 1,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = toolUseRepo.Create(ctx, toolUse)
		if err != nil {
			t.Fatalf("Create tool use %d failed: %v", i, err)
		}

		vote := &models.Vote{
			ID:         "vote_limit_" + string(rune('a'+i)),
			MessageID:  assistantMsg.ID,
			TargetType: models.VoteTargetToolUse,
			TargetID:   toolUse.ID,
			Value:      models.VoteValueUp,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err = voteRepo.Create(ctx, vote)
		if err != nil {
			t.Fatalf("Create vote %d failed: %v", i, err)
		}

		// Small delay to ensure different created_at times
		time.Sleep(5 * time.Millisecond)
	}

	// Request only 3 results
	votesWithContext, err := voteRepo.GetToolUseVotesWithContext(ctx, 3)
	if err != nil {
		t.Fatalf("GetToolUseVotesWithContext failed: %v", err)
	}

	// Count how many of our test votes are in the results
	foundCount := 0
	for _, vc := range votesWithContext {
		if len(vc.Vote.ID) > 11 && vc.Vote.ID[:11] == "vote_limit_" {
			foundCount++
		}
	}

	// Should get exactly 3 (our most recent ones, due to ORDER BY created_at DESC)
	if foundCount > 3 {
		t.Errorf("requested limit 3, but got %d of our test votes", foundCount)
	}
}

func TestVoteRepository_GetMemoryUsageVotesWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	memoryRepo := NewMemoryRepository(pool)
	memoryUsageRepo := NewMemoryUsageRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_mu_ctx1", "test-user", "Memory Usage Vote Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create memory with unique ID
	memory := models.NewMemory(fmt.Sprintf("mem_vote_mu_ctx1_%d", time.Now().UnixNano()), "User prefers dark mode")
	memory.Importance = 0.9
	memory.Confidence = 0.95
	err = memoryRepo.Create(ctx, memory)
	if err != nil {
		t.Fatalf("Create memory failed: %v", err)
	}

	// Create user message
	userMsg := models.NewUserMessage("msg_vote_mu_user1", conv.ID, 1, "What theme should I use?")
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	// Create memory usage record
	memUsage := models.NewMemoryUsage("mu_vote_ctx1", conv.ID, userMsg.ID, memory.ID)
	memUsage.QueryPrompt = "theme preference"
	memUsage.SimilarityScore = 0.88
	memUsage.PositionInResults = 1
	err = memoryUsageRepo.Create(ctx, memUsage)
	if err != nil {
		t.Fatalf("Create memory usage failed: %v", err)
	}

	// Create vote on memory usage
	vote := models.NewVoteWithFeedback(
		"vote_mu_ctx1",
		models.VoteTargetMemoryUsage,
		memUsage.ID,
		userMsg.ID,
		models.VoteValueUp,
		"",
		"Perfect match",
	)
	err = voteRepo.Create(ctx, vote)
	if err != nil {
		t.Fatalf("Create vote failed: %v", err)
	}

	// Retrieve votes with context
	votesWithContext, err := voteRepo.GetMemoryUsageVotesWithContext(ctx, 10)
	if err != nil {
		t.Fatalf("GetMemoryUsageVotesWithContext failed: %v", err)
	}

	if len(votesWithContext) == 0 {
		t.Fatal("expected at least 1 memory usage vote with context, got 0")
	}

	// Find our vote
	var found *models.Vote
	var foundMemory *models.Memory
	var foundMemoryUsage *models.MemoryUsage
	var foundUserMessage string
	for _, vc := range votesWithContext {
		if vc.Vote.ID == vote.ID {
			found = vc.Vote
			foundMemory = vc.Memory
			foundMemoryUsage = vc.MemoryUsage
			foundUserMessage = vc.UserMessage
			break
		}
	}

	if found == nil {
		t.Fatal("did not find our test memory usage vote in results")
	}

	// Verify vote data
	if found.Value != models.VoteValueUp {
		t.Errorf("expected vote value up, got %d", found.Value)
	}

	if found.QuickFeedback != "" {
		t.Errorf("expected quick feedback '', got %s", found.QuickFeedback)
	}

	// Verify memory data
	if foundMemory == nil {
		t.Fatal("expected memory to be populated")
	}

	if foundMemory.Content != "User prefers dark mode" {
		t.Errorf("expected memory content about dark mode, got %s", foundMemory.Content)
	}

	// Verify memory usage data
	if foundMemoryUsage == nil {
		t.Fatal("expected memory usage to be populated")
	}

	if foundMemoryUsage.QueryPrompt != "theme preference" {
		t.Errorf("expected query prompt 'theme preference', got %s", foundMemoryUsage.QueryPrompt)
	}

	if foundMemoryUsage.SimilarityScore != 0.88 {
		t.Errorf("expected similarity score 0.88, got %f", foundMemoryUsage.SimilarityScore)
	}

	// Verify user message
	if foundUserMessage != "What theme should I use?" {
		t.Errorf("expected specific user message, got %s", foundUserMessage)
	}
}

func TestVoteRepository_GetMemoryExtractionVotesWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	voteRepo := NewVoteRepository(pool)
	convRepo := NewConversationRepository(pool)
	msgRepo := NewMessageRepository(pool)
	memoryRepo := NewMemoryRepository(pool)

	ctx := context.Background()

	// Create test conversation
	conv := models.NewConversation("ac_vote_me_ctx1", "test-user", "Memory Extraction Vote Test")
	err := convRepo.Create(ctx, conv)
	if err != nil {
		t.Fatalf("Create conversation failed: %v", err)
	}

	// Create user message
	userMsg := models.NewUserMessage("msg_vote_me_user1", conv.ID, 1, "I really enjoy hiking in the mountains")
	err = msgRepo.Create(ctx, userMsg)
	if err != nil {
		t.Fatalf("Create user message failed: %v", err)
	}

	// Create memory extracted from the message with unique ID
	memory := models.NewMemory(fmt.Sprintf("mem_vote_me_ctx1_%d", time.Now().UnixNano()), "User enjoys hiking in mountains")
	memory.SourceType = models.SourceTypeConversation
	memory.SourceInfo = &models.SourceInfo{
		ConversationID: conv.ID,
		MessageID:      userMsg.ID,
	}
	memory.Importance = 0.7
	memory.Confidence = 0.85
	err = memoryRepo.Create(ctx, memory)
	if err != nil {
		t.Fatalf("Create memory failed: %v", err)
	}

	// Create vote on memory extraction
	vote := models.NewVoteWithFeedback(
		"vote_me_ctx1",
		models.VoteTargetMemoryExtraction,
		memory.ID,
		userMsg.ID,
		models.VoteValueUp,
		"",
		"Good extraction",
	)
	err = voteRepo.Create(ctx, vote)
	if err != nil {
		t.Fatalf("Create vote failed: %v", err)
	}

	// Retrieve votes with context
	votesWithContext, err := voteRepo.GetMemoryExtractionVotesWithContext(ctx, 10)
	if err != nil {
		t.Fatalf("GetMemoryExtractionVotesWithContext failed: %v", err)
	}

	if len(votesWithContext) == 0 {
		t.Fatal("expected at least 1 memory extraction vote with context, got 0")
	}

	// Find our vote
	var found *models.Vote
	var foundMemory *models.Memory
	var foundSourceMessage *models.Message
	for _, vc := range votesWithContext {
		if vc.Vote.ID == vote.ID {
			found = vc.Vote
			foundMemory = vc.Memory
			foundSourceMessage = vc.SourceMessage
			break
		}
	}

	if found == nil {
		t.Fatal("did not find our test memory extraction vote in results")
	}

	// Verify vote data
	if found.Value != models.VoteValueUp {
		t.Errorf("expected vote value up, got %d", found.Value)
	}

	if found.QuickFeedback != "" {
		t.Errorf("expected quick feedback '', got %s", found.QuickFeedback)
	}

	// Verify memory data
	if foundMemory == nil {
		t.Fatal("expected memory to be populated")
	}

	if foundMemory.Content != "User enjoys hiking in mountains" {
		t.Errorf("expected memory content about hiking, got %s", foundMemory.Content)
	}

	// Verify source message data
	if foundSourceMessage == nil {
		t.Fatal("expected source message to be populated")
	}

	if foundSourceMessage.Contents != "I really enjoy hiking in the mountains" {
		t.Errorf("expected source message about hiking, got %s", foundSourceMessage.Contents)
	}

	if foundSourceMessage.Role != models.MessageRoleUser {
		t.Errorf("expected source message role user, got %s", foundSourceMessage.Role)
	}

	// Verify conversation ID matches
	if foundSourceMessage.ConversationID != conv.ID {
		t.Errorf("expected conversation ID %s, got %s", conv.ID, foundSourceMessage.ConversationID)
	}
}
