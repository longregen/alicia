package livekit

import (
	"testing"

	"github.com/longregen/alicia/pkg/protocol"
)

// TestCodec_EncodeDecode_UserMessage tests encoding and decoding of UserMessage
func TestCodec_EncodeDecode_UserMessage(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.UserMessage{
		ID:             "msg_123",
		PreviousID:     "msg_122",
		ConversationID: "conv_abc",
		Content:        "Hello, Alicia!",
		Timestamp:      1234567890,
	}

	env := protocol.NewEnvelope(1, "conv_abc", protocol.TypeUserMessage, msg)

	// Encode
	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Encode returned empty data")
	}

	// Decode
	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.StanzaID != 1 {
		t.Errorf("expected StanzaID 1, got %d", decoded.StanzaID)
	}

	if decoded.ConversationID != "conv_abc" {
		t.Errorf("expected conversationId 'conv_abc', got %s", decoded.ConversationID)
	}

	if decoded.Type != protocol.TypeUserMessage {
		t.Errorf("expected type %d, got %d", protocol.TypeUserMessage, decoded.Type)
	}

	// Check body
	userMsg, ok := decoded.Body.(*protocol.UserMessage)
	if !ok {
		t.Fatalf("body is not UserMessage, got %T", decoded.Body)
	}

	if userMsg.Content != "Hello, Alicia!" {
		t.Errorf("expected content 'Hello, Alicia!', got %s", userMsg.Content)
	}

	if userMsg.ID != "msg_123" {
		t.Errorf("expected ID 'msg_123', got %s", userMsg.ID)
	}

	if userMsg.PreviousID != "msg_122" {
		t.Errorf("expected PreviousID 'msg_122', got %s", userMsg.PreviousID)
	}

	if userMsg.Timestamp != 1234567890 {
		t.Errorf("expected Timestamp 1234567890, got %d", userMsg.Timestamp)
	}
}

// TestCodec_EncodeDecode_AssistantMessage tests AssistantMessage with all fields
func TestCodec_EncodeDecode_AssistantMessage(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.AssistantMessage{
		ID:             "msg_456",
		PreviousID:     "msg_123",
		ConversationID: "conv_abc",
		Content:        "I'm here to help!",
		Timestamp:      1234567891,
	}

	env := protocol.NewEnvelope(-1, "conv_abc", protocol.TypeAssistantMessage, msg)

	// Encode
	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	assistantMsg, ok := decoded.Body.(*protocol.AssistantMessage)
	if !ok {
		t.Fatalf("body is not AssistantMessage, got %T", decoded.Body)
	}

	if assistantMsg.Content != "I'm here to help!" {
		t.Errorf("expected content 'I'm here to help!', got %s", assistantMsg.Content)
	}

	if assistantMsg.ID != "msg_456" {
		t.Errorf("expected ID 'msg_456', got %s", assistantMsg.ID)
	}
}

