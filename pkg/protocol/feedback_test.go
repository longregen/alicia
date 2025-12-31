package protocol

import (
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// =============================================================================
// New Message Type Tests (Types 20-27)
// =============================================================================

func TestFeedbackMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType MessageType
		want    string
	}{
		{TypeFeedback, "Feedback"},
		{TypeFeedbackConfirmation, "FeedbackConfirmation"},
		{TypeUserNote, "UserNote"},
		{TypeNoteConfirmation, "NoteConfirmation"},
		{TypeMemoryAction, "MemoryAction"},
		{TypeMemoryConfirmation, "MemoryConfirmation"},
		{TypeServerInfo, "ServerInfo"},
		{TypeSessionStats, "SessionStats"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.msgType.String(); got != tt.want {
				t.Errorf("MessageType(%d).String() = %q, want %q", tt.msgType, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Feedback Message Tests
// =============================================================================

func TestFeedbackFields(t *testing.T) {
	msg := Feedback{
		ID:             "feedback_123",
		ConversationID: "conv_abc",
		MessageID:      "msg_456",
		TargetType:     "message",
		TargetID:       "msg_456",
		Vote:           "up",
		QuickFeedback:  "helpful",
		Note:           "Great response!",
		Timestamp:      1234567890,
	}

	if msg.ID != "feedback_123" {
		t.Errorf("expected ID 'feedback_123', got %s", msg.ID)
	}
	if msg.TargetType != "message" {
		t.Errorf("expected TargetType 'message', got %s", msg.TargetType)
	}
	if msg.Vote != "up" {
		t.Errorf("expected Vote 'up', got %s", msg.Vote)
	}
	if msg.QuickFeedback != "helpful" {
		t.Errorf("expected QuickFeedback 'helpful', got %s", msg.QuickFeedback)
	}
	if msg.Timestamp != 1234567890 {
		t.Errorf("expected Timestamp 1234567890, got %d", msg.Timestamp)
	}
}

func TestFeedbackMarshalUnmarshal(t *testing.T) {
	original := Feedback{
		ID:             "feedback_123",
		ConversationID: "conv_abc",
		MessageID:      "msg_456",
		TargetType:     "tool_use",
		TargetID:       "tool_789",
		Vote:           "down",
		QuickFeedback:  "wrong_tool",
		Note:           "Should have used different tool",
		Timestamp:      1234567890,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal Feedback: %v", err)
	}

	var decoded Feedback
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal Feedback: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, decoded.ID)
	}
	if decoded.TargetType != original.TargetType {
		t.Errorf("expected TargetType %s, got %s", original.TargetType, decoded.TargetType)
	}
	if decoded.Vote != original.Vote {
		t.Errorf("expected Vote %s, got %s", original.Vote, decoded.Vote)
	}
}

// =============================================================================
// FeedbackConfirmation Tests
// =============================================================================

func TestFeedbackConfirmationFields(t *testing.T) {
	specialVotes := map[string]int{
		"critical": 2,
	}
	msg := FeedbackConfirmation{
		FeedbackID: "feedback_123",
		TargetType: "memory",
		TargetID:   "mem_456",
		Aggregates: FeedbackAggregates{
			Upvotes:      10,
			Downvotes:    2,
			SpecialVotes: specialVotes,
		},
		UserVote: "up",
	}

	if msg.FeedbackID != "feedback_123" {
		t.Errorf("expected FeedbackID 'feedback_123', got %s", msg.FeedbackID)
	}
	if msg.Aggregates.Upvotes != 10 {
		t.Errorf("expected Upvotes 10, got %d", msg.Aggregates.Upvotes)
	}
	if msg.Aggregates.Downvotes != 2 {
		t.Errorf("expected Downvotes 2, got %d", msg.Aggregates.Downvotes)
	}
	if msg.UserVote != "up" {
		t.Errorf("expected UserVote 'up', got %s", msg.UserVote)
	}
}

func TestFeedbackConfirmationMarshalUnmarshal(t *testing.T) {
	original := FeedbackConfirmation{
		FeedbackID: "feedback_123",
		TargetType: "reasoning",
		TargetID:   "reason_789",
		Aggregates: FeedbackAggregates{
			Upvotes:   5,
			Downvotes: 1,
		},
		UserVote: "down",
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal FeedbackConfirmation: %v", err)
	}

	var decoded FeedbackConfirmation
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal FeedbackConfirmation: %v", err)
	}

	if decoded.Aggregates.Upvotes != original.Aggregates.Upvotes {
		t.Errorf("expected Upvotes %d, got %d", original.Aggregates.Upvotes, decoded.Aggregates.Upvotes)
	}
}

// =============================================================================
// UserNote Tests
// =============================================================================

func TestUserNoteFields(t *testing.T) {
	msg := UserNote{
		ID:        "note_123",
		MessageID: "msg_456",
		Content:   "This needs improvement",
		Category:  "improvement",
		Action:    "create",
		Timestamp: 1234567890,
	}

	if msg.ID != "note_123" {
		t.Errorf("expected ID 'note_123', got %s", msg.ID)
	}
	if msg.Category != "improvement" {
		t.Errorf("expected Category 'improvement', got %s", msg.Category)
	}
	if msg.Action != "create" {
		t.Errorf("expected Action 'create', got %s", msg.Action)
	}
}

func TestUserNoteMarshalUnmarshal(t *testing.T) {
	original := UserNote{
		ID:        "note_123",
		MessageID: "msg_456",
		Content:   "Factually incorrect",
		Category:  "correction",
		Action:    "update",
		Timestamp: 1234567890,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal UserNote: %v", err)
	}

	var decoded UserNote
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal UserNote: %v", err)
	}

	if decoded.Content != original.Content {
		t.Errorf("expected Content %s, got %s", original.Content, decoded.Content)
	}
	if decoded.Category != original.Category {
		t.Errorf("expected Category %s, got %s", original.Category, decoded.Category)
	}
}

