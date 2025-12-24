package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/longregen/alicia/internal/domain/models"
)

// Integration tests require a real PostgreSQL instance
// These tests use the standard pattern of checking behavior without asserting on SQL

func TestConversationRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	conv := models.NewConversation("ac_test123", "test-user", "Test Conversation")
	conv.Preferences = &models.ConversationPreferences{
		TTSVoice:        "en-US-Wavenet-A",
		Language:        "en",
		EnableMemory:    true,
		EnableReasoning: false,
	}

	err := repo.Create(context.Background(), conv)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify we can retrieve it
	retrieved, err := repo.GetByID(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != conv.ID {
		t.Errorf("expected ID %s, got %s", conv.ID, retrieved.ID)
	}
	if retrieved.Title != conv.Title {
		t.Errorf("expected title %s, got %s", conv.Title, retrieved.Title)
	}
	if retrieved.Preferences.TTSVoice != "en-US-Wavenet-A" {
		t.Errorf("preferences not preserved")
	}
}

func TestConversationRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestConversationRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_update123", "test-user", "Original Title")
	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update title
	conv.Title = "Updated Title"
	conv.UpdatedAt = time.Now()

	if err := repo.Update(context.Background(), conv); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("expected updated title, got %s", retrieved.Title)
	}
}

func TestConversationRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_delete123", "test-user", "To Delete")
	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete it
	if err := repo.Delete(context.Background(), conv.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should not be retrievable (soft deleted)
	_, err := repo.GetByID(context.Background(), conv.ID)
	if err != pgx.ErrNoRows {
		t.Errorf("expected conversation to be not found after deletion")
	}
}

func TestConversationRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create multiple conversations
	conv1 := models.NewConversation("ac_list1", "test-user", "Conversation 1")
	conv2 := models.NewConversation("ac_list2", "test-user", "Conversation 2")
	conv3 := models.NewConversation("ac_list3", "test-user", "Conversation 3")

	for _, c := range []*models.Conversation{conv1, conv2, conv3} {
		if err := repo.Create(context.Background(), c); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List all
	conversations, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(conversations) < 3 {
		t.Errorf("expected at least 3 conversations, got %d", len(conversations))
	}
}

func TestConversationRepository_ListActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversations
	active := models.NewConversation("ac_active1", "test-user", "Active")
	archived := models.NewConversation("ac_archived1", "test-user", "Archived")
	archived.Status = models.ConversationStatusArchived

	if err := repo.Create(context.Background(), active); err != nil {
		t.Fatalf("Create active failed: %v", err)
	}
	if err := repo.Create(context.Background(), archived); err != nil {
		t.Fatalf("Create archived failed: %v", err)
	}

	// List active only
	activeConvs, err := repo.ListActive(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}

	// Should contain active but not archived
	foundActive := false
	foundArchived := false
	for _, c := range activeConvs {
		if c.ID == active.ID {
			foundActive = true
		}
		if c.ID == archived.ID {
			foundArchived = true
		}
	}

	if !foundActive {
		t.Error("active conversation not found in list")
	}
	if foundArchived {
		t.Error("archived conversation should not be in active list")
	}
}

func TestConversationRepository_GetByLiveKitRoom(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation with LiveKit room
	conv := models.NewConversation("ac_lk123", "test-user", "LiveKit Test")
	conv.LiveKitRoomName = "room_test123"

	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve by room name
	retrieved, err := repo.GetByLiveKitRoom(context.Background(), "room_test123")
	if err != nil {
		t.Fatalf("GetByLiveKitRoom failed: %v", err)
	}

	if retrieved.ID != conv.ID {
		t.Errorf("expected ID %s, got %s", conv.ID, retrieved.ID)
	}
	if retrieved.LiveKitRoomName != "room_test123" {
		t.Errorf("expected room name room_test123, got %s", retrieved.LiveKitRoomName)
	}
}

func TestConversationRepository_UpdateStanzaIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_stanza123", "test-user", "Stanza Test")
	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update stanza IDs
	err := repo.UpdateStanzaIDs(context.Background(), conv.ID, 100, -50)
	if err != nil {
		t.Fatalf("UpdateStanzaIDs failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.LastClientStanzaID != 100 {
		t.Errorf("expected client stanza ID 100, got %d", retrieved.LastClientStanzaID)
	}
	if retrieved.LastServerStanzaID != -50 {
		t.Errorf("expected server stanza ID -50, got %d", retrieved.LastServerStanzaID)
	}
}

