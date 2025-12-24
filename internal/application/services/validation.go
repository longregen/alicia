package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/longregen/alicia/internal/domain"
)

// ValidateID checks that an ID is not empty
func ValidateID(id string, entityType string) error {
	if id == "" {
		return domain.NewDomainError(domain.ErrInvalidID, entityType+" ID cannot be empty")
	}
	return nil
}

// ValidateNotDeleted checks that an entity is not soft-deleted
func ValidateNotDeleted(deletedAt *time.Time, entityType string) error {
	if deletedAt != nil {
		return domain.NewDomainError(domain.ErrDeleted, entityType+" has been deleted")
	}
	return nil
}

// ValidateRequired checks that a required string field is not empty
func ValidateRequired(value string, fieldName string) error {
	if value == "" {
		return domain.NewDomainError(domain.ErrInvalidInput, fieldName+" is required")
	}
	return nil
}

// ValidatePositive checks that a number is positive
func ValidatePositive(value int, fieldName string) error {
	if value <= 0 {
		return domain.NewDomainError(domain.ErrInvalidInput, fieldName+" must be positive")
	}
	return nil
}

// ValidateStringLength checks that a string's length is within the specified range
func ValidateStringLength(value string, fieldName string, minLen, maxLen int) error {
	length := len(value)
	if minLen > 0 && length < minLen {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s must be at least %d characters (got %d)", fieldName, minLen, length))
	}
	if maxLen > 0 && length > maxLen {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s must be at most %d characters (got %d)", fieldName, maxLen, length))
	}
	return nil
}

// ValidateRange checks that a number is within the specified range (inclusive)
func ValidateRange(value int, fieldName string, min, max int) error {
	if value < min {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s must be at least %d (got %d)", fieldName, min, value))
	}
	if value > max {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s must be at most %d (got %d)", fieldName, max, value))
	}
	return nil
}

// ValidateJSONSize checks that a JSON-serializable object is within the size limit (in bytes)
func ValidateJSONSize(value interface{}, fieldName string, maxSizeBytes int) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s contains invalid JSON: %v", fieldName, err))
	}

	size := len(jsonBytes)
	if size > maxSizeBytes {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("%s size exceeds limit: %d bytes (limit: %d bytes)", fieldName, size, maxSizeBytes))
	}
	return nil
}

// ValidateConversationIDFormat checks that a conversation ID follows the expected format (ac_...)
func ValidateConversationIDFormat(conversationID string) error {
	if conversationID == "" {
		return domain.NewDomainError(domain.ErrInvalidInput, "conversation ID cannot be empty")
	}
	if !strings.HasPrefix(conversationID, "ac_") {
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("conversation ID must start with 'ac_' (got: %s)", conversationID))
	}
	if len(conversationID) < 4 { // "ac_" + at least one character
		return domain.NewDomainError(domain.ErrInvalidInput,
			fmt.Sprintf("conversation ID is too short (got: %s)", conversationID))
	}
	return nil
}