// =============================================================================
// NoteConfirmation Tests
// =============================================================================

func TestNoteConfirmationFields(t *testing.T) {
	msg := NoteConfirmation{
		NoteID:    "note_123",
		MessageID: "msg_456",
		Success:   true,
	}

	if msg.NoteID != "note_123" {
		t.Errorf("expected NoteID 'note_123', got %s", msg.NoteID)
	}
	if !msg.Success {
		t.Error("expected Success to be true")
	}
}

func TestNoteConfirmationMarshalUnmarshal(t *testing.T) {
	original := NoteConfirmation{
		NoteID:    "note_456",
		MessageID: "msg_789",
		Success:   false,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal NoteConfirmation: %v", err)
	}

	var decoded NoteConfirmation
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal NoteConfirmation: %v", err)
	}

	if decoded.Success != original.Success {
		t.Errorf("expected Success %v, got %v", original.Success, decoded.Success)
	}
}

// =============================================================================
// MemoryAction Tests
// =============================================================================

func TestMemoryActionFields(t *testing.T) {
	memData := &MemoryData{
		Content:  "User prefers TypeScript",
		Category: "preference",
		Pinned:   true,
	}
	msg := MemoryAction{
		ID:        "mem_action_123",
		Action:    "create",
		Memory:    memData,
		Timestamp: 1234567890,
	}

	if msg.Action != "create" {
		t.Errorf("expected Action 'create', got %s", msg.Action)
	}
	if msg.Memory.Content != "User prefers TypeScript" {
		t.Errorf("expected Memory.Content 'User prefers TypeScript', got %s", msg.Memory.Content)
	}
	if !msg.Memory.Pinned {
		t.Error("expected Memory.Pinned to be true")
	}
}

