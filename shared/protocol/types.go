package protocol

type MessageType uint16

const (
	TypeError             MessageType = 1
	TypeUserMessage       MessageType = 2
	TypeAssistantMsg      MessageType = 3
	TypeReasoningStep     MessageType = 5
	TypeToolUseRequest    MessageType = 6
	TypeToolUseResult     MessageType = 7
	TypeAck               MessageType = 8
	TypeStartAnswer       MessageType = 13
	TypeMemoryTrace       MessageType = 14
	TypeAssistantSentence MessageType = 16
	TypeGenRequest        MessageType = 33
	TypeThinkingSummary   MessageType = 34
	TypeTitleUpdate       MessageType = 35
	TypeSubscribe         MessageType = 40
	TypeUnsubscribe       MessageType = 41
	TypeSubscribeAck      MessageType = 42
	TypeUnsubscribeAck    MessageType = 43
	TypeBranchUpdate      MessageType = 50
	TypeVoiceJoinRequest  MessageType = 51
	TypeVoiceJoinAck      MessageType = 52
	TypeVoiceLeaveRequest MessageType = 53
	TypeVoiceLeaveAck     MessageType = 54
	TypeVoiceStatus       MessageType = 55
	TypeVoiceSpeaking     MessageType = 56
	TypePreferencesUpdate          MessageType = 60
	TypeAssistantToolsRegister     MessageType = 70
	TypeAssistantToolsAck          MessageType = 71
	TypeAssistantHeartbeat         MessageType = 72
	TypeGenerationComplete         MessageType = 80
	TypeWhatsAppPairRequest        MessageType = 90
	TypeWhatsAppQR                 MessageType = 91
	TypeWhatsAppStatus             MessageType = 92
	TypeWhatsAppDebug              MessageType = 93
)

