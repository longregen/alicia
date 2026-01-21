// Package protocol defines the Alicia real-time binary protocol for
// streaming conversations between clients and servers over LiveKit.
package protocol

// MessageType represents the type of protocol message
type MessageType uint16

const (
	// TypeErrorMessage (1) - Error notification
	TypeErrorMessage MessageType = 1
	// TypeUserMessage (2) - User's text input
	TypeUserMessage MessageType = 2
	// TypeAssistantMessage (3) - Complete assistant response (non-streaming)
	TypeAssistantMessage MessageType = 3
	// TypeAudioChunk (4) - Raw audio data segment
	TypeAudioChunk MessageType = 4
	// TypeReasoningStep (5) - Internal reasoning trace
	TypeReasoningStep MessageType = 5
	// TypeToolUseRequest (6) - Request to execute a tool
	TypeToolUseRequest MessageType = 6
	// TypeToolUseResult (7) - Tool execution result
	TypeToolUseResult MessageType = 7
	// TypeAcknowledgement (8) - Confirm receipt
	TypeAcknowledgement MessageType = 8
	// TypeTranscription (9) - Speech-to-text output
	TypeTranscription MessageType = 9
	// TypeControlStop (10) - Stop current operation
	TypeControlStop MessageType = 10
	// TypeControlVariation (11) - Edit/vary previous message
	TypeControlVariation MessageType = 11
	// TypeConfiguration (12) - Session configuration
	TypeConfiguration MessageType = 12
	// TypeStartAnswer (13) - Begin streaming response
	TypeStartAnswer MessageType = 13
	// TypeMemoryTrace (14) - Memory retrieval log
	TypeMemoryTrace MessageType = 14
	// TypeCommentary (15) - Assistant's internal commentary
	TypeCommentary MessageType = 15
	// TypeAssistantSentence (16) - Streaming response chunk
	TypeAssistantSentence MessageType = 16
	// TypeSyncRequest (17) - Client sync request
	TypeSyncRequest MessageType = 17
	// TypeSyncResponse (18) - Server sync response
	TypeSyncResponse MessageType = 18
	// TypeFeedback (20) - Vote message sent from client to server
	TypeFeedback MessageType = 20
	// TypeFeedbackConfirmation (21) - Server confirmation with aggregates
	TypeFeedbackConfirmation MessageType = 21
	// TypeUserNote (22) - Note message
	TypeUserNote MessageType = 22
	// TypeNoteConfirmation (23) - Note confirmation
	TypeNoteConfirmation MessageType = 23
	// TypeMemoryAction (24) - Memory CRUD actions
	TypeMemoryAction MessageType = 24
	// TypeMemoryConfirmation (25) - Memory confirmation
	TypeMemoryConfirmation MessageType = 25
	// TypeServerInfo (26) - Server info broadcast
	TypeServerInfo MessageType = 26
	// TypeSessionStats (27) - Session statistics
	TypeSessionStats MessageType = 27
	// TypeConversationUpdate (28) - Conversation metadata update (e.g., title change)
	TypeConversationUpdate MessageType = 28
	// TypeDimensionPreference (29) - User adjusts dimension weights
	TypeDimensionPreference MessageType = 29
	// TypeEliteSelect (30) - User selects a specific elite solution
	TypeEliteSelect MessageType = 30
	// TypeEliteOptions (31) - Server sends available elite solutions
	TypeEliteOptions MessageType = 31
	// TypeOptimizationProgress (32) - Real-time optimization progress updates
	TypeOptimizationProgress MessageType = 32
	// TypeResponseGenerationRequest (33) - Request for agent to generate response
	TypeResponseGenerationRequest MessageType = 33
	// TypeThinkingSummary (34) - Summary of what the agent is about to do
	TypeThinkingSummary MessageType = 34
	// TypeSubscribe (40) - Client subscribes to a conversation
	TypeSubscribe MessageType = 40
	// TypeUnsubscribe (41) - Client unsubscribes from a conversation
	TypeUnsubscribe MessageType = 41
	// TypeSubscribeAck (42) - Server acknowledges subscription
	TypeSubscribeAck MessageType = 42
	// TypeUnsubscribeAck (43) - Server acknowledges unsubscription
	TypeUnsubscribeAck MessageType = 43
	// TypeBranchUpdate (50) - Notifies frontend of new sibling branches
	TypeBranchUpdate MessageType = 50
)

