package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/longregen/alicia/internal/domain/models"
)

// Integration tests for SystemPromptVersionRepository
// These require a real PostgreSQL instance with the test database

func TestSystemPromptVersionRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	version := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash123abc",
		PromptContent: "You are a helpful assistant.",
		PromptType:    models.PromptTypeMain,
		Description:   "Initial main prompt",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), version)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify we can retrieve it
	retrieved, err := repo.GetByID(context.Background(), version.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != version.ID {
		t.Errorf("expected ID %s, got %s", version.ID, retrieved.ID)
	}

	if retrieved.PromptHash != version.PromptHash {
		t.Errorf("expected prompt hash %s, got %s", version.PromptHash, retrieved.PromptHash)
	}

	if retrieved.PromptContent != version.PromptContent {
		t.Errorf("expected prompt content %s, got %s", version.PromptContent, retrieved.PromptContent)
	}

	if retrieved.PromptType != version.PromptType {
		t.Errorf("expected prompt type %s, got %s", version.PromptType, retrieved.PromptType)
	}

	if retrieved.Active != version.Active {
		t.Errorf("expected active %v, got %v", version.Active, retrieved.Active)
	}
}

func TestSystemPromptVersionRepository_GetByHash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	version := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "unique_hash_456",
		PromptContent: "You are a code assistant.",
		PromptType:    models.PromptTypeToolSelection,
		Description:   "Tool selection prompt v1",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), version)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Retrieve by hash
	retrieved, err := repo.GetByHash(context.Background(), models.PromptTypeToolSelection, "unique_hash_456")
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}

	if retrieved.ID != version.ID {
		t.Errorf("expected ID %s, got %s", version.ID, retrieved.ID)
	}

	if retrieved.PromptHash != "unique_hash_456" {
		t.Errorf("expected hash unique_hash_456, got %s", retrieved.PromptHash)
	}

	// Try with wrong type
	_, err = repo.GetByHash(context.Background(), models.PromptTypeMain, "unique_hash_456")
	if err == nil {
		t.Error("expected error when retrieving with wrong prompt type, got nil")
	}
}

func TestSystemPromptVersionRepository_SetActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	// Create first version and set as active
	version1 := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash1",
		PromptContent: "Version 1",
		PromptType:    models.PromptTypeMain,
		Description:   "First version",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), version1)
	if err != nil {
		t.Fatalf("Create version1 failed: %v", err)
	}

	err = repo.SetActive(context.Background(), version1.ID)
	if err != nil {
		t.Fatalf("SetActive for version1 failed: %v", err)
	}

	// Verify version1 is active
	active, err := repo.GetActiveByType(context.Background(), models.PromptTypeMain)
	if err != nil {
		t.Fatalf("GetActiveByType failed: %v", err)
	}

	if active.ID != version1.ID {
		t.Errorf("expected active version ID %s, got %s", version1.ID, active.ID)
	}

	if !active.Active {
		t.Error("expected active to be true")
	}

	if active.ActivatedAt == nil {
		t.Error("expected activated_at to be set")
	}

	// Create second version and set as active
	version2 := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash2",
		PromptContent: "Version 2",
		PromptType:    models.PromptTypeMain,
		Description:   "Second version",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err = repo.Create(context.Background(), version2)
	if err != nil {
		t.Fatalf("Create version2 failed: %v", err)
	}

	err = repo.SetActive(context.Background(), version2.ID)
	if err != nil {
		t.Fatalf("SetActive for version2 failed: %v", err)
	}

	// Verify version2 is now active
	active, err = repo.GetActiveByType(context.Background(), models.PromptTypeMain)
	if err != nil {
		t.Fatalf("GetActiveByType failed after second activation: %v", err)
	}

	if active.ID != version2.ID {
		t.Errorf("expected active version ID %s, got %s", version2.ID, active.ID)
	}

	// Verify version1 is no longer active
	retrieved1, err := repo.GetByID(context.Background(), version1.ID)
	if err != nil {
		t.Fatalf("GetByID for version1 failed: %v", err)
	}

	if retrieved1.Active {
		t.Error("expected version1 to be deactivated")
	}

	if retrieved1.DeactivatedAt == nil {
		t.Error("expected deactivated_at to be set for version1")
	}
}

func TestSystemPromptVersionRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	// Create multiple versions of different types
	for i := 0; i < 3; i++ {
		version := &models.SystemPromptVersion{
			ID:            idGen.GenerateSystemPromptVersionID(),
			PromptHash:    "main_hash_" + string(rune('a'+i)),
			PromptContent: "Main prompt version " + string(rune('1'+i)),
			PromptType:    models.PromptTypeMain,
			Description:   "Version " + string(rune('1'+i)),
			Active:        i == 0, // First one is active
			CreatedAt:     time.Now(),
		}
		if err := repo.Create(context.Background(), version); err != nil {
			t.Fatalf("Create main version failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		version := &models.SystemPromptVersion{
			ID:            idGen.GenerateSystemPromptVersionID(),
			PromptHash:    "tool_hash_" + string(rune('a'+i)),
			PromptContent: "Tool selection prompt version " + string(rune('1'+i)),
			PromptType:    models.PromptTypeToolSelection,
			Description:   "Version " + string(rune('1'+i)),
			Active:        false,
			CreatedAt:     time.Now(),
		}
		if err := repo.Create(context.Background(), version); err != nil {
			t.Fatalf("Create tool version failed: %v", err)
		}
	}

	// List main prompt versions
	versions, err := repo.List(context.Background(), models.PromptTypeMain, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("expected 3 main prompt versions, got %d", len(versions))
	}

	// Verify ordering (should be DESC by version_number)
	// Note: version_number is auto-generated by DB, so we just check we got results

	// List tool selection versions
	versions, err = repo.List(context.Background(), models.PromptTypeToolSelection, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("expected 2 tool selection versions, got %d", len(versions))
	}

	// Test limit
	versions, err = repo.List(context.Background(), models.PromptTypeMain, 2)
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("expected 2 versions with limit=2, got %d", len(versions))
	}
}

func TestSystemPromptVersionRepository_GetLatestByType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	// Create versions at different times
	version1 := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash_v1",
		PromptContent: "Version 1",
		PromptType:    models.PromptTypeMemorySelection,
		Description:   "First",
		Active:        false,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
	}

	err := repo.Create(context.Background(), version1)
	if err != nil {
		t.Fatalf("Create version1 failed: %v", err)
	}

	// Small delay to ensure different version numbers
	time.Sleep(10 * time.Millisecond)

	version2 := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash_v2",
		PromptContent: "Version 2",
		PromptType:    models.PromptTypeMemorySelection,
		Description:   "Second",
		Active:        false,
		CreatedAt:     time.Now().Add(-1 * time.Hour),
	}

	err = repo.Create(context.Background(), version2)
	if err != nil {
		t.Fatalf("Create version2 failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	version3 := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash_v3",
		PromptContent: "Version 3",
		PromptType:    models.PromptTypeMemorySelection,
		Description:   "Third",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err = repo.Create(context.Background(), version3)
	if err != nil {
		t.Fatalf("Create version3 failed: %v", err)
	}

	// Get latest
	latest, err := repo.GetLatestByType(context.Background(), models.PromptTypeMemorySelection)
	if err != nil {
		t.Fatalf("GetLatestByType failed: %v", err)
	}

	// Latest should be version3 (highest version_number due to SERIAL auto-increment)
	if latest.ID != version3.ID {
		t.Errorf("expected latest version ID %s, got %s", version3.ID, latest.ID)
	}

	if latest.PromptContent != "Version 3" {
		t.Errorf("expected latest content 'Version 3', got %s", latest.PromptContent)
	}
}

func TestSystemPromptVersionRepository_GetActiveByType_NoActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	// Create a version but don't activate it
	version := &models.SystemPromptVersion{
		ID:            idGen.GenerateSystemPromptVersionID(),
		PromptHash:    "hash_inactive",
		PromptContent: "Inactive version",
		PromptType:    models.PromptTypeMemoryExtraction,
		Description:   "Not activated",
		Active:        false,
		CreatedAt:     time.Now(),
	}

	err := repo.Create(context.Background(), version)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Try to get active version (should fail)
	_, err = repo.GetActiveByType(context.Background(), models.PromptTypeMemoryExtraction)
	if err == nil {
		t.Error("expected error when no active version exists, got nil")
	}
}

func TestSystemPromptVersionRepository_MultiplePromptTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	idGen := newTestIDGenerator()
	repo := NewSystemPromptVersionRepository(pool, idGen)

	// Create and activate versions for different prompt types
	promptTypes := []string{
		models.PromptTypeMain,
		models.PromptTypeToolSelection,
		models.PromptTypeMemorySelection,
		models.PromptTypeMemoryExtraction,
	}

	createdVersions := make(map[string]string) // type -> ID

	for _, promptType := range promptTypes {
		version := &models.SystemPromptVersion{
			ID:            idGen.GenerateSystemPromptVersionID(),
			PromptHash:    "hash_" + promptType,
			PromptContent: "Prompt for " + promptType,
			PromptType:    promptType,
			Description:   "Test version",
			Active:        false,
			CreatedAt:     time.Now(),
		}

		err := repo.Create(context.Background(), version)
		if err != nil {
			t.Fatalf("Create for %s failed: %v", promptType, err)
		}

		err = repo.SetActive(context.Background(), version.ID)
		if err != nil {
			t.Fatalf("SetActive for %s failed: %v", promptType, err)
		}

		createdVersions[promptType] = version.ID
	}

	// Verify each type has its own active version
	for _, promptType := range promptTypes {
		active, err := repo.GetActiveByType(context.Background(), promptType)
		if err != nil {
			t.Fatalf("GetActiveByType for %s failed: %v", promptType, err)
		}

		expectedID := createdVersions[promptType]
		if active.ID != expectedID {
			t.Errorf("for type %s, expected active ID %s, got %s", promptType, expectedID, active.ID)
		}

		if !active.Active {
			t.Errorf("for type %s, expected active=true", promptType)
		}
	}
}