// TestCodec_EncodeDecode_AssistantSentence tests streaming response chunks
func TestCodec_EncodeDecode_AssistantSentence(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.AssistantSentence{
		ID:             "sentence_1",
		PreviousID:     "start_answer_1",
		ConversationID: "conv_abc",
		Sequence:       1,
		Text:           "This is a streaming sentence.",
		IsFinal:        false,
		Audio:          []byte{0x01, 0x02, 0x03},
	}

	env := protocol.NewEnvelope(-2, "conv_abc", protocol.TypeAssistantSentence, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	sentence, ok := decoded.Body.(*protocol.AssistantSentence)
	if !ok {
		t.Fatalf("body is not AssistantSentence, got %T", decoded.Body)
	}

	if sentence.Sequence != 1 {
		t.Errorf("expected Sequence 1, got %d", sentence.Sequence)
	}

	if sentence.IsFinal {
		t.Error("expected IsFinal false, got true")
	}

	if len(sentence.Audio) != 3 {
		t.Errorf("expected Audio length 3, got %d", len(sentence.Audio))
	}
}

// TestCodec_EncodeDecode_ToolUseRequest tests ToolUseRequest with parameters map
func TestCodec_EncodeDecode_ToolUseRequest(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.ToolUseRequest{
		ID:             "tool_req_1",
		MessageID:      "msg_123",
		ConversationID: "conv_abc",
		ToolName:       "search_web",
		Parameters: map[string]interface{}{
			"query":    "golang testing",
			"maxItems": 10,
			"detailed": true,
		},
		Execution: protocol.ToolExecutionServer,
		TimeoutMs: 5000,
	}

	env := protocol.NewEnvelope(-3, "conv_abc", protocol.TypeToolUseRequest, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	toolReq, ok := decoded.Body.(*protocol.ToolUseRequest)
	if !ok {
		t.Fatalf("body is not ToolUseRequest, got %T", decoded.Body)
	}

	if toolReq.ToolName != "search_web" {
		t.Errorf("expected ToolName 'search_web', got %s", toolReq.ToolName)
	}

	if toolReq.Execution != protocol.ToolExecutionServer {
		t.Errorf("expected Execution 'server', got %s", toolReq.Execution)
	}

	if toolReq.Parameters["query"] != "golang testing" {
		t.Errorf("expected query 'golang testing', got %v", toolReq.Parameters["query"])
	}

	if toolReq.TimeoutMs != 5000 {
		t.Errorf("expected TimeoutMs 5000, got %d", toolReq.TimeoutMs)
	}
}

// TestCodec_EncodeDecode_ToolUseResult tests ToolUseResult with success/failure states
func TestCodec_EncodeDecode_ToolUseResult_Success(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.ToolUseResult{
		ID:             "tool_result_1",
		RequestID:      "tool_req_1",
		ConversationID: "conv_abc",
		Success:        true,
		Result: map[string]interface{}{
			"items": []string{"result1", "result2"},
			"count": 2,
		},
	}

	env := protocol.NewEnvelope(2, "conv_abc", protocol.TypeToolUseResult, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	toolResult, ok := decoded.Body.(*protocol.ToolUseResult)
	if !ok {
		t.Fatalf("body is not ToolUseResult, got %T", decoded.Body)
	}

	if !toolResult.Success {
		t.Error("expected Success true, got false")
	}

	if toolResult.RequestID != "tool_req_1" {
		t.Errorf("expected RequestID 'tool_req_1', got %s", toolResult.RequestID)
	}
}

func TestCodec_EncodeDecode_ToolUseResult_Failure(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.ToolUseResult{
		ID:             "tool_result_2",
		RequestID:      "tool_req_2",
		ConversationID: "conv_abc",
		Success:        false,
		ErrorCode:      "TIMEOUT",
		ErrorMessage:   "Tool execution timed out after 5000ms",
	}

	env := protocol.NewEnvelope(3, "conv_abc", protocol.TypeToolUseResult, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	toolResult, ok := decoded.Body.(*protocol.ToolUseResult)
	if !ok {
		t.Fatalf("body is not ToolUseResult, got %T", decoded.Body)
	}

	if toolResult.Success {
		t.Error("expected Success false, got true")
	}

	if toolResult.ErrorCode != "TIMEOUT" {
		t.Errorf("expected ErrorCode 'TIMEOUT', got %s", toolResult.ErrorCode)
	}

	if toolResult.ErrorMessage != "Tool execution timed out after 5000ms" {
		t.Errorf("expected ErrorMessage 'Tool execution timed out after 5000ms', got %s", toolResult.ErrorMessage)
	}
}

// TestCodec_EncodeDecode_ErrorMessage tests ErrorMessage with all severity levels
func TestCodec_EncodeDecode_ErrorMessage_Info(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.ErrorMessage{
		ID:             "err_1",
		ConversationID: "conv_abc",
		Code:           200,
		Message:        "Informational message",
		Severity:       protocol.SeverityInfo,
		Recoverable:    true,
		OriginatingID:  "msg_123",
	}

	env := protocol.NewEnvelope(-4, "conv_abc", protocol.TypeErrorMessage, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	errMsg, ok := decoded.Body.(*protocol.ErrorMessage)
	if !ok {
		t.Fatalf("body is not ErrorMessage, got %T", decoded.Body)
	}

	if errMsg.Severity != protocol.SeverityInfo {
		t.Errorf("expected Severity Info (0), got %d", errMsg.Severity)
	}

	if !errMsg.Recoverable {
		t.Error("expected Recoverable true, got false")
	}
}

func TestCodec_EncodeDecode_ErrorMessage_Critical(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.ErrorMessage{
		ID:             "err_2",
		ConversationID: "conv_abc",
		Code:           500,
		Message:        "Critical system failure",
		Severity:       protocol.SeverityCritical,
		Recoverable:    false,
	}

	env := protocol.NewEnvelope(-5, "conv_abc", protocol.TypeErrorMessage, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	errMsg, ok := decoded.Body.(*protocol.ErrorMessage)
	if !ok {
		t.Fatalf("body is not ErrorMessage, got %T", decoded.Body)
	}

	if errMsg.Severity != protocol.SeverityCritical {
		t.Errorf("expected Severity Critical (3), got %d", errMsg.Severity)
	}

	if errMsg.Recoverable {
		t.Error("expected Recoverable false, got true")
	}
}

// TestCodec_Decode_InvalidData tests error handling for invalid input
func TestCodec_Decode_InvalidData(t *testing.T) {
	codec := NewCodec()

	_, err := codec.Decode([]byte("invalid data"))
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}
}

func TestCodec_Decode_EmptyData(t *testing.T) {
	codec := NewCodec()

	_, err := codec.Decode([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestCodec_Decode_NilData(t *testing.T) {
	codec := NewCodec()

	_, err := codec.Decode(nil)
	if err == nil {
		t.Error("expected error for nil data, got nil")
	}
}

// TestCodec_Encode_NilEnvelope tests error handling for nil envelope
func TestCodec_Encode_NilEnvelope(t *testing.T) {
	codec := NewCodec()

	_, err := codec.Encode(nil)
	if err == nil {
		t.Error("expected error for nil envelope, got nil")
	}
}

func TestCodec_Encode_NilBody(t *testing.T) {
	codec := NewCodec()

	env := &protocol.Envelope{
		StanzaID:       1,
		ConversationID: "conv_abc",
		Type:           protocol.TypeUserMessage,
		Body:           nil,
	}

	_, err := codec.Encode(env)
	if err == nil {
		t.Error("expected error for nil body, got nil")
	}
}

func TestCodec_Encode_InvalidMessageType(t *testing.T) {
	codec := NewCodec()

	env := &protocol.Envelope{
		StanzaID:       1,
		ConversationID: "conv_abc",
		Type:           protocol.MessageType(999),
		Body:           &protocol.UserMessage{Content: "test"},
	}

	_, err := codec.Encode(env)
	if err == nil {
		t.Error("expected error for invalid message type, got nil")
	}
}

// TestCodec_AllMessageTypes tests round-trip encoding/decoding for all 16 message types
func TestCodec_AllMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType protocol.MessageType
		body    interface{}
	}{
		{
			"ErrorMessage",
			protocol.TypeErrorMessage,
			&protocol.ErrorMessage{
				ID:             "err_1",
				ConversationID: "conv_test",
				Code:           500,
				Message:        "error",
				Severity:       protocol.SeverityError,
			},
		},
		{
			"UserMessage",
			protocol.TypeUserMessage,
			&protocol.UserMessage{
				ID:             "user_1",
				ConversationID: "conv_test",
				Content:        "hello",
			},
		},
		{
			"AssistantMessage",
			protocol.TypeAssistantMessage,
			&protocol.AssistantMessage{
				ID:             "asst_1",
				ConversationID: "conv_test",
				Content:        "response",
			},
		},
		{
			"AudioChunk",
			protocol.TypeAudioChunk,
			&protocol.AudioChunk{
				ConversationID: "conv_test",
				Format:         "audio/opus",
				Sequence:       1,
				DurationMs:     100,
				Data:           []byte{0x00, 0x01},
			},
		},
		{
			"ReasoningStep",
			protocol.TypeReasoningStep,
			&protocol.ReasoningStep{
				ID:             "reason_1",
				MessageID:      "msg_1",
				ConversationID: "conv_test",
				Sequence:       1,
				Content:        "thinking...",
			},
		},
		{
			"ToolUseRequest",
			protocol.TypeToolUseRequest,
			&protocol.ToolUseRequest{
				ID:             "tool_req_1",
				MessageID:      "msg_123",
				ConversationID: "conv_test",
				ToolName:       "calculator",
				Parameters:     map[string]interface{}{"x": 5, "y": 3},
				Execution:      protocol.ToolExecutionServer,
			},
		},
		{
			"ToolUseResult",
			protocol.TypeToolUseResult,
			&protocol.ToolUseResult{
				ID:             "tool_res_1",
				RequestID:      "tool_req_1",
				ConversationID: "conv_test",
				Success:        true,
				Result:         "8",
			},
		},
		{
			"Acknowledgement",
			protocol.TypeAcknowledgement,
			&protocol.Acknowledgement{
				ConversationID: "conv_test",
				AckedStanzaID:  5,
				Success:        true,
			},
		},
		{
			"Transcription",
			protocol.TypeTranscription,
			&protocol.Transcription{
				ID:             "trans_1",
				ConversationID: "conv_test",
				Text:           "transcribed text",
				Final:          true,
				Confidence:     0.95,
				Language:       "en-US",
			},
		},
		{
			"ControlStop",
			protocol.TypeControlStop,
			&protocol.ControlStop{
				ConversationID: "conv_test",
				TargetID:       "msg_5",
				Reason:         "user requested",
				StopType:       protocol.StopTypeGeneration,
			},
		},
		{
			"ControlVariation",
			protocol.TypeControlVariation,
			&protocol.ControlVariation{
				ConversationID: "conv_test",
				TargetID:       "msg_3",
				Mode:           protocol.VariationTypeRegenerate,
			},
		},
		{
			"Configuration",
			protocol.TypeConfiguration,
			&protocol.Configuration{
				ConversationID:    "conv_test",
				LastSequenceSeen:  10,
				ClientVersion:     "1.0.0",
				PreferredLanguage: "en",
				Device:            "web",
				Features:          []string{"streaming", "audio_output"},
			},
		},
		{
			"StartAnswer",
			protocol.TypeStartAnswer,
			&protocol.StartAnswer{
				ID:                   "start_1",
				PreviousID:           "user_1",
				ConversationID:       "conv_test",
				AnswerType:           protocol.AnswerTypeText,
				PlannedSentenceCount: 3,
			},
		},
		{
			"MemoryTrace",
			protocol.TypeMemoryTrace,
			&protocol.MemoryTrace{
				ID:             "mem_1",
				MessageID:      "msg_1",
				ConversationID: "conv_test",
				MemoryID:       "memory_123",
				Content:        "remembered fact",
				Relevance:      0.85,
			},
		},
		{
			"Commentary",
			protocol.TypeCommentary,
			&protocol.Commentary{
				ID:             "comment_1",
				MessageID:      "msg_1",
				ConversationID: "conv_test",
				Content:        "internal thought",
				CommentaryType: "analysis",
			},
		},
		{
			"AssistantSentence",
			protocol.TypeAssistantSentence,
			&protocol.AssistantSentence{
				ID:             "sentence_1",
				PreviousID:     "start_1",
				ConversationID: "conv_test",
				Sequence:       1,
				Text:           "This is a sentence.",
				IsFinal:        false,
			},
		},
	}

	codec := NewCodec()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := protocol.NewEnvelope(1, "conv_test", tt.msgType, tt.body)

			// Encode
			data, err := codec.Encode(env)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			if len(data) == 0 {
				t.Error("Encode returned empty data")
			}

			// Decode
			decoded, err := codec.Decode(data)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if decoded.Type != tt.msgType {
				t.Errorf("type mismatch: got %d, want %d", decoded.Type, tt.msgType)
			}

			if decoded.StanzaID != 1 {
				t.Errorf("stanzaID mismatch: got %d, want 1", decoded.StanzaID)
			}

			if decoded.ConversationID != "conv_test" {
				t.Errorf("conversationID mismatch: got %s, want conv_test", decoded.ConversationID)
			}

			if decoded.Body == nil {
				t.Fatal("decoded body is nil")
			}
		})
	}
}

// TestCodec_RoundTrip tests that encoding followed by decoding produces identical messages
func TestCodec_RoundTrip_ComplexMessage(t *testing.T) {
	codec := NewCodec()

	original := &protocol.ToolUseRequest{
		ID:             "tool_complex",
		MessageID:      "msg_complex",
		ConversationID: "conv_roundtrip",
		ToolName:       "data_processor",
		Parameters: map[string]interface{}{
			"nested": map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			"array": []interface{}{"item1", "item2", "item3"},
			"bool":  false,
		},
		Execution: protocol.ToolExecutionClient,
		TimeoutMs: 15000,
	}

	env := protocol.NewEnvelope(100, "conv_roundtrip", protocol.TypeToolUseRequest, original)

	// Add metadata
	env.WithMeta("timestamp", int64(1234567890))
	env.WithTracing("trace-123", "span-456")

	// Encode
	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify envelope fields
	if decoded.StanzaID != 100 {
		t.Errorf("StanzaID mismatch: got %d, want 100", decoded.StanzaID)
	}

	if decoded.ConversationID != "conv_roundtrip" {
		t.Errorf("ConversationID mismatch: got %s, want conv_roundtrip", decoded.ConversationID)
	}

	// Verify metadata
	if decoded.Meta == nil {
		t.Fatal("Meta is nil")
	}

	if decoded.Meta[protocol.MetaKeyTraceID] != "trace-123" {
		t.Errorf("TraceID mismatch: got %v, want trace-123", decoded.Meta[protocol.MetaKeyTraceID])
	}

	// Verify body
	decodedMsg, ok := decoded.Body.(*protocol.ToolUseRequest)
	if !ok {
		t.Fatalf("body is not ToolUseRequest, got %T", decoded.Body)
	}

	if decodedMsg.ToolName != original.ToolName {
		t.Errorf("ToolName mismatch: got %s, want %s", decodedMsg.ToolName, original.ToolName)
	}

	if decodedMsg.TimeoutMs != original.TimeoutMs {
		t.Errorf("TimeoutMs mismatch: got %d, want %d", decodedMsg.TimeoutMs, original.TimeoutMs)
	}

	// Verify complex nested parameters
	nestedParam, ok := decodedMsg.Parameters["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("nested parameter is not a map")
	}

	if nestedParam["key1"] != "value1" {
		t.Errorf("nested key1 mismatch: got %v, want value1", nestedParam["key1"])
	}
}

// TestCodec_EncodeMessage tests the convenience method
func TestCodec_EncodeMessage(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.UserMessage{
		ID:             "msg_encode",
		ConversationID: "conv_encode",
		Content:        "test message",
	}

	data, err := codec.EncodeMessage(5, "conv_encode", protocol.TypeUserMessage, msg)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("EncodeMessage returned empty data")
	}

	// Decode to verify
	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.StanzaID != 5 {
		t.Errorf("StanzaID mismatch: got %d, want 5", decoded.StanzaID)
	}

	userMsg, ok := decoded.Body.(*protocol.UserMessage)
	if !ok {
		t.Fatalf("body is not UserMessage, got %T", decoded.Body)
	}

	if userMsg.Content != "test message" {
		t.Errorf("Content mismatch: got %s, want 'test message'", userMsg.Content)
	}
}

// TestCodec_EncodeDecode_WithAllSeverities tests all error severity levels
func TestCodec_ErrorMessage_AllSeverities(t *testing.T) {
	severities := []struct {
		name     string
		severity protocol.Severity
	}{
		{"Info", protocol.SeverityInfo},
		{"Warning", protocol.SeverityWarning},
		{"Error", protocol.SeverityError},
		{"Critical", protocol.SeverityCritical},
	}

	codec := NewCodec()

	for _, s := range severities {
		t.Run(s.name, func(t *testing.T) {
			msg := &protocol.ErrorMessage{
				ID:             "err_sev_test",
				ConversationID: "conv_test",
				Code:           100,
				Message:        "test error",
				Severity:       s.severity,
			}

			env := protocol.NewEnvelope(-1, "conv_test", protocol.TypeErrorMessage, msg)

			data, err := codec.Encode(env)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			decoded, err := codec.Decode(data)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			errMsg, ok := decoded.Body.(*protocol.ErrorMessage)
			if !ok {
				t.Fatalf("body is not ErrorMessage, got %T", decoded.Body)
			}

			if errMsg.Severity != s.severity {
				t.Errorf("Severity mismatch: got %d, want %d", errMsg.Severity, s.severity)
			}
		})
	}
}

// TestCodec_Configuration tests Configuration message with all features
func TestCodec_Configuration_AllFeatures(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.Configuration{
		ConversationID:    "conv_config",
		LastSequenceSeen:  42,
		ClientVersion:     "2.1.0",
		PreferredLanguage: "en-US",
		Device:            "mobile",
		Features: []string{
			"streaming",
			"partial_responses",
			"audio_output",
			"reasoning_steps",
			"tool_use",
		},
	}

	env := protocol.NewEnvelope(1, "conv_config", protocol.TypeConfiguration, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	config, ok := decoded.Body.(*protocol.Configuration)
	if !ok {
		t.Fatalf("body is not Configuration, got %T", decoded.Body)
	}

	if len(config.Features) != 5 {
		t.Errorf("expected 5 features, got %d", len(config.Features))
	}

	if config.ClientVersion != "2.1.0" {
		t.Errorf("expected ClientVersion '2.1.0', got %s", config.ClientVersion)
	}

	if config.LastSequenceSeen != 42 {
		t.Errorf("expected LastSequenceSeen 42, got %d", config.LastSequenceSeen)
	}
}

// TestCodec_AudioChunk tests AudioChunk with binary data
func TestCodec_AudioChunk_BinaryData(t *testing.T) {
	codec := NewCodec()

	// Create some sample audio data
	audioData := make([]byte, 1024)
	for i := range audioData {
		audioData[i] = byte(i % 256)
	}

	msg := &protocol.AudioChunk{
		ConversationID: "conv_audio",
		Format:         "audio/opus",
		Sequence:       10,
		DurationMs:     200,
		TrackSID:       "track_abc123",
		Data:           audioData,
	}

	env := protocol.NewEnvelope(-10, "conv_audio", protocol.TypeAudioChunk, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	chunk, ok := decoded.Body.(*protocol.AudioChunk)
	if !ok {
		t.Fatalf("body is not AudioChunk, got %T", decoded.Body)
	}

	if len(chunk.Data) != 1024 {
		t.Errorf("expected Data length 1024, got %d", len(chunk.Data))
	}

	if chunk.Format != "audio/opus" {
		t.Errorf("expected Format 'audio/opus', got %s", chunk.Format)
	}

	if chunk.Sequence != 10 {
		t.Errorf("expected Sequence 10, got %d", chunk.Sequence)
	}
}

// TestCodec_Transcription tests Transcription with all fields
func TestCodec_Transcription_Complete(t *testing.T) {
	codec := NewCodec()

	msg := &protocol.Transcription{
		ID:             "trans_full",
		PreviousID:     "trans_partial",
		ConversationID: "conv_trans",
		Text:           "This is the full transcription",
		Final:          true,
		Confidence:     0.98,
		Language:       "en-US",
	}

	env := protocol.NewEnvelope(20, "conv_trans", protocol.TypeTranscription, msg)

	data, err := codec.Encode(env)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := codec.Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	trans, ok := decoded.Body.(*protocol.Transcription)
	if !ok {
		t.Fatalf("body is not Transcription, got %T", decoded.Body)
	}

	if !trans.Final {
		t.Error("expected Final true, got false")
	}

	if trans.Confidence != 0.98 {
		t.Errorf("expected Confidence 0.98, got %f", trans.Confidence)
	}

	if trans.Language != "en-US" {
		t.Errorf("expected Language 'en-US', got %s", trans.Language)
	}
}