func TestConversationRepository_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create 5 conversations
	for i := 0; i < 5; i++ {
		conv := models.NewConversation(
			fmt.Sprintf("ac_page%d", i),
			"test-user",
			fmt.Sprintf("Page Test %d", i),
		)
		if err := repo.Create(context.Background(), conv); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test pagination
	page1, err := repo.List(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}

	page2, err := repo.List(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}

	// Should get different results
	if len(page1) != 2 {
		t.Errorf("expected 2 results in page 1, got %d", len(page1))
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 results in page 2, got %d", len(page2))
	}

	// Pages should not overlap
	if page1[0].ID == page2[0].ID {
		t.Error("pagination returned overlapping results")
	}
}

// setupTestDB creates a test database pool
// This requires a PostgreSQL instance to be running
//
// The function respects the following environment variables:
//   - TEST_DATABASE_URL: Complete database URL (takes precedence)
//   - PGHOST: Database host or Unix socket directory (from nix shell)
//   - PGPORT: Database port (default: 5432, nix shell uses 5555)
//   - PGUSER: Database user (uses system USER env var if PGUSER is unset or "postgres")
//   - PGDATABASE: Database name (default: alicia_test, nix shell uses alicia)
//
// Note: When initdb is run with --auth=trust (as in nix shell), it creates a role
// matching the system username, not "postgres". This function automatically uses
// the system USER when PGUSER is "postgres" to handle this case.
//
// To run tests with the nix shell database:
//
//	nix develop --command go test ./...
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Use environment variable for test database URL
	dbURL := getTestDatabaseURL()
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean up test data before test starts
	cleanupTestData(t, pool)

	// Clean up test data and close pool after test completes
	// Note: t.Cleanup runs in LIFO order, so this cleanup runs before pool.Close()
	t.Cleanup(func() {
		cleanupTestData(t, pool)
		pool.Close()
	})

	return pool
}

func getTestDatabaseURL() string {
	// Try environment variable first
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}

	// Build URL from environment variables (compatible with nix shell)
	pgHost := os.Getenv("PGHOST")
	pgPort := os.Getenv("PGPORT")
	pgUser := os.Getenv("PGUSER")
	pgDatabase := os.Getenv("PGDATABASE")

	// Default values if not set
	if pgHost == "" {
		pgHost = "localhost"
	}
	if pgPort == "" {
		pgPort = "5432"
	}
	if pgUser == "" {
		pgUser = "postgres"
	}
	if pgDatabase == "" {
		pgDatabase = "alicia_test"
	}

	// If PGHOST is a directory path (Unix socket), use host parameter
	if len(pgHost) > 0 && pgHost[0] == '/' {
		// Unix socket connection
		return fmt.Sprintf("postgres://%s@:%s/%s?host=%s&sslmode=disable",
			pgUser, pgPort, pgDatabase, pgHost)
	}

	// TCP connection
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
		pgUser, pgHost, pgPort, pgDatabase)
}

func TestConversationRepository_GetByIDAndUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation for user1
	conv := models.NewConversation("ac_user_test1", "user1", "User Test")
	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should retrieve with correct user ID
	retrieved, err := repo.GetByIDAndUserID(context.Background(), conv.ID, "user1")
	if err != nil {
		t.Fatalf("GetByIDAndUserID failed: %v", err)
	}
	if retrieved.ID != conv.ID {
		t.Errorf("expected ID %s, got %s", conv.ID, retrieved.ID)
	}

	// Should not retrieve with wrong user ID
	_, err = repo.GetByIDAndUserID(context.Background(), conv.ID, "user2")
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows for wrong user, got %v", err)
	}
}

