package protocol

import (
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// =============================================================================
// Envelope Tests
// =============================================================================

func TestNewEnvelope(t *testing.T) {
	body := UserMessage{ID: "msg_123", Content: "Hello"}
	env := NewEnvelope(1, "conv_abc", TypeUserMessage, body)

	if env.StanzaID != 1 {
		t.Errorf("expected StanzaID 1, got %d", env.StanzaID)
	}
	if env.ConversationID != "conv_abc" {
		t.Errorf("expected ConversationID 'conv_abc', got %s", env.ConversationID)
	}
	if env.Type != TypeUserMessage {
		t.Errorf("expected Type TypeUserMessage, got %v", env.Type)
	}
	if env.Body == nil {
		t.Error("expected Body to be non-nil")
	}
	if env.Meta != nil {
		t.Errorf("expected Meta to be nil, got %v", env.Meta)
	}
}

func TestNewEnvelopeWithNegativeStanzaID(t *testing.T) {
	// Server messages use negative StanzaIDs
	body := AssistantMessage{ID: "msg_456", Content: "Hello back"}
	env := NewEnvelope(-1, "conv_xyz", TypeAssistantMessage, body)

	if env.StanzaID != -1 {
		t.Errorf("expected StanzaID -1, got %d", env.StanzaID)
	}
}

func TestWithMeta(t *testing.T) {
	env := NewEnvelope(1, "conv_abc", TypeUserMessage, UserMessage{ID: "msg_123"})
	env.WithMeta("timestamp", int64(1234567890))

	if env.Meta == nil {
		t.Fatal("expected Meta to be initialized")
	}
	if val, ok := env.Meta["timestamp"]; !ok || val != int64(1234567890) {
		t.Errorf("expected Meta['timestamp'] to be 1234567890, got %v", val)
	}
}

func TestWithMetaMultiple(t *testing.T) {
	env := NewEnvelope(1, "conv_abc", TypeUserMessage, UserMessage{ID: "msg_123"})
	env.WithMeta("key1", "value1").WithMeta("key2", "value2")

	if len(env.Meta) != 2 {
		t.Errorf("expected Meta to have 2 entries, got %d", len(env.Meta))
	}
	if env.Meta["key1"] != "value1" {
		t.Errorf("expected Meta['key1'] to be 'value1', got %v", env.Meta["key1"])
	}
	if env.Meta["key2"] != "value2" {
		t.Errorf("expected Meta['key2'] to be 'value2', got %v", env.Meta["key2"])
	}
}

func TestWithTracing(t *testing.T) {
	env := NewEnvelope(1, "conv_abc", TypeUserMessage, UserMessage{ID: "msg_123"})
	env.WithTracing("trace-id-123", "span-id-456")

	if env.Meta == nil {
		t.Fatal("expected Meta to be initialized")
	}
	if env.Meta[MetaKeyTraceID] != "trace-id-123" {
		t.Errorf("expected traceID to be 'trace-id-123', got %v", env.Meta[MetaKeyTraceID])
	}
	if env.Meta[MetaKeySpanID] != "span-id-456" {
		t.Errorf("expected spanID to be 'span-id-456', got %v", env.Meta[MetaKeySpanID])
	}
}

func TestWithTracingChaining(t *testing.T) {
	env := NewEnvelope(1, "conv_abc", TypeUserMessage, UserMessage{ID: "msg_123"})
	result := env.WithTracing("trace-id", "span-id").WithMeta("custom", "value")

	if result != env {
		t.Error("expected WithTracing to return the same envelope for chaining")
	}
	if len(env.Meta) != 3 {
		t.Errorf("expected Meta to have 3 entries, got %d", len(env.Meta))
	}
}

// =============================================================================
// MessageType Tests
// =============================================================================

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want    string
	}{
		{TypeErrorMessage, "ErrorMessage"},
		{TypeUserMessage, "UserMessage"},
		{TypeAssistantMessage, "AssistantMessage"},
		{TypeAudioChunk, "AudioChunk"},
		{TypeReasoningStep, "ReasoningStep"},
		{TypeToolUseRequest, "ToolUseRequest"},
		{TypeToolUseResult, "ToolUseResult"},
		{TypeAcknowledgement, "Acknowledgement"},
		{TypeTranscription, "Transcription"},
		{TypeControlStop, "ControlStop"},
		{TypeControlVariation, "ControlVariation"},
		{TypeConfiguration, "Configuration"},
		{TypeStartAnswer, "StartAnswer"},
		{TypeMemoryTrace, "MemoryTrace"},
		{TypeCommentary, "Commentary"},
		{TypeAssistantSentence, "AssistantSentence"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.msgType.String(); got != tt.want {
				t.Errorf("MessageType(%d).String() = %q, want %q", tt.msgType, got, tt.want)
			}
		})
	}
}

