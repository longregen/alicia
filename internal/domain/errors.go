package domain

import "errors"

// Common domain errors
var (
	// Conversation errors
	ErrConversationNotFound        = errors.New("conversation not found")
	ErrConversationArchived        = errors.New("conversation is archived")
	ErrConversationDeleted         = errors.New("conversation is deleted")
	ErrInvalidStatusTransition     = errors.New("invalid conversation status transition")
	ErrCannotUnarchiveDeleted      = errors.New("cannot unarchive a deleted conversation")
	ErrCannotModifyDeletedConv     = errors.New("cannot modify a deleted conversation")
	ErrConversationAlreadyArchived = errors.New("conversation is already archived")
	ErrConversationAlreadyActive   = errors.New("conversation is already active")

	// Message errors
	ErrMessageNotFound = errors.New("message not found")
	ErrInvalidRole     = errors.New("invalid message role")

	// Audio errors
	ErrAudioNotFound          = errors.New("audio not found")
	ErrAudioFormatUnsupported = errors.New("audio format not supported")
	ErrTranscriptionFailed    = errors.New("transcription failed")

	// Memory errors
	ErrMemoryNotFound     = errors.New("memory not found")
	ErrEmbeddingsFailed   = errors.New("failed to generate embeddings")
	ErrMemorySearchFailed = errors.New("memory search failed")

	// Tool errors
	ErrToolNotFound        = errors.New("tool not found")
	ErrToolDisabled        = errors.New("tool is disabled")
	ErrToolExecutionFailed = errors.New("tool execution failed")
	ErrToolTimeout         = errors.New("tool execution timed out")
	ErrInvalidToolArgs     = errors.New("invalid tool arguments")

	// LLM errors
	ErrLLMUnavailable    = errors.New("LLM service unavailable")
	ErrLLMRequestFailed  = errors.New("LLM request failed")
	ErrLLMContextTooLong = errors.New("context too long for LLM")

	// LiveKit errors
	ErrLiveKitUnavailable = errors.New("LiveKit service unavailable")
	ErrRoomNotFound       = errors.New("LiveKit room not found")
	ErrTrackNotFound      = errors.New("LiveKit track not found")

	// Speech errors
	ErrASRUnavailable = errors.New("ASR service unavailable")
	ErrTTSUnavailable = errors.New("TTS service unavailable")
	ErrASRFailed      = errors.New("ASR processing failed")
	ErrTTSFailed      = errors.New("TTS processing failed")

	// Validation errors
	ErrInvalidID    = errors.New("invalid ID format")
	ErrEmptyContent = errors.New("content cannot be empty")
	ErrInvalidState = errors.New("invalid state transition")
	ErrInvalidInput = errors.New("invalid input")
	ErrDeleted      = errors.New("entity has been deleted")
	ErrNotFound     = errors.New("resource not found")
)

// DomainError wraps a domain error with additional context
type DomainError struct {
	Err     error
	Message string
	Code    string
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(err error, message string) *DomainError {
	return &DomainError{
		Err:     err,
		Message: message,
	}
}

func NewDomainErrorWithCode(err error, message, code string) *DomainError {
	return &DomainError{
		Err:     err,
		Message: message,
		Code:    code,
	}
}
