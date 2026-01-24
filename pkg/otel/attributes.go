package otel

import "go.opentelemetry.io/otel/attribute"

// Standard attribute keys for Alicia services.
const (
	AttrSessionID           = "session.id"
	AttrUserID              = "user.id"
	AttrConversationID      = "conversation.id"
	AttrMessageID           = "message.id"
	AttrRequestID           = "request.id"
	AttrRequestType         = "request.type"
	AttrLLMModel            = "llm.model"
	AttrLLMProvider         = "llm.provider"
	AttrLLMPromptTokens     = "llm.usage.prompt_tokens"
	AttrLLMCompletionTokens = "llm.usage.completion_tokens"
	AttrLLMTotalTokens      = "llm.usage.total_tokens"
	AttrToolName            = "tool.name"
	AttrToolID              = "tool.id"
	AttrToolStatus          = "tool.status"
	AttrASRModel            = "asr.model"
	AttrASRDurationMs       = "asr.duration_ms"
	AttrASRLatencyMs        = "asr.latency_ms"
	AttrTTSModel            = "tts.model"
	AttrTTSVoice            = "tts.voice"
	AttrTTSDurationMs       = "tts.duration_ms"
	AttrTTSLatencyMs        = "tts.latency_ms"
	AttrWSMessageType       = "ws.message_type"
	AttrWSDirection         = "ws.direction"
	// Langfuse prompt tracking
	AttrPromptName    = "langfuse.prompt.name"
	AttrPromptVersion = "langfuse.prompt.version"
	// Langfuse trace naming
	AttrTraceName = "langfuse.trace.name"
	// Alicia response type tag for managed evaluators
	AttrAliciaType = "alicia.type"
)

func SessionID(id string) attribute.KeyValue      { return attribute.String(AttrSessionID, id) }
func UserID(id string) attribute.KeyValue         { return attribute.String(AttrUserID, id) }
func ConversationID(id string) attribute.KeyValue { return attribute.String(AttrConversationID, id) }
func MessageID(id string) attribute.KeyValue      { return attribute.String(AttrMessageID, id) }
func RequestID(id string) attribute.KeyValue      { return attribute.String(AttrRequestID, id) }
func RequestType(t string) attribute.KeyValue     { return attribute.String(AttrRequestType, t) }

func LLMModel(model string) attribute.KeyValue       { return attribute.String(AttrLLMModel, model) }
func LLMProvider(provider string) attribute.KeyValue { return attribute.String(AttrLLMProvider, provider) }
func LLMPromptTokens(n int) attribute.KeyValue       { return attribute.Int(AttrLLMPromptTokens, n) }
func LLMCompletionTokens(n int) attribute.KeyValue   { return attribute.Int(AttrLLMCompletionTokens, n) }
func LLMTotalTokens(n int) attribute.KeyValue        { return attribute.Int(AttrLLMTotalTokens, n) }

func ToolName(name string) attribute.KeyValue   { return attribute.String(AttrToolName, name) }
func ToolID(id string) attribute.KeyValue       { return attribute.String(AttrToolID, id) }
func ToolStatus(status string) attribute.KeyValue { return attribute.String(AttrToolStatus, status) }

func ASRModel(model string) attribute.KeyValue  { return attribute.String(AttrASRModel, model) }
func ASRDurationMs(ms int64) attribute.KeyValue { return attribute.Int64(AttrASRDurationMs, ms) }
func ASRLatencyMs(ms int64) attribute.KeyValue  { return attribute.Int64(AttrASRLatencyMs, ms) }

func TTSModel(model string) attribute.KeyValue  { return attribute.String(AttrTTSModel, model) }
func TTSVoice(voice string) attribute.KeyValue  { return attribute.String(AttrTTSVoice, voice) }
func TTSDurationMs(ms int64) attribute.KeyValue { return attribute.Int64(AttrTTSDurationMs, ms) }
func TTSLatencyMs(ms int64) attribute.KeyValue  { return attribute.Int64(AttrTTSLatencyMs, ms) }

func WSMessageType(t string) attribute.KeyValue { return attribute.String(AttrWSMessageType, t) }
func WSDirection(dir string) attribute.KeyValue { return attribute.String(AttrWSDirection, dir) }

func PromptName(name string) attribute.KeyValue    { return attribute.String(AttrPromptName, name) }
func PromptVersion(version int) attribute.KeyValue { return attribute.Int(AttrPromptVersion, version) }
func TraceName(name string) attribute.KeyValue     { return attribute.String(AttrTraceName, name) }
func AliciaType(t string) attribute.KeyValue       { return attribute.String(AttrAliciaType, t) }

// AliciaResponseTag is the standard tag value used for traces that should be
// evaluated by Langfuse managed evaluators.
const AliciaResponseTag = "alicia-response"