func TestMessageTypeStringUnknown(t *testing.T) {
	unknownType := MessageType(999)
	if got := unknownType.String(); got != "Unknown" {
		t.Errorf("Unknown MessageType.String() = %q, want %q", got, "Unknown")
	}
}

// =============================================================================
// Message Struct Tests
// =============================================================================

func TestErrorMessageFields(t *testing.T) {
	msg := ErrorMessage{
		ID:             "err_123",
		ConversationID: "conv_abc",
		Code:           ErrCodeInternalError,
		Message:        "Internal error",
		Severity:       SeverityError,
		Recoverable:    true,
		OriginatingID:  "msg_456",
	}

	if msg.ID != "err_123" {
		t.Errorf("expected ID 'err_123', got %s", msg.ID)
	}
	if msg.Code != ErrCodeInternalError {
		t.Errorf("expected Code %d, got %d", ErrCodeInternalError, msg.Code)
	}
	if msg.Severity != SeverityError {
		t.Errorf("expected Severity %d, got %d", SeverityError, msg.Severity)
	}
	if !msg.Recoverable {
		t.Error("expected Recoverable to be true")
	}
}

func TestUserMessageFields(t *testing.T) {
	msg := UserMessage{
		ID:             "msg_123",
		PreviousID:     "msg_122",
		ConversationID: "conv_abc",
		Content:        "Hello, world!",
		Timestamp:      1234567890,
	}

	if msg.ID != "msg_123" {
		t.Errorf("expected ID 'msg_123', got %s", msg.ID)
	}
	if msg.PreviousID != "msg_122" {
		t.Errorf("expected PreviousID 'msg_122', got %s", msg.PreviousID)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected Content 'Hello, world!', got %s", msg.Content)
	}
	if msg.Timestamp != 1234567890 {
		t.Errorf("expected Timestamp 1234567890, got %d", msg.Timestamp)
	}
}

func TestAssistantMessageFields(t *testing.T) {
	msg := AssistantMessage{
		ID:             "msg_456",
		PreviousID:     "msg_123",
		ConversationID: "conv_abc",
		Content:        "Hello back!",
		Timestamp:      1234567890,
	}

	if msg.ID != "msg_456" {
		t.Errorf("expected ID 'msg_456', got %s", msg.ID)
	}
	if msg.Content != "Hello back!" {
		t.Errorf("expected Content 'Hello back!', got %s", msg.Content)
	}
}

func TestAudioChunkFields(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	msg := AudioChunk{
		ConversationID: "conv_abc",
		Format:         "audio/opus",
		Sequence:       1,
		DurationMs:     100,
		TrackSID:       "track_123",
		Data:           data,
	}

	if msg.Format != "audio/opus" {
		t.Errorf("expected Format 'audio/opus', got %s", msg.Format)
	}
	if msg.Sequence != 1 {
		t.Errorf("expected Sequence 1, got %d", msg.Sequence)
	}
	if msg.DurationMs != 100 {
		t.Errorf("expected DurationMs 100, got %d", msg.DurationMs)
	}
	if len(msg.Data) != 3 {
		t.Errorf("expected Data length 3, got %d", len(msg.Data))
	}
}

func TestReasoningStepFields(t *testing.T) {
	msg := ReasoningStep{
		ID:             "step_123",
		MessageID:      "msg_456",
		ConversationID: "conv_abc",
		Sequence:       1,
		Content:        "Thinking about the problem...",
	}

	if msg.ID != "step_123" {
		t.Errorf("expected ID 'step_123', got %s", msg.ID)
	}
	if msg.MessageID != "msg_456" {
		t.Errorf("expected MessageID 'msg_456', got %s", msg.MessageID)
	}
	if msg.Sequence != 1 {
		t.Errorf("expected Sequence 1, got %d", msg.Sequence)
	}
}

