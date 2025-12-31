/**
 * Barrel export file for commonly used types.
 * Import from here for convenience: import { MessageRole, SyncStatus } from '@/types'
 */

// Core domain models
export type { Message, MessageRole, SyncStatus, Conversation, ConversationStatus } from './models';

// Streaming/normalized store types
export type {
  NormalizedMessage,
  MessageId,
  SentenceId,
  ToolCallId,
  AudioRefId,
  MemoryTraceId,
  ConversationId,
  MessageSentence,
  ToolCall,
  AudioRef,
  MemoryTrace as StreamingMemoryTrace,
  ConversationStoreState,
} from './streaming';
export { MessageStatus, MicrophoneStatus } from './streaming';

// Protocol types
export type {
  Envelope,
  ErrorMessage,
  UserMessage,
  AssistantMessage,
  AudioChunk,
  ReasoningStep,
  ToolUseRequest,
  ToolUseResult,
  Acknowledgement,
  Transcription,
  ControlStop,
  ControlVariation,
  Configuration,
  StartAnswer,
  MemoryTrace,
  Commentary,
  AssistantSentence,
  Feedback,
  FeedbackConfirmation,
  UserNote,
  NoteConfirmation,
  MemoryAction,
  MemoryConfirmation,
  ServerInfo,
  SessionStats,
  DimensionPreference,
  EliteSelect,
  EliteOptions,
  ConnectionStatus,
  MCPServerStatus,
} from './protocol';
export { MessageType, Severity } from './protocol';

// Sync types
export type {
  SyncMessageRequest,
  SyncRequest,
  SyncResponse,
  SyncStatusResponse,
  MessageResponse,
  SyncState,
} from './sync';

// Component types
export type {
  MessageData,
  MessageMetadata,
  ToolData,
  MessageAddon,
  LanguageData,
  Size,
  Variant,
  RecordingState,
  MessageState,
  AudioState,
  ConnectionState,
} from './components';

// MCP types
export type { MCPServerConfig, MCPTool, MCPServer, MCPServersResponse, MCPToolsResponse } from './mcp';