func TestMemoryActionMarshalUnmarshal(t *testing.T) {
	memData := &MemoryData{
		Content:  "Working on React projects",
		Category: "context",
		Pinned:   false,
	}
	original := MemoryAction{
		ID:        "mem_action_456",
		Action:    "update",
		Memory:    memData,
		Timestamp: 1234567890,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal MemoryAction: %v", err)
	}

	var decoded MemoryAction
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal MemoryAction: %v", err)
	}

	if decoded.Action != original.Action {
		t.Errorf("expected Action %s, got %s", original.Action, decoded.Action)
	}
	if decoded.Memory.Content != original.Memory.Content {
		t.Errorf("expected Memory.Content %s, got %s", original.Memory.Content, decoded.Memory.Content)
	}
}

func TestMemoryActionDelete(t *testing.T) {
	msg := MemoryAction{
		ID:        "mem_action_789",
		Action:    "delete",
		Memory:    nil, // No memory data for delete
		Timestamp: 1234567890,
	}

	if msg.Action != "delete" {
		t.Errorf("expected Action 'delete', got %s", msg.Action)
	}
	if msg.Memory != nil {
		t.Error("expected Memory to be nil for delete action")
	}
}

// =============================================================================
// MemoryConfirmation Tests
// =============================================================================

func TestMemoryConfirmationFields(t *testing.T) {
	msg := MemoryConfirmation{
		MemoryID: "mem_123",
		Action:   "pin",
		Success:  true,
	}

	if msg.MemoryID != "mem_123" {
		t.Errorf("expected MemoryID 'mem_123', got %s", msg.MemoryID)
	}
	if msg.Action != "pin" {
		t.Errorf("expected Action 'pin', got %s", msg.Action)
	}
	if !msg.Success {
		t.Error("expected Success to be true")
	}
}

func TestMemoryConfirmationMarshalUnmarshal(t *testing.T) {
	original := MemoryConfirmation{
		MemoryID: "mem_456",
		Action:   "archive",
		Success:  false,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal MemoryConfirmation: %v", err)
	}

	var decoded MemoryConfirmation
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal MemoryConfirmation: %v", err)
	}

	if decoded.Action != original.Action {
		t.Errorf("expected Action %s, got %s", original.Action, decoded.Action)
	}
	if decoded.Success != original.Success {
		t.Errorf("expected Success %v, got %v", original.Success, decoded.Success)
	}
}

// =============================================================================
// ServerInfo Tests
// =============================================================================

func TestServerInfoFields(t *testing.T) {
	mcpServers := []MCPServerInfo{
		{Name: "filesystem", Status: "connected"},
		{Name: "github", Status: "connected"},
	}
	msg := ServerInfo{
		Connection: ConnectionInfo{
			Status:  "connected",
			Latency: 42,
		},
		Model: ModelInfo{
			Name:     "claude-3-opus",
			Provider: "anthropic",
		},
		MCPServers: mcpServers,
	}

	if msg.Connection.Status != "connected" {
		t.Errorf("expected Connection.Status 'connected', got %s", msg.Connection.Status)
	}
	if msg.Connection.Latency != 42 {
		t.Errorf("expected Connection.Latency 42, got %d", msg.Connection.Latency)
	}
	if msg.Model.Name != "claude-3-opus" {
		t.Errorf("expected Model.Name 'claude-3-opus', got %s", msg.Model.Name)
	}
	if len(msg.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(msg.MCPServers))
	}
}