func TestToolUseRequestFields(t *testing.T) {
	params := map[string]interface{}{
		"query": "test query",
		"limit": 10,
	}
	msg := ToolUseRequest{
		ID:             "tool_req_123",
		MessageID:      "msg_456",
		ConversationID: "conv_abc",
		ToolName:       "search",
		Parameters:     params,
		Execution:      ToolExecutionServer,
		TimeoutMs:      DefaultToolTimeout,
	}

	if msg.ToolName != "search" {
		t.Errorf("expected ToolName 'search', got %s", msg.ToolName)
	}
	if msg.Execution != ToolExecutionServer {
		t.Errorf("expected Execution 'server', got %s", msg.Execution)
	}
	if msg.TimeoutMs != DefaultToolTimeout {
		t.Errorf("expected TimeoutMs %d, got %d", DefaultToolTimeout, msg.TimeoutMs)
	}
	if len(msg.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(msg.Parameters))
	}
}

func TestToolUseResultFields(t *testing.T) {
	result := map[string]interface{}{
		"data": "result data",
	}
	msg := ToolUseResult{
		ID:             "tool_res_123",
		RequestID:      "tool_req_123",
		ConversationID: "conv_abc",
		Success:        true,
		Result:         result,
	}

	if !msg.Success {
		t.Error("expected Success to be true")
	}
	if msg.RequestID != "tool_req_123" {
		t.Errorf("expected RequestID 'tool_req_123', got %s", msg.RequestID)
	}
	if msg.Result == nil {
		t.Error("expected Result to be non-nil")
	}
}

func TestToolUseResultWithError(t *testing.T) {
	msg := ToolUseResult{
		ID:             "tool_res_456",
		RequestID:      "tool_req_456",
		ConversationID: "conv_abc",
		Success:        false,
		ErrorCode:      "TIMEOUT",
		ErrorMessage:   "Tool execution timed out",
	}

	if msg.Success {
		t.Error("expected Success to be false")
	}
	if msg.ErrorCode != "TIMEOUT" {
		t.Errorf("expected ErrorCode 'TIMEOUT', got %s", msg.ErrorCode)
	}
	if msg.ErrorMessage != "Tool execution timed out" {
		t.Errorf("expected ErrorMessage 'Tool execution timed out', got %s", msg.ErrorMessage)
	}
}

func TestAcknowledgementFields(t *testing.T) {
	msg := Acknowledgement{
		ConversationID: "conv_abc",
		AckedStanzaID:  5,
		Success:        true,
	}

	if msg.AckedStanzaID != 5 {
		t.Errorf("expected AckedStanzaID 5, got %d", msg.AckedStanzaID)
	}
	if !msg.Success {
		t.Error("expected Success to be true")
	}
}

func TestTranscriptionFields(t *testing.T) {
	msg := Transcription{
		ID:             "trans_123",
		PreviousID:     "trans_122",
		ConversationID: "conv_abc",
		Text:           "Hello world",
		Final:          true,
		Confidence:     0.95,
		Language:       "en-US",
	}

	if msg.Text != "Hello world" {
		t.Errorf("expected Text 'Hello world', got %s", msg.Text)
	}
	if !msg.Final {
		t.Error("expected Final to be true")
	}
	if msg.Confidence != 0.95 {
		t.Errorf("expected Confidence 0.95, got %f", msg.Confidence)
	}
	if msg.Language != "en-US" {
		t.Errorf("expected Language 'en-US', got %s", msg.Language)
	}
}

func TestControlStopFields(t *testing.T) {
	msg := ControlStop{
		ConversationID: "conv_abc",
		TargetID:       "msg_123",
		Reason:         "User requested stop",
		StopType:       StopTypeGeneration,
	}

	if msg.TargetID != "msg_123" {
		t.Errorf("expected TargetID 'msg_123', got %s", msg.TargetID)
	}
	if msg.StopType != StopTypeGeneration {
		t.Errorf("expected StopType 'generation', got %s", msg.StopType)
	}
}

