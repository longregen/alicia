// Package protocol re-exports shared protocol types for API consumers.
package protocol

import "github.com/longregen/alicia/shared/protocol"

// Re-export MessageType and constants
type MessageType = protocol.MessageType

const (
	TypeError            = protocol.TypeError
	TypeUserMessage      = protocol.TypeUserMessage
	TypeAssistantMsg     = protocol.TypeAssistantMsg
	TypeReasoningStep    = protocol.TypeReasoningStep
	TypeToolUseRequest   = protocol.TypeToolUseRequest
	TypeToolUseResult    = protocol.TypeToolUseResult
	TypeAck              = protocol.TypeAck
	TypeStartAnswer      = protocol.TypeStartAnswer
	TypeMemoryTrace      = protocol.TypeMemoryTrace
	TypeAssistantSentence = protocol.TypeAssistantSentence
	TypeGenRequest       = protocol.TypeGenRequest
	TypeThinkingSummary  = protocol.TypeThinkingSummary
	TypeTitleUpdate      = protocol.TypeTitleUpdate
	TypeSubscribe        = protocol.TypeSubscribe
	TypeUnsubscribe      = protocol.TypeUnsubscribe
	TypeSubscribeAck     = protocol.TypeSubscribeAck
	TypeUnsubscribeAck   = protocol.TypeUnsubscribeAck
	TypeBranchUpdate     = protocol.TypeBranchUpdate
	TypeVoiceJoinRequest  = protocol.TypeVoiceJoinRequest
	TypeVoiceJoinAck      = protocol.TypeVoiceJoinAck
	TypeVoiceLeaveRequest = protocol.TypeVoiceLeaveRequest
	TypeVoiceLeaveAck     = protocol.TypeVoiceLeaveAck
	TypeVoiceStatus       = protocol.TypeVoiceStatus
	TypeVoiceSpeaking     = protocol.TypeVoiceSpeaking
	TypePreferencesUpdate          = protocol.TypePreferencesUpdate
	TypeAssistantToolsRegister     = protocol.TypeAssistantToolsRegister
	TypeAssistantToolsAck          = protocol.TypeAssistantToolsAck
	TypeAssistantHeartbeat         = protocol.TypeAssistantHeartbeat
	TypeGenerationComplete         = protocol.TypeGenerationComplete
	TypeWhatsAppPairRequest        = protocol.TypeWhatsAppPairRequest
	TypeWhatsAppQR                 = protocol.TypeWhatsAppQR
	TypeWhatsAppStatus             = protocol.TypeWhatsAppStatus
	TypeWhatsAppDebug              = protocol.TypeWhatsAppDebug
)

type (
	Error              = protocol.Error
	UserMessage        = protocol.UserMessage
	AssistantMessage   = protocol.AssistantMessage
	AssistantSentence  = protocol.AssistantSentence
	StartAnswer        = protocol.StartAnswer
	ToolUseRequest     = protocol.ToolUseRequest
	ToolUseResult      = protocol.ToolUseResult
	MemoryTrace        = protocol.MemoryTrace
	ThinkingSummary    = protocol.ThinkingSummary
	TitleUpdate        = protocol.TitleUpdate
	GenerationRequest  = protocol.GenerationRequest
	Subscribe          = protocol.Subscribe
	Unsubscribe        = protocol.Unsubscribe
	SubscribeAck       = protocol.SubscribeAck
	UnsubscribeAck     = protocol.UnsubscribeAck
	Ack                = protocol.Ack
	ReasoningStep      = protocol.ReasoningStep
	SiblingInfo        = protocol.SiblingInfo
	BranchUpdate       = protocol.BranchUpdate
	VoiceJoinRequest   = protocol.VoiceJoinRequest
	VoiceJoinAck       = protocol.VoiceJoinAck
	VoiceLeaveRequest  = protocol.VoiceLeaveRequest
	VoiceLeaveAck      = protocol.VoiceLeaveAck
	VoiceStatus        = protocol.VoiceStatus
	VoiceSpeaking      = protocol.VoiceSpeaking
	PreferencesUpdate          = protocol.PreferencesUpdate
	AssistantToolsRegister     = protocol.AssistantToolsRegister
	AssistantTool              = protocol.AssistantTool
	AssistantToolsAck          = protocol.AssistantToolsAck
	GenerationComplete         = protocol.GenerationComplete
	WhatsAppPairRequest        = protocol.WhatsAppPairRequest
	WhatsAppQR                 = protocol.WhatsAppQR
	WhatsAppStatus             = protocol.WhatsAppStatus
	WhatsAppDebug              = protocol.WhatsAppDebug
)
