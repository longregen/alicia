package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/longregen/alicia/internal/domain"
	"github.com/longregen/alicia/internal/domain/models"
	"github.com/longregen/alicia/internal/ports"
)

// PromptVersionService manages system prompt versions for GEPA optimization
type PromptVersionService struct {
	repo        ports.SystemPromptVersionRepository
	idGenerator ports.IDGenerator
}

// NewPromptVersionService creates a new prompt version service
func NewPromptVersionService(
	repo ports.SystemPromptVersionRepository,
	idGenerator ports.IDGenerator,
) *PromptVersionService {
	return &PromptVersionService{
		repo:        repo,
		idGenerator: idGenerator,
	}
}

// EnsureVersion creates a version if it doesn't exist, or returns existing if hash matches
func (s *PromptVersionService) EnsureVersion(
	ctx context.Context,
	promptType string,
	content string,
	description string,
) (*models.SystemPromptVersion, error) {
	if err := ValidateRequired(promptType, "prompt type"); err != nil {
		return nil, err
	}

	if err := ValidateRequired(content, "prompt content"); err != nil {
		return nil, err
	}

	hash := s.hashPrompt(content)

	// Check if this exact prompt already exists
	existing, err := s.repo.GetByHash(ctx, promptType, hash)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Create new version
	version := models.NewSystemPromptVersion(
		s.idGenerator.GenerateSystemPromptVersionID(),
		hash,
		content,
		promptType,
		description,
	)

	if err := s.repo.Create(ctx, version); err != nil {
		return nil, domain.NewDomainError(err, "failed to create prompt version")
	}

	return version, nil
}

// GetActiveVersion returns the currently active version for a prompt type
func (s *PromptVersionService) GetActiveVersion(ctx context.Context, promptType string) (*models.SystemPromptVersion, error) {
	if err := ValidateRequired(promptType, "prompt type"); err != nil {
		return nil, err
	}

	version, err := s.repo.GetActiveByType(ctx, promptType)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to get active prompt version")
	}

	return version, nil
}

// ActivateVersion sets a version as active (repo handles deactivating others)
func (s *PromptVersionService) ActivateVersion(ctx context.Context, versionID string) error {
	if err := ValidateID(versionID, "version"); err != nil {
		return err
	}

	// Verify the version exists
	_, err := s.repo.GetByID(ctx, versionID)
	if err != nil {
		return domain.NewDomainError(err, "prompt version not found")
	}

	if err := s.repo.SetActive(ctx, versionID); err != nil {
		return domain.NewDomainError(err, "failed to activate prompt version")
	}

	return nil
}

// GetOrCreateForConversation ensures prompt version exists and returns ID for conversation
func (s *PromptVersionService) GetOrCreateForConversation(ctx context.Context, systemPrompt string) (string, error) {
	if err := ValidateRequired(systemPrompt, "system prompt"); err != nil {
		return "", err
	}

	version, err := s.EnsureVersion(ctx, models.PromptTypeMain, systemPrompt, "")
	if err != nil {
		return "", err
	}

	return version.ID, nil
}

// hashPrompt computes SHA256 hash of prompt content and returns hex string
func (s *PromptVersionService) hashPrompt(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ListVersions returns versions for a prompt type
func (s *PromptVersionService) ListVersions(ctx context.Context, promptType string, limit int) ([]*models.SystemPromptVersion, error) {
	if err := ValidateRequired(promptType, "prompt type"); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	versions, err := s.repo.List(ctx, promptType, limit)
	if err != nil {
		return nil, domain.NewDomainError(err, "failed to list prompt versions")
	}

	return versions, nil
}