func TestControlVariationFields(t *testing.T) {
	msg := ControlVariation{
		ConversationID: "conv_abc",
		TargetID:       "msg_123",
		Mode:           VariationTypeEdit,
		NewContent:     "Updated content",
	}

	if msg.TargetID != "msg_123" {
		t.Errorf("expected TargetID 'msg_123', got %s", msg.TargetID)
	}
	if msg.Mode != VariationTypeEdit {
		t.Errorf("expected Mode 'edit', got %s", msg.Mode)
	}
	if msg.NewContent != "Updated content" {
		t.Errorf("expected NewContent 'Updated content', got %s", msg.NewContent)
	}
}

func TestConfigurationFields(t *testing.T) {
	features := []string{FeatureStreaming, FeatureToolUse}
	msg := Configuration{
		ConversationID:    "conv_abc",
		LastSequenceSeen:  10,
		ClientVersion:     "1.0.0",
		PreferredLanguage: "en-US",
		Device:            "web",
		Features:          features,
	}

	if msg.ClientVersion != "1.0.0" {
		t.Errorf("expected ClientVersion '1.0.0', got %s", msg.ClientVersion)
	}
	if msg.LastSequenceSeen != 10 {
		t.Errorf("expected LastSequenceSeen 10, got %d", msg.LastSequenceSeen)
	}
	if len(msg.Features) != 2 {
		t.Errorf("expected 2 features, got %d", len(msg.Features))
	}
}

func TestStartAnswerFields(t *testing.T) {
	msg := StartAnswer{
		ID:                   "ans_123",
		PreviousID:           "msg_122",
		ConversationID:       "conv_abc",
		AnswerType:           AnswerTypeText,
		PlannedSentenceCount: 5,
	}

	if msg.ID != "ans_123" {
		t.Errorf("expected ID 'ans_123', got %s", msg.ID)
	}
	if msg.AnswerType != AnswerTypeText {
		t.Errorf("expected AnswerType 'text', got %s", msg.AnswerType)
	}
	if msg.PlannedSentenceCount != 5 {
		t.Errorf("expected PlannedSentenceCount 5, got %d", msg.PlannedSentenceCount)
	}
}

func TestMemoryTraceFields(t *testing.T) {
	msg := MemoryTrace{
		ID:             "mem_123",
		MessageID:      "msg_456",
		ConversationID: "conv_abc",
		MemoryID:       "memory_789",
		Content:        "Retrieved memory content",
		Relevance:      0.85,
	}

	if msg.MemoryID != "memory_789" {
		t.Errorf("expected MemoryID 'memory_789', got %s", msg.MemoryID)
	}
	if msg.Content != "Retrieved memory content" {
		t.Errorf("expected Content 'Retrieved memory content', got %s", msg.Content)
	}
	if msg.Relevance != 0.85 {
		t.Errorf("expected Relevance 0.85, got %f", msg.Relevance)
	}
}

func TestCommentaryFields(t *testing.T) {
	msg := Commentary{
		ID:             "com_123",
		MessageID:      "msg_456",
		ConversationID: "conv_abc",
		Content:        "Internal thought process",
		CommentaryType: "reflection",
	}

	if msg.ID != "com_123" {
		t.Errorf("expected ID 'com_123', got %s", msg.ID)
	}
	if msg.Content != "Internal thought process" {
		t.Errorf("expected Content 'Internal thought process', got %s", msg.Content)
	}
	if msg.CommentaryType != "reflection" {
		t.Errorf("expected CommentaryType 'reflection', got %s", msg.CommentaryType)
	}
}

