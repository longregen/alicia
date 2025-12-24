package models

import (
	"testing"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name        string
		from        ConversationStatus
		to          ConversationStatus
		shouldError bool
	}{
		// Valid transitions from active
		{
			name:        "active to archived",
			from:        ConversationStatusActive,
			to:          ConversationStatusArchived,
			shouldError: false,
		},
		{
			name:        "active to deleted",
			from:        ConversationStatusActive,
			to:          ConversationStatusDeleted,
			shouldError: false,
		},

		// Valid transitions from archived
		{
			name:        "archived to active",
			from:        ConversationStatusArchived,
			to:          ConversationStatusActive,
			shouldError: false,
		},
		{
			name:        "archived to deleted",
			from:        ConversationStatusArchived,
			to:          ConversationStatusDeleted,
			shouldError: false,
		},

		// Invalid transitions from deleted (terminal state)
		{
			name:        "deleted to active",
			from:        ConversationStatusDeleted,
			to:          ConversationStatusActive,
			shouldError: true,
		},
		{
			name:        "deleted to archived",
			from:        ConversationStatusDeleted,
			to:          ConversationStatusArchived,
			shouldError: true,
		},

		// No-op transitions (same state)
		{
			name:        "active to active",
			from:        ConversationStatusActive,
			to:          ConversationStatusActive,
			shouldError: false,
		},
		{
			name:        "archived to archived",
			from:        ConversationStatusArchived,
			to:          ConversationStatusArchived,
			shouldError: false,
		},
		{
			name:        "deleted to deleted",
			from:        ConversationStatusDeleted,
			to:          ConversationStatusDeleted,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if tt.shouldError && err == nil {
				t.Errorf("expected error for transition from %s to %s, got nil", tt.from, tt.to)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error for transition from %s to %s, got: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from     ConversationStatus
		to       ConversationStatus
		expected bool
	}{
		{ConversationStatusActive, ConversationStatusArchived, true},
		{ConversationStatusActive, ConversationStatusDeleted, true},
		{ConversationStatusArchived, ConversationStatusActive, true},
		{ConversationStatusArchived, ConversationStatusDeleted, true},
		{ConversationStatusDeleted, ConversationStatusActive, false},
		{ConversationStatusDeleted, ConversationStatusArchived, false},
	}

	for _, tt := range tests {
		result := IsValidTransition(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("IsValidTransition(%s, %s) = %v, want %v",
				tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestGetValidTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     ConversationStatus
		expected int // expected number of valid transitions
	}{
		{
			name:     "from active",
			from:     ConversationStatusActive,
			expected: 2, // can go to archived or deleted
		},
		{
			name:     "from archived",
			from:     ConversationStatusArchived,
			expected: 2, // can go to active or deleted
		},
		{
			name:     "from deleted",
			from:     ConversationStatusDeleted,
			expected: 0, // terminal state
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validStates := GetValidTransitions(tt.from)
			if len(validStates) != tt.expected {
				t.Errorf("GetValidTransitions(%s) returned %d states, want %d",
					tt.from, len(validStates), tt.expected)
			}
		})
	}
}

func TestCanArchive(t *testing.T) {
	tests := []struct {
		status   ConversationStatus
		expected bool
	}{
		{ConversationStatusActive, true},
		{ConversationStatusArchived, true}, // no-op transition is valid
		{ConversationStatusDeleted, false},
	}

	for _, tt := range tests {
		result := CanArchive(tt.status)
		if result != tt.expected {
			t.Errorf("CanArchive(%s) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestCanUnarchive(t *testing.T) {
	tests := []struct {
		status   ConversationStatus
		expected bool
	}{
		{ConversationStatusActive, true}, // no-op transition is valid
		{ConversationStatusArchived, true},
		{ConversationStatusDeleted, false},
	}

	for _, tt := range tests {
		result := CanUnarchive(tt.status)
		if result != tt.expected {
			t.Errorf("CanUnarchive(%s) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestCanDelete(t *testing.T) {
	tests := []struct {
		status   ConversationStatus
		expected bool
	}{
		{ConversationStatusActive, true},
		{ConversationStatusArchived, true},
		{ConversationStatusDeleted, true}, // no-op transition is valid
	}

	for _, tt := range tests {
		result := CanDelete(tt.status)
		if result != tt.expected {
			t.Errorf("CanDelete(%s) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestInvalidTransitionError(t *testing.T) {
	err := NewInvalidTransitionError(ConversationStatusDeleted, ConversationStatusActive)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check that error message contains useful information
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message is empty")
	}

	// Check that the error has the correct fields
	if err.From != ConversationStatusDeleted {
		t.Errorf("expected From = %s, got %s", ConversationStatusDeleted, err.From)
	}
	if err.To != ConversationStatusActive {
		t.Errorf("expected To = %s, got %s", ConversationStatusActive, err.To)
	}
}

func TestNewStateEvent(t *testing.T) {
	event := NewStateEvent(
		"test_conv_123",
		ConversationStatusActive,
		ConversationStatusArchived,
		"user archived conversation",
	)

	if event.ConversationID != "test_conv_123" {
		t.Errorf("expected ConversationID = test_conv_123, got %s", event.ConversationID)
	}
	if event.FromStatus != ConversationStatusActive {
		t.Errorf("expected FromStatus = active, got %s", event.FromStatus)
	}
	if event.ToStatus != ConversationStatusArchived {
		t.Errorf("expected ToStatus = archived, got %s", event.ToStatus)
	}
	if event.Reason != "user archived conversation" {
		t.Errorf("expected Reason = 'user archived conversation', got %s", event.Reason)
	}
}