func TestConversationRepository_ListByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversations for different users
	conv1 := models.NewConversation("ac_user_list1", "alice", "Alice Conv 1")
	conv2 := models.NewConversation("ac_user_list2", "alice", "Alice Conv 2")
	conv3 := models.NewConversation("ac_user_list3", "bob", "Bob Conv 1")

	for _, c := range []*models.Conversation{conv1, conv2, conv3} {
		if err := repo.Create(context.Background(), c); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List conversations for alice
	aliceConvs, err := repo.ListByUserID(context.Background(), "alice", 10, 0)
	if err != nil {
		t.Fatalf("ListByUserID failed: %v", err)
	}

	// Should only contain alice's conversations
	aliceCount := 0
	for _, c := range aliceConvs {
		if c.UserID == "alice" && (c.ID == conv1.ID || c.ID == conv2.ID) {
			aliceCount++
		}
		if c.ID == conv3.ID {
			t.Error("bob's conversation should not be in alice's list")
		}
	}

	if aliceCount != 2 {
		t.Errorf("expected 2 conversations for alice, got %d", aliceCount)
	}
}

func TestConversationRepository_ListActiveByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create active and archived conversations for same user
	active := models.NewConversation("ac_user_active1", "charlie", "Active")
	archived := models.NewConversation("ac_user_archived1", "charlie", "Archived")
	archived.Status = models.ConversationStatusArchived

	if err := repo.Create(context.Background(), active); err != nil {
		t.Fatalf("Create active failed: %v", err)
	}
	if err := repo.Create(context.Background(), archived); err != nil {
		t.Fatalf("Create archived failed: %v", err)
	}

	// List active conversations for charlie
	activeConvs, err := repo.ListActiveByUserID(context.Background(), "charlie", 10, 0)
	if err != nil {
		t.Fatalf("ListActiveByUserID failed: %v", err)
	}

	foundActive := false
	foundArchived := false
	for _, c := range activeConvs {
		if c.ID == active.ID {
			foundActive = true
		}
		if c.ID == archived.ID {
			foundArchived = true
		}
	}

	if !foundActive {
		t.Error("active conversation not found")
	}
	if foundArchived {
		t.Error("archived conversation should not be in active list")
	}
}

func TestConversationRepository_DeleteByIDAndUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// Create conversation
	conv := models.NewConversation("ac_user_delete1", "dave", "To Delete")
	if err := repo.Create(context.Background(), conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should not delete with wrong user ID
	err := repo.DeleteByIDAndUserID(context.Background(), conv.ID, "eve")
	if err != nil {
		t.Fatalf("DeleteByIDAndUserID with wrong user failed: %v", err)
	}

	// Verify still exists
	retrieved, err := repo.GetByID(context.Background(), conv.ID)
	if err != nil {
		t.Fatal("conversation should still exist after wrong user delete")
	}
	if retrieved.ID != conv.ID {
		t.Error("conversation should still exist")
	}

	// Should delete with correct user ID
	err = repo.DeleteByIDAndUserID(context.Background(), conv.ID, "dave")
	if err != nil {
		t.Fatalf("DeleteByIDAndUserID failed: %v", err)
	}

	// Verify deleted
	_, err = repo.GetByID(context.Background(), conv.ID)
	if err != pgx.ErrNoRows {
		t.Error("conversation should be deleted")
	}
}

func TestConversationRepository_GetByLiveKitRoom_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	_, err := repo.GetByLiveKitRoom(context.Background(), "nonexistent_room")
	if err != pgx.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestConversationRepository_EmptyList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)

	repo := NewConversationRepository(pool)

	// List with offset beyond all results
	convs, err := repo.List(context.Background(), 10, 10000)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if convs == nil {
		t.Error("expected empty slice, got nil")
	}
}

func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Delete test conversations (those with IDs starting with ac_test, ac_list, etc.)
	_, err := pool.Exec(ctx, `
		DELETE FROM alicia_conversations
		WHERE id LIKE 'ac_test%'
		   OR id LIKE 'ac_list%'
		   OR id LIKE 'ac_active%'
		   OR id LIKE 'ac_archived%'
		   OR id LIKE 'ac_lk%'
		   OR id LIKE 'ac_stanza%'
		   OR id LIKE 'ac_page%'
		   OR id LIKE 'ac_update%'
		   OR id LIKE 'ac_delete%'
		   OR id LIKE 'ac_msg_%'
		   OR id LIKE 'ac_user_%'
		   OR id LIKE 'ac_tx_%'
	`)
	if err != nil {
		t.Logf("Warning: failed to clean up test data: %v", err)
	}
}