func TestAssistantSentenceFields(t *testing.T) {
	audio := []byte{0x01, 0x02, 0x03}
	msg := AssistantSentence{
		ID:             "sent_123",
		PreviousID:     "ans_122",
		ConversationID: "conv_abc",
		Sequence:       1,
		Text:           "This is a sentence.",
		IsFinal:        false,
		Audio:          audio,
	}

	if msg.ID != "sent_123" {
		t.Errorf("expected ID 'sent_123', got %s", msg.ID)
	}
	if msg.Sequence != 1 {
		t.Errorf("expected Sequence 1, got %d", msg.Sequence)
	}
	if msg.Text != "This is a sentence." {
		t.Errorf("expected Text 'This is a sentence.', got %s", msg.Text)
	}
	if msg.IsFinal {
		t.Error("expected IsFinal to be false")
	}
	if len(msg.Audio) != 3 {
		t.Errorf("expected Audio length 3, got %d", len(msg.Audio))
	}
}

// =============================================================================
// MessagePack Serialization Tests
// =============================================================================

func TestEnvelopeMarshalUnmarshal(t *testing.T) {
	body := UserMessage{
		ID:             "msg_123",
		ConversationID: "conv_abc",
		Content:        "Test message",
	}
	original := NewEnvelope(1, "conv_abc", TypeUserMessage, body)
	original.WithMeta("timestamp", int64(1234567890))

	// Marshal
	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	// Unmarshal
	var decoded Envelope
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	// Verify fields
	if decoded.StanzaID != original.StanzaID {
		t.Errorf("expected StanzaID %d, got %d", original.StanzaID, decoded.StanzaID)
	}
	if decoded.ConversationID != original.ConversationID {
		t.Errorf("expected ConversationID %s, got %s", original.ConversationID, decoded.ConversationID)
	}
	if decoded.Type != original.Type {
		t.Errorf("expected Type %v, got %v", original.Type, decoded.Type)
	}
	if decoded.Meta == nil {
		t.Error("expected Meta to be non-nil")
	}
	if decoded.Body == nil {
		t.Error("expected Body to be non-nil")
	}
}

func TestUserMessageMarshalUnmarshal(t *testing.T) {
	original := UserMessage{
		ID:             "msg_123",
		PreviousID:     "msg_122",
		ConversationID: "conv_abc",
		Content:        "Hello, world!",
		Timestamp:      1234567890,
	}

	// Marshal
	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal UserMessage: %v", err)
	}

	// Unmarshal
	var decoded UserMessage
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal UserMessage: %v", err)
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, decoded.ID)
	}
	if decoded.PreviousID != original.PreviousID {
		t.Errorf("expected PreviousID %s, got %s", original.PreviousID, decoded.PreviousID)
	}
	if decoded.Content != original.Content {
		t.Errorf("expected Content %s, got %s", original.Content, decoded.Content)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("expected Timestamp %d, got %d", original.Timestamp, decoded.Timestamp)
	}
}

func TestAssistantMessageMarshalUnmarshal(t *testing.T) {
	original := AssistantMessage{
		ID:             "msg_456",
		PreviousID:     "msg_123",
		ConversationID: "conv_abc",
		Content:        "Hello back!",
		Timestamp:      1234567890,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal AssistantMessage: %v", err)
	}

	var decoded AssistantMessage
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal AssistantMessage: %v", err)
	}

	if decoded.ID != original.ID || decoded.Content != original.Content {
		t.Errorf("decoded message doesn't match original")
	}
}

func TestToolUseRequestMarshalUnmarshal(t *testing.T) {
	params := map[string]interface{}{
		"query": "test query",
		"limit": 10,
	}
	original := ToolUseRequest{
		ID:             "tool_req_123",
		MessageID:      "msg_456",
		ConversationID: "conv_abc",
		ToolName:       "search",
		Parameters:     params,
		Execution:      ToolExecutionServer,
		TimeoutMs:      30000,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ToolUseRequest: %v", err)
	}

	var decoded ToolUseRequest
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal ToolUseRequest: %v", err)
	}

	if decoded.ToolName != original.ToolName {
		t.Errorf("expected ToolName %s, got %s", original.ToolName, decoded.ToolName)
	}
	if decoded.Execution != original.Execution {
		t.Errorf("expected Execution %s, got %s", original.Execution, decoded.Execution)
	}
	if len(decoded.Parameters) != len(original.Parameters) {
		t.Errorf("expected %d parameters, got %d", len(original.Parameters), len(decoded.Parameters))
	}
}

