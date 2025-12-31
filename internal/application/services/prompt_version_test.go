package services

import (
	"context"
	"testing"

	"github.com/longregen/alicia/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSystemPromptVersionRepository is a mock implementation of SystemPromptVersionRepository
type MockSystemPromptVersionRepository struct {
	mock.Mock
}

func (m *MockSystemPromptVersionRepository) Create(ctx context.Context, version *models.SystemPromptVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockSystemPromptVersionRepository) GetByID(ctx context.Context, id string) (*models.SystemPromptVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemPromptVersion), args.Error(1)
}

func (m *MockSystemPromptVersionRepository) GetActiveByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error) {
	args := m.Called(ctx, promptType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemPromptVersion), args.Error(1)
}

func (m *MockSystemPromptVersionRepository) GetByHash(ctx context.Context, promptType, hash string) (*models.SystemPromptVersion, error) {
	args := m.Called(ctx, promptType, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemPromptVersion), args.Error(1)
}

func (m *MockSystemPromptVersionRepository) SetActive(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSystemPromptVersionRepository) List(ctx context.Context, promptType string, limit int) ([]*models.SystemPromptVersion, error) {
	args := m.Called(ctx, promptType, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SystemPromptVersion), args.Error(1)
}

func (m *MockSystemPromptVersionRepository) GetLatestByType(ctx context.Context, promptType string) (*models.SystemPromptVersion, error) {
	args := m.Called(ctx, promptType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemPromptVersion), args.Error(1)
}

func TestPromptVersionService_EnsureVersion_CreatesNewVersion(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	promptType := models.PromptTypeMain
	content := "You are a helpful assistant"
	description := "Initial version"

	// Mock: hash doesn't exist
	repo.On("GetByHash", ctx, promptType, mock.Anything).Return(nil, assert.AnError)

	// Mock: create succeeds
	repo.On("Create", ctx, mock.MatchedBy(func(v *models.SystemPromptVersion) bool {
		return v.ID == "spv_test" &&
			v.PromptType == promptType &&
			v.PromptContent == content &&
			v.Description == description &&
			!v.Active
	})).Return(nil)

	version, err := service.EnsureVersion(ctx, promptType, content, description)

	assert.NoError(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, "spv_test", version.ID)
	assert.Equal(t, promptType, version.PromptType)
	assert.Equal(t, content, version.PromptContent)
	assert.Equal(t, description, version.Description)
	assert.False(t, version.Active)
	repo.AssertExpectations(t)
}

func TestPromptVersionService_EnsureVersion_ReturnsExisting(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	promptType := models.PromptTypeMain
	content := "You are a helpful assistant"
	description := "Initial version"

	existingVersion := &models.SystemPromptVersion{
		ID:            "spv_existing",
		PromptHash:    "somehash",
		PromptContent: content,
		PromptType:    promptType,
		Active:        true,
	}

	// Mock: hash exists
	repo.On("GetByHash", ctx, promptType, mock.Anything).Return(existingVersion, nil)

	version, err := service.EnsureVersion(ctx, promptType, content, description)

	assert.NoError(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, "spv_existing", version.ID)
	assert.Equal(t, content, version.PromptContent)
	assert.True(t, version.Active)
	repo.AssertExpectations(t)
	// Should not call Create
	repo.AssertNotCalled(t, "Create")
}

func TestPromptVersionService_GetActiveVersion(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	promptType := models.PromptTypeMain

	activeVersion := &models.SystemPromptVersion{
		ID:            "spv_active",
		PromptContent: "Active prompt",
		PromptType:    promptType,
		Active:        true,
	}

	repo.On("GetActiveByType", ctx, promptType).Return(activeVersion, nil)

	version, err := service.GetActiveVersion(ctx, promptType)

	assert.NoError(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, "spv_active", version.ID)
	assert.True(t, version.Active)
	repo.AssertExpectations(t)
}

func TestPromptVersionService_ActivateVersion(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	versionID := "spv_123"

	version := &models.SystemPromptVersion{
		ID:            versionID,
		PromptContent: "Some prompt",
		PromptType:    models.PromptTypeMain,
		Active:        false,
	}

	// Mock: verify version exists
	repo.On("GetByID", ctx, versionID).Return(version, nil)

	// Mock: activate
	repo.On("SetActive", ctx, versionID).Return(nil)

	err := service.ActivateVersion(ctx, versionID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestPromptVersionService_GetOrCreateForConversation(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	systemPrompt := "You are Alicia, a helpful AI assistant"

	// Mock: hash doesn't exist, create new
	repo.On("GetByHash", ctx, models.PromptTypeMain, mock.Anything).Return(nil, assert.AnError)
	repo.On("Create", ctx, mock.Anything).Return(nil)

	versionID, err := service.GetOrCreateForConversation(ctx, systemPrompt)

	assert.NoError(t, err)
	assert.Equal(t, "spv_test", versionID)
	repo.AssertExpectations(t)
}

func TestPromptVersionService_ListVersions(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()
	promptType := models.PromptTypeMain
	limit := 10

	versions := []*models.SystemPromptVersion{
		{ID: "spv_1", PromptType: promptType, VersionNumber: 1},
		{ID: "spv_2", PromptType: promptType, VersionNumber: 2},
	}

	repo.On("List", ctx, promptType, limit).Return(versions, nil)

	result, err := service.ListVersions(ctx, promptType, limit)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "spv_1", result[0].ID)
	assert.Equal(t, "spv_2", result[1].ID)
	repo.AssertExpectations(t)
}

func TestPromptVersionService_HashPrompt(t *testing.T) {
	service := &PromptVersionService{}

	hash1 := service.hashPrompt("test content")
	hash2 := service.hashPrompt("test content")
	hash3 := service.hashPrompt("different content")

	// Same content produces same hash
	assert.Equal(t, hash1, hash2)

	// Different content produces different hash
	assert.NotEqual(t, hash1, hash3)

	// Hash is hex string (SHA256 produces 64 hex chars)
	assert.Len(t, hash1, 64)
}

func TestPromptVersionService_ValidatesInputs(t *testing.T) {
	repo := new(MockSystemPromptVersionRepository)
	idGen := &mockIDGenerator{}
	service := NewPromptVersionService(repo, idGen)

	ctx := context.Background()

	t.Run("EnsureVersion validates empty prompt type", func(t *testing.T) {
		_, err := service.EnsureVersion(ctx, "", "content", "desc")
		assert.Error(t, err)
	})

	t.Run("EnsureVersion validates empty content", func(t *testing.T) {
		_, err := service.EnsureVersion(ctx, models.PromptTypeMain, "", "desc")
		assert.Error(t, err)
	})

	t.Run("GetActiveVersion validates empty prompt type", func(t *testing.T) {
		_, err := service.GetActiveVersion(ctx, "")
		assert.Error(t, err)
	})

	t.Run("ActivateVersion validates empty version ID", func(t *testing.T) {
		err := service.ActivateVersion(ctx, "")
		assert.Error(t, err)
	})

	t.Run("GetOrCreateForConversation validates empty system prompt", func(t *testing.T) {
		_, err := service.GetOrCreateForConversation(ctx, "")
		assert.Error(t, err)
	})

	t.Run("ListVersions validates empty prompt type", func(t *testing.T) {
		_, err := service.ListVersions(ctx, "", 10)
		assert.Error(t, err)
	})

	t.Run("ListVersions defaults limit to 20 if zero", func(t *testing.T) {
		repo.On("List", ctx, models.PromptTypeMain, 20).Return([]*models.SystemPromptVersion{}, nil)
		_, err := service.ListVersions(ctx, models.PromptTypeMain, 0)
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}