type GenerationComplete struct {
	MessageID      string `msgpack:"messageId" json:"messageId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Success        bool   `msgpack:"success" json:"success"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type Error struct {
	Code           string `msgpack:"code" json:"code"`
	Message        string `msgpack:"message" json:"message"`
	MessageID      string `msgpack:"messageId,omitempty" json:"messageId,omitempty"`
	ConversationID string `msgpack:"conversationId,omitempty" json:"conversationId,omitempty"`
}

type UserMessage struct {
	ID             string `msgpack:"id" json:"id"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Content        string `msgpack:"content" json:"content"`
	PreviousID     string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
}

type AssistantMessage struct {
	ID             string `msgpack:"id" json:"id"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Content        string `msgpack:"content" json:"content"`
	PreviousID     string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
	Reasoning      string `msgpack:"reasoning,omitempty" json:"reasoning,omitempty"`
	Timestamp      int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

type AssistantSentence struct {
	ID             string `msgpack:"id,omitempty" json:"id,omitempty"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	PreviousID     string `msgpack:"previousId" json:"previousId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Sequence       int    `msgpack:"sequence" json:"sequence"`
	Text           string `msgpack:"text" json:"text"`
	IsFinal        bool   `msgpack:"isFinal,omitempty" json:"isFinal,omitempty"`
}

type StartAnswer struct {
	MessageID      string `msgpack:"messageId" json:"messageId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	PreviousID     string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
}

type ToolUseRequest struct {
	ID             string         `msgpack:"id" json:"id"`
	MessageID      string         `msgpack:"messageId" json:"messageId"`
	ConversationID string         `msgpack:"conversationId" json:"conversationId"`
	ToolName       string         `msgpack:"toolName" json:"toolName"`
	Arguments      map[string]any `msgpack:"arguments" json:"arguments"`
	Execution      string         `msgpack:"execution,omitempty" json:"execution,omitempty"`
}

type ToolUseResult struct {
	ID             string `msgpack:"id" json:"id"`
	RequestID      string `msgpack:"requestId" json:"requestId"`
	MessageID      string `msgpack:"messageId,omitempty" json:"messageId,omitempty"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Success        bool   `msgpack:"success" json:"success"`
	Result         any    `msgpack:"result,omitempty" json:"result,omitempty"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type MemoryTrace struct {
	ID             string  `msgpack:"id" json:"id"`
	MemoryID       string  `msgpack:"memoryId" json:"memoryId"`
	MessageID      string  `msgpack:"messageId" json:"messageId"`
	ConversationID string  `msgpack:"conversationId" json:"conversationId"`
	Content        string  `msgpack:"content" json:"content"`
	Relevance      float32 `msgpack:"relevance" json:"relevance"`
}

type ThinkingSummary struct {
	ID             string  `msgpack:"id" json:"id"`
	MessageID      string  `msgpack:"messageId" json:"messageId"`
	ConversationID string  `msgpack:"conversationId" json:"conversationId"`
	Content        string  `msgpack:"content" json:"content"`
	Progress       float32 `msgpack:"progress,omitempty" json:"progress,omitempty"`
	Timestamp      int64   `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

type TitleUpdate struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Title          string `msgpack:"title" json:"title"`
}

type GenerationRequest struct {
	ConversationID  string `msgpack:"conversationId" json:"conversationId"`
	MessageID       string `msgpack:"messageId" json:"messageId"`
	PreviousID      string `msgpack:"previousId,omitempty" json:"previousId,omitempty"`
	RequestType     string `msgpack:"requestType" json:"requestType"`
	NewContent      string `msgpack:"newContent,omitempty" json:"newContent,omitempty"`
	EnableTools     bool   `msgpack:"enableTools" json:"enableTools"`
	EnableReasoning bool   `msgpack:"enableReasoning" json:"enableReasoning"`
	EnableStreaming bool   `msgpack:"enableStreaming" json:"enableStreaming"`
	UsePareto       bool   `msgpack:"usePareto" json:"usePareto"`
	Timestamp       int64  `msgpack:"timestamp,omitempty" json:"timestamp,omitempty"`
}

type Subscribe struct {
	ConversationID string `msgpack:"conversationId,omitempty" json:"conversationId,omitempty"`
	AgentMode      bool   `msgpack:"agentMode,omitempty" json:"agentMode,omitempty"`
	VoiceMode      bool   `msgpack:"voiceMode,omitempty" json:"voiceMode,omitempty"`
	MonitorMode    bool   `msgpack:"monitorMode,omitempty" json:"monitorMode,omitempty"`
	AssistantMode  bool   `msgpack:"assistantMode,omitempty" json:"assistantMode,omitempty"`
	WhatsAppMode   bool   `msgpack:"whatsappMode,omitempty" json:"whatsappMode,omitempty"`
}

type Unsubscribe struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
}

type SubscribeAck struct {
	ConversationID string `msgpack:"conversationId,omitempty" json:"conversationId,omitempty"`
	AgentMode      bool   `msgpack:"agentMode,omitempty" json:"agentMode,omitempty"`
	Success        bool   `msgpack:"success" json:"success"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type UnsubscribeAck struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Success        bool   `msgpack:"success" json:"success"`
}

type Ack struct{}

type ReasoningStep struct {
	ID             string `msgpack:"id" json:"id"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Sequence       int    `msgpack:"sequence" json:"sequence"`
	Content        string `msgpack:"content" json:"content"`
}

type SiblingInfo struct {
	ID        string `msgpack:"id" json:"id"`
	Content   string `msgpack:"content" json:"content"`
	CreatedAt string `msgpack:"createdAt" json:"createdAt"`
}

type BranchUpdate struct {
	ConversationID  string        `msgpack:"conversationId" json:"conversationId"`
	ParentMessageID string        `msgpack:"parentMessageId" json:"parentMessageId"`
	NewSibling      SiblingInfo   `msgpack:"newSibling" json:"newSibling"`
	AllSiblings     []SiblingInfo `msgpack:"allSiblings" json:"allSiblings"`
	TotalCount      int           `msgpack:"totalCount" json:"totalCount"`
}

type VoiceJoinRequest struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	UserID         string `msgpack:"userId" json:"userId"`
}

type VoiceJoinAck struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Success        bool   `msgpack:"success" json:"success"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
	SampleRate     int    `msgpack:"sampleRate,omitempty" json:"sampleRate,omitempty"`
}

type VoiceLeaveRequest struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
}

type VoiceLeaveAck struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Success        bool   `msgpack:"success" json:"success"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type VoiceSpeaking struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	MessageID      string `msgpack:"messageId" json:"messageId"`
	Speaking       bool   `msgpack:"speaking" json:"speaking"`
	SentenceSeq    int    `msgpack:"sentenceSeq,omitempty" json:"sentenceSeq,omitempty"`
}