func TestServerInfoMarshalUnmarshal(t *testing.T) {
	original := ServerInfo{
		Connection: ConnectionInfo{
			Status:  "reconnecting",
			Latency: 150,
		},
		Model: ModelInfo{
			Name:     "claude-3-sonnet",
			Provider: "anthropic",
		},
		MCPServers: []MCPServerInfo{
			{Name: "database", Status: "disconnected"},
		},
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ServerInfo: %v", err)
	}

	var decoded ServerInfo
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal ServerInfo: %v", err)
	}

	if decoded.Connection.Status != original.Connection.Status {
		t.Errorf("expected Connection.Status %s, got %s", original.Connection.Status, decoded.Connection.Status)
	}
	if decoded.Model.Name != original.Model.Name {
		t.Errorf("expected Model.Name %s, got %s", original.Model.Name, decoded.Model.Name)
	}
	if len(decoded.MCPServers) != len(original.MCPServers) {
		t.Errorf("expected %d MCP servers, got %d", len(original.MCPServers), len(decoded.MCPServers))
	}
}

// =============================================================================
// SessionStats Tests
// =============================================================================

func TestSessionStatsFields(t *testing.T) {
	msg := SessionStats{
		MessageCount:    24,
		ToolCallCount:   8,
		MemoriesUsed:    3,
		SessionDuration: 2700,
	}

	if msg.MessageCount != 24 {
		t.Errorf("expected MessageCount 24, got %d", msg.MessageCount)
	}
	if msg.ToolCallCount != 8 {
		t.Errorf("expected ToolCallCount 8, got %d", msg.ToolCallCount)
	}
	if msg.MemoriesUsed != 3 {
		t.Errorf("expected MemoriesUsed 3, got %d", msg.MemoriesUsed)
	}
	if msg.SessionDuration != 2700 {
		t.Errorf("expected SessionDuration 2700, got %d", msg.SessionDuration)
	}
}

func TestSessionStatsMarshalUnmarshal(t *testing.T) {
	original := SessionStats{
		MessageCount:    100,
		ToolCallCount:   25,
		MemoriesUsed:    10,
		SessionDuration: 5400,
	}

	data, err := msgpack.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal SessionStats: %v", err)
	}

	var decoded SessionStats
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal SessionStats: %v", err)
	}

	if decoded.MessageCount != original.MessageCount {
		t.Errorf("expected MessageCount %d, got %d", original.MessageCount, decoded.MessageCount)
	}
	if decoded.ToolCallCount != original.ToolCallCount {
		t.Errorf("expected ToolCallCount %d, got %d", original.ToolCallCount, decoded.ToolCallCount)
	}
}

// =============================================================================
// Envelope Integration Tests
// =============================================================================

func TestFeedbackEnvelope(t *testing.T) {
	feedback := Feedback{
		ID:             "feedback_123",
		ConversationID: "conv_abc",
		MessageID:      "msg_456",
		TargetType:     "message",
		TargetID:       "msg_456",
		Vote:           "up",
		Timestamp:      1234567890,
	}

	env := NewEnvelope(1, "conv_abc", TypeFeedback, feedback)

	if env.Type != TypeFeedback {
		t.Errorf("expected Type TypeFeedback, got %v", env.Type)
	}

	data, err := msgpack.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var decoded Envelope
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if decoded.Type != TypeFeedback {
		t.Errorf("expected Type TypeFeedback, got %v", decoded.Type)
	}
}

func TestServerInfoEnvelope(t *testing.T) {
	serverInfo := ServerInfo{
		Connection: ConnectionInfo{
			Status:  "connected",
			Latency: 42,
		},
		Model: ModelInfo{
			Name:     "claude-3-opus",
			Provider: "anthropic",
		},
		MCPServers: []MCPServerInfo{},
	}

	env := NewEnvelope(-1, "conv_abc", TypeServerInfo, serverInfo)

	if env.Type != TypeServerInfo {
		t.Errorf("expected Type TypeServerInfo, got %v", env.Type)
	}
	if env.StanzaID != -1 {
		t.Errorf("expected StanzaID -1 for server message, got %d", env.StanzaID)
	}

	data, err := msgpack.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var decoded Envelope
	err = msgpack.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if decoded.Type != TypeServerInfo {
		t.Errorf("expected Type TypeServerInfo, got %v", decoded.Type)
	}
}
