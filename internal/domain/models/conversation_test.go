package models

import (
	"testing"
	"time"
)

func TestConversation_Archive(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus ConversationStatus
		shouldError   bool
	}{
		{
			name:          "archive active conversation",
			initialStatus: ConversationStatusActive,
			shouldError:   false,
		},
		{
			name:          "archive already archived conversation",
			initialStatus: ConversationStatusArchived,
			shouldError:   false, // no-op transition
		},
		{
			name:          "archive deleted conversation",
			initialStatus: ConversationStatusDeleted,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.initialStatus,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if tt.initialStatus == ConversationStatusDeleted {
				now := time.Now()
				conv.DeletedAt = &now
			}

			err := conv.Archive()

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.shouldError && conv.Status != ConversationStatusArchived {
				t.Errorf("expected status archived, got %s", conv.Status)
			}
		})
	}
}

func TestConversation_Unarchive(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus ConversationStatus
		shouldError   bool
	}{
		{
			name:          "unarchive archived conversation",
			initialStatus: ConversationStatusArchived,
			shouldError:   false,
		},
		{
			name:          "unarchive active conversation",
			initialStatus: ConversationStatusActive,
			shouldError:   false, // no-op transition
		},
		{
			name:          "unarchive deleted conversation",
			initialStatus: ConversationStatusDeleted,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.initialStatus,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if tt.initialStatus == ConversationStatusDeleted {
				now := time.Now()
				conv.DeletedAt = &now
			}

			err := conv.Unarchive()

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.shouldError && conv.Status != ConversationStatusActive {
				t.Errorf("expected status active, got %s", conv.Status)
			}
		})
	}
}

func TestConversation_MarkAsDeleted(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus ConversationStatus
		shouldError   bool
	}{
		{
			name:          "delete active conversation",
			initialStatus: ConversationStatusActive,
			shouldError:   false,
		},
		{
			name:          "delete archived conversation",
			initialStatus: ConversationStatusArchived,
			shouldError:   false,
		},
		{
			name:          "delete already deleted conversation",
			initialStatus: ConversationStatusDeleted,
			shouldError:   false, // no-op transition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.initialStatus,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if tt.initialStatus == ConversationStatusDeleted {
				now := time.Now()
				conv.DeletedAt = &now
			}

			err := conv.MarkAsDeleted()

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.shouldError {
				if conv.Status != ConversationStatusDeleted {
					t.Errorf("expected status deleted, got %s", conv.Status)
				}
				if conv.DeletedAt == nil {
					t.Error("expected DeletedAt to be set")
				}
			}
		})
	}
}

func TestConversation_ChangeStatus(t *testing.T) {
	tests := []struct {
		name        string
		from        ConversationStatus
		to          ConversationStatus
		shouldError bool
	}{
		{
			name:        "valid transition active to archived",
			from:        ConversationStatusActive,
			to:          ConversationStatusArchived,
			shouldError: false,
		},
		{
			name:        "valid transition archived to active",
			from:        ConversationStatusArchived,
			to:          ConversationStatusActive,
			shouldError: false,
		},
		{
			name:        "invalid transition deleted to active",
			from:        ConversationStatusDeleted,
			to:          ConversationStatusActive,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.from,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			err := conv.ChangeStatus(tt.to)

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.shouldError && conv.Status != tt.to {
				t.Errorf("expected status %s, got %s", tt.to, conv.Status)
			}
		})
	}
}

func TestConversation_ChangeStatus_SetsDeletedAt(t *testing.T) {
	conv := &Conversation{
		ID:        "test_123",
		Title:     "Test",
		Status:    ConversationStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := conv.ChangeStatus(ConversationStatusDeleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conv.DeletedAt == nil {
		t.Error("expected DeletedAt to be set when transitioning to deleted")
	}
	if conv.Status != ConversationStatusDeleted {
		t.Errorf("expected status deleted, got %s", conv.Status)
	}
}

func TestConversation_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		status   ConversationStatus
		target   ConversationStatus
		expected bool
	}{
		{
			name:     "active can transition to archived",
			status:   ConversationStatusActive,
			target:   ConversationStatusArchived,
			expected: true,
		},
		{
			name:     "active can transition to deleted",
			status:   ConversationStatusActive,
			target:   ConversationStatusDeleted,
			expected: true,
		},
		{
			name:     "deleted cannot transition to active",
			status:   ConversationStatusDeleted,
			target:   ConversationStatusActive,
			expected: false,
		},
		{
			name:     "archived can transition to active",
			status:   ConversationStatusArchived,
			target:   ConversationStatusActive,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			result := conv.CanTransitionTo(tt.target)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConversation_IsActive(t *testing.T) {
	tests := []struct {
		name      string
		status    ConversationStatus
		deletedAt *time.Time
		expected  bool
	}{
		{
			name:      "active and not deleted",
			status:    ConversationStatusActive,
			deletedAt: nil,
			expected:  true,
		},
		{
			name:      "archived and not deleted",
			status:    ConversationStatusArchived,
			deletedAt: nil,
			expected:  false,
		},
		{
			name:      "active but deleted",
			status:    ConversationStatusActive,
			deletedAt: func() *time.Time { t := time.Now(); return &t }(),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &Conversation{
				ID:        "test_123",
				Title:     "Test",
				Status:    tt.status,
				DeletedAt: tt.deletedAt,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			result := conv.IsActive()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