type VoiceStatus struct {
	ConversationID string `msgpack:"conversationId" json:"conversationId"`
	Status         string `msgpack:"status" json:"status"` // "queue_full", "queue_ok", "speaking", "idle"
	QueueLength    int    `msgpack:"queueLength" json:"queueLength"`
	Error          string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type PreferencesUpdate struct {
	UserID                   string  `msgpack:"userId" json:"userId"`
	Theme                    string  `msgpack:"theme" json:"theme"`
	AudioOutputEnabled       bool    `msgpack:"audioOutputEnabled" json:"audioOutputEnabled"`
	VoiceSpeed               float32 `msgpack:"voiceSpeed" json:"voiceSpeed"`
	MemoryMinImportance      *int    `msgpack:"memoryMinImportance" json:"memoryMinImportance"`
	MemoryMinHistorical      *int    `msgpack:"memoryMinHistorical" json:"memoryMinHistorical"`
	MemoryMinPersonal        *int    `msgpack:"memoryMinPersonal" json:"memoryMinPersonal"`
	MemoryMinFactual         *int    `msgpack:"memoryMinFactual" json:"memoryMinFactual"`
	MemoryRetrievalCount     int     `msgpack:"memoryRetrievalCount" json:"memoryRetrievalCount"`
	MaxTokens                int     `msgpack:"maxTokens" json:"maxTokens"`
	Temperature              float32 `msgpack:"temperature" json:"temperature"`
	ParetoTargetScore        float32 `msgpack:"paretoTargetScore" json:"paretoTargetScore"`
	ParetoMaxGenerations     int     `msgpack:"paretoMaxGenerations" json:"paretoMaxGenerations"`
	ParetoBranchesPerGen     int     `msgpack:"paretoBranchesPerGen" json:"paretoBranchesPerGen"`
	ParetoArchiveSize        int     `msgpack:"paretoArchiveSize" json:"paretoArchiveSize"`
	ParetoEnableCrossover    bool    `msgpack:"paretoEnableCrossover" json:"paretoEnableCrossover"`
	NotesSimilarityThreshold float32 `msgpack:"notesSimilarityThreshold" json:"notesSimilarityThreshold"`
	NotesMaxCount            int     `msgpack:"notesMaxCount" json:"notesMaxCount"`
	ConfirmDeleteMemory      bool    `msgpack:"confirmDeleteMemory" json:"confirmDeleteMemory"`
	ShowRelevanceScores      bool    `msgpack:"showRelevanceScores" json:"showRelevanceScores"`
}

type AssistantToolsRegister struct {
	Tools []AssistantTool `msgpack:"tools" json:"tools"`
}

type AssistantTool struct {
	Name        string         `msgpack:"name" json:"name"`
	Description string         `msgpack:"description" json:"description"`
	InputSchema map[string]any `msgpack:"inputSchema,omitempty" json:"inputSchema,omitempty"`
}

type AssistantToolsAck struct {
	Success   bool   `msgpack:"success" json:"success"`
	ToolCount int    `msgpack:"toolCount" json:"toolCount"`
	Error     string `msgpack:"error,omitempty" json:"error,omitempty"`
}

type WhatsAppPairRequest struct {
	Role string `msgpack:"role" json:"role"`
}

type WhatsAppQR struct {
	Code  string `msgpack:"code" json:"code"`
	Event string `msgpack:"event" json:"event"` // "code", "login", "timeout", "error"
	Role  string `msgpack:"role" json:"role"`
}

type WhatsAppStatus struct {
	Connected bool   `msgpack:"connected" json:"connected"`
	Phone     string `msgpack:"phone,omitempty" json:"phone,omitempty"`
	Error     string `msgpack:"error,omitempty" json:"error,omitempty"`
	Role      string `msgpack:"role" json:"role"`
}

type WhatsAppDebug struct {
	Role   string `msgpack:"role" json:"role"`
	Event  string `msgpack:"event" json:"event"`
	Detail string `msgpack:"detail" json:"detail"`
}
