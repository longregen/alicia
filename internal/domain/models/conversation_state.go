package models

import (
	"fmt"
)

// ConversationTransition represents a state transition
type ConversationTransition struct {
	From ConversationStatus
	To   ConversationStatus
}

// validTransitions defines the allowed state transitions for conversations
var validTransitions = map[ConversationTransition]bool{
	// From active
	{ConversationStatusActive, ConversationStatusArchived}: true,
	{ConversationStatusActive, ConversationStatusDeleted}:  true,

	// From archived
	{ConversationStatusArchived, ConversationStatusActive}:  true,
	{ConversationStatusArchived, ConversationStatusDeleted}: true,

	// Deleted is a terminal state - no transitions allowed
}

// ValidateTransition checks if a state transition is valid and returns an error if not
func ValidateTransition(from, to ConversationStatus) error {
	// No-op transition is always valid
	if from == to {
		return nil
	}

	// Check if the transition is in the valid transitions map
	transition := ConversationTransition{From: from, To: to}
	if !validTransitions[transition] {
		return NewInvalidTransitionError(from, to)
	}

	return nil
}

// IsValidTransition checks if a transition between two states is valid
func IsValidTransition(from, to ConversationStatus) bool {
	return ValidateTransition(from, to) == nil
}

// GetValidTransitions returns all valid transitions from a given state
func GetValidTransitions(from ConversationStatus) []ConversationStatus {
	validStates := make([]ConversationStatus, 0)

	for transition := range validTransitions {
		if transition.From == from {
			validStates = append(validStates, transition.To)
		}
	}

	return validStates
}

// InvalidTransitionError represents an error for invalid state transitions
type InvalidTransitionError struct {
	From    ConversationStatus
	To      ConversationStatus
	Message string
}

func (e *InvalidTransitionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("invalid conversation state transition from '%s' to '%s'", e.From, e.To)
}

// NewInvalidTransitionError creates a new InvalidTransitionError with a descriptive message
func NewInvalidTransitionError(from, to ConversationStatus) *InvalidTransitionError {
	message := generateTransitionErrorMessage(from, to)
	return &InvalidTransitionError{
		From:    from,
		To:      to,
		Message: message,
	}
}

// generateTransitionErrorMessage creates a helpful error message for invalid transitions
func generateTransitionErrorMessage(from, to ConversationStatus) string {
	switch from {
	case ConversationStatusDeleted:
		return "cannot transition from deleted state: conversation is permanently deleted"
	case ConversationStatusActive:
		if to == ConversationStatusActive {
			return "conversation is already active"
		}
		return fmt.Sprintf("cannot transition from active to '%s': use Archive() or Delete() methods", to)
	case ConversationStatusArchived:
		if to == ConversationStatusArchived {
			return "conversation is already archived"
		}
		return fmt.Sprintf("cannot transition from archived to '%s': use Unarchive() or Delete() methods", to)
	default:
		validStates := GetValidTransitions(from)
		if len(validStates) > 0 {
			return fmt.Sprintf("invalid transition from '%s' to '%s': valid transitions are %v", from, to, validStates)
		}
		return fmt.Sprintf("invalid transition from '%s' to '%s': no valid transitions from this state", from, to)
	}
}

// CanArchive checks if a conversation can be archived from its current state
func CanArchive(status ConversationStatus) bool {
	return IsValidTransition(status, ConversationStatusArchived)
}

// CanUnarchive checks if a conversation can be unarchived from its current state
func CanUnarchive(status ConversationStatus) bool {
	return IsValidTransition(status, ConversationStatusActive)
}

// CanDelete checks if a conversation can be deleted from its current state
func CanDelete(status ConversationStatus) bool {
	return IsValidTransition(status, ConversationStatusDeleted)
}

// ConversationStateEvent represents a state change event for auditing
type ConversationStateEvent struct {
	ConversationID string
	FromStatus     ConversationStatus
	ToStatus       ConversationStatus
	Reason         string
	Timestamp      int64 // Unix timestamp
}

// NewStateEvent creates a new state change event
func NewStateEvent(conversationID string, from, to ConversationStatus, reason string) *ConversationStateEvent {
	return &ConversationStateEvent{
		ConversationID: conversationID,
		FromStatus:     from,
		ToStatus:       to,
		Reason:         reason,
		Timestamp:      0, // Will be set by the caller with time.Now().Unix()
	}
}