func TestToolUseResultMarshalUnmarshal(t *testing.T) {
	result := map[string]interface{}{
		"data": "result data",
	}
	original := ToolUseResult{
		ID:             "tool_res_123",
		RequestID:      "tool_req_123",
		ConversationID: "conv_abc",
		Success:        true,
		Result:         result,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ToolUseResult: %v", err)
	}

	var decoded ToolUseResult
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal ToolUseResult: %v", err)
	}

	if decoded.Success != original.Success {
		t.Errorf("expected Success %v, got %v", original.Success, decoded.Success)
	}
	if decoded.RequestID != original.RequestID {
		t.Errorf("expected RequestID %s, got %s", original.RequestID, decoded.RequestID)
	}
}

func TestAudioChunkMarshalUnmarshal(t *testing.T) {
	audioData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	original := AudioChunk{
		ConversationID: "conv_abc",
		Format:         "audio/opus",
		Sequence:       1,
		DurationMs:     100,
		TrackSID:       "track_123",
		Data:           audioData,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal AudioChunk: %v", err)
	}

	var decoded AudioChunk
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal AudioChunk: %v", err)
	}

	if decoded.Format != original.Format {
		t.Errorf("expected Format %s, got %s", original.Format, decoded.Format)
	}
	if decoded.Sequence != original.Sequence {
		t.Errorf("expected Sequence %d, got %d", original.Sequence, decoded.Sequence)
	}
	if len(decoded.Data) != len(original.Data) {
		t.Errorf("expected Data length %d, got %d", len(original.Data), len(decoded.Data))
	}
}

func TestErrorMessageMarshalUnmarshal(t *testing.T) {
	original := ErrorMessage{
		ID:             "err_123",
		ConversationID: "conv_abc",
		Code:           ErrCodeInternalError,
		Message:        "Internal error occurred",
		Severity:       SeverityError,
		Recoverable:    true,
		OriginatingID:  "msg_456",
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ErrorMessage: %v", err)
	}

	var decoded ErrorMessage
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal ErrorMessage: %v", err)
	}

	if decoded.Code != original.Code {
		t.Errorf("expected Code %d, got %d", original.Code, decoded.Code)
	}
	if decoded.Severity != original.Severity {
		t.Errorf("expected Severity %d, got %d", original.Severity, decoded.Severity)
	}
	if decoded.Recoverable != original.Recoverable {
		t.Errorf("expected Recoverable %v, got %v", original.Recoverable, decoded.Recoverable)
	}
}

func TestConfigurationMarshalUnmarshal(t *testing.T) {
	features := []string{FeatureStreaming, FeatureToolUse, FeatureAudioOutput}
	original := Configuration{
		ConversationID:    "conv_abc",
		LastSequenceSeen:  10,
		ClientVersion:     "1.0.0",
		PreferredLanguage: "en-US",
		Device:            "web",
		Features:          features,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal Configuration: %v", err)
	}

	var decoded Configuration
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal Configuration: %v", err)
	}

	if decoded.ClientVersion != original.ClientVersion {
		t.Errorf("expected ClientVersion %s, got %s", original.ClientVersion, decoded.ClientVersion)
	}
	if len(decoded.Features) != len(original.Features) {
		t.Errorf("expected %d features, got %d", len(original.Features), len(decoded.Features))
	}
}

func TestEnvelopeWithTracingMarshalUnmarshal(t *testing.T) {
	body := UserMessage{
		ID:             "msg_123",
		ConversationID: "conv_abc",
		Content:        "Test message",
	}
	original := NewEnvelope(1, "conv_abc", TypeUserMessage, body)
	original.WithTracing("trace-id-123", "span-id-456")

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal envelope with tracing: %v", err)
	}

	var decoded Envelope
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal envelope with tracing: %v", err)
	}

	if decoded.Meta[MetaKeyTraceID] != "trace-id-123" {
		t.Errorf("expected traceID 'trace-id-123', got %v", decoded.Meta[MetaKeyTraceID])
	}
	if decoded.Meta[MetaKeySpanID] != "span-id-456" {
		t.Errorf("expected spanID 'span-id-456', got %v", decoded.Meta[MetaKeySpanID])
	}
}

// =============================================================================
// Additional Type Tests
// =============================================================================