// String returns the string representation of the message type
func (t MessageType) String() string {
	switch t {
	case TypeErrorMessage:
		return "ErrorMessage"
	case TypeUserMessage:
		return "UserMessage"
	case TypeAssistantMessage:
		return "AssistantMessage"
	case TypeAudioChunk:
		return "AudioChunk"
	case TypeReasoningStep:
		return "ReasoningStep"
	case TypeToolUseRequest:
		return "ToolUseRequest"
	case TypeToolUseResult:
		return "ToolUseResult"
	case TypeAcknowledgement:
		return "Acknowledgement"
	case TypeTranscription:
		return "Transcription"
	case TypeControlStop:
		return "ControlStop"
	case TypeControlVariation:
		return "ControlVariation"
	case TypeConfiguration:
		return "Configuration"
	case TypeStartAnswer:
		return "StartAnswer"
	case TypeMemoryTrace:
		return "MemoryTrace"
	case TypeCommentary:
		return "Commentary"
	case TypeAssistantSentence:
		return "AssistantSentence"
	case TypeSyncRequest:
		return "SyncRequest"
	case TypeSyncResponse:
		return "SyncResponse"
	case TypeFeedback:
		return "Feedback"
	case TypeFeedbackConfirmation:
		return "FeedbackConfirmation"
	case TypeUserNote:
		return "UserNote"
	case TypeNoteConfirmation:
		return "NoteConfirmation"
	case TypeMemoryAction:
		return "MemoryAction"
	case TypeMemoryConfirmation:
		return "MemoryConfirmation"
	case TypeServerInfo:
		return "ServerInfo"
	case TypeSessionStats:
		return "SessionStats"
	case TypeConversationUpdate:
		return "ConversationUpdate"
	case TypeDimensionPreference:
		return "DimensionPreference"
	case TypeEliteSelect:
		return "EliteSelect"
	case TypeEliteOptions:
		return "EliteOptions"
	case TypeOptimizationProgress:
		return "OptimizationProgress"
	case TypeResponseGenerationRequest:
		return "ResponseGenerationRequest"
	case TypeThinkingSummary:
		return "ThinkingSummary"
	case TypeSubscribe:
		return "Subscribe"
	case TypeUnsubscribe:
		return "Unsubscribe"
	case TypeSubscribeAck:
		return "SubscribeAck"
	case TypeUnsubscribeAck:
		return "UnsubscribeAck"
	case TypeBranchUpdate:
		return "BranchUpdate"
	default:
		return "Unknown"
	}
}

// Severity levels for error messages
type Severity int32

const (
	SeverityInfo     Severity = 0
	SeverityWarning  Severity = 1
	SeverityError    Severity = 2
	SeverityCritical Severity = 3
)

// StopType indicates what to stop in ControlStop
type StopType string

const (
	StopTypeGeneration StopType = "generation"
	StopTypeSpeech     StopType = "speech"
	StopTypeAll        StopType = "all"
)

// ToolExecution specifies who executes a tool
type ToolExecution string

const (
	ToolExecutionServer ToolExecution = "server"
	ToolExecutionClient ToolExecution = "client"
	ToolExecutionEither ToolExecution = "either"
)

// AnswerType describes the format of an answer
type AnswerType string

const (
	AnswerTypeText      AnswerType = "text"
	AnswerTypeVoice     AnswerType = "voice"
	AnswerTypeTextVoice AnswerType = "text+voice"
)

// VariationType indicates the type of variation request
type VariationType string

const (
	VariationTypeRegenerate VariationType = "regenerate"
	VariationTypeEdit       VariationType = "edit"
	VariationTypeContinue   VariationType = "continue"
)
