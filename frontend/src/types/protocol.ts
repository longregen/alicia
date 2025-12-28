// Protocol types for Alicia real-time binary protocol
// Based on backend protocol in pkg/protocol/

export enum MessageType {
  ErrorMessage = 1,
  UserMessage = 2,
  AssistantMessage = 3,
  AudioChunk = 4,
  ReasoningStep = 5,
  ToolUseRequest = 6,
  ToolUseResult = 7,
  Acknowledgement = 8,
  Transcription = 9,
  ControlStop = 10,
  ControlVariation = 11,
  Configuration = 12,
  StartAnswer = 13,
  MemoryTrace = 14,
  Commentary = 15,
  AssistantSentence = 16,
  // WebSocket sync message types
  SyncRequest = 17,
  SyncResponse = 18,
  // Feedback protocol message types
  Feedback = 20,
  FeedbackConfirmation = 21,
  UserNote = 22,
  NoteConfirmation = 23,
  MemoryAction = 24,
  MemoryConfirmation = 25,
  ServerInfo = 26,
  SessionStats = 27,
  // Dimension optimization message types
  DimensionPreference = 29,
  EliteSelect = 30,
  EliteOptions = 31,
}

export enum Severity {
  Info = 0,
  Warning = 1,
  Error = 2,
  Critical = 3,
}

export type StopType = 'generation' | 'speech' | 'all';
export type ToolExecution = 'server' | 'client' | 'either';
export type AnswerType = 'text' | 'voice' | 'text+voice';
export type VariationType = 'regenerate' | 'edit' | 'continue';

// Envelope wraps all protocol messages
export interface Envelope {
  stanzaId: number;
  conversationId: string;
  type: MessageType;
  meta?: Record<string, unknown>;
  body: unknown;
}

// Message type definitions

export interface ErrorMessage {
  id: string;
  conversationId: string;
  code: number;
  message: string;
  severity: Severity;
  recoverable: boolean;
  originatingId?: string;
}

export interface UserMessage {
  id: string;
  previousId?: string;
  conversationId: string;
  content: string;
  timestamp?: number;
}

export interface AssistantMessage {
  id: string;
  previousId?: string;
  conversationId: string;
  content: string;
  timestamp?: number;
}

export interface AudioChunk {
  conversationId: string;
  format: string;
  sequence: number;
  durationMs: number;
  trackSid?: string;
  data?: Uint8Array;
  isLast?: boolean;
  timestamp?: number;
}

export interface ReasoningStep {
  id: string;
  messageId: string;
  conversationId: string;
  sequence: number;
  content: string;
}

export interface ToolUseRequest {
  id: string;
  messageId: string;
  conversationId: string;
  toolName: string;
  parameters: Record<string, unknown>;
  execution: ToolExecution;
  timeoutMs?: number;
}

export interface ToolUseResult {
  id: string;
  requestId: string;
  conversationId: string;
  success: boolean;
  result?: unknown;
  errorCode?: string;
  errorMessage?: string;
}

export interface Acknowledgement {
  conversationId: string;
  acknowledgedStanzaId: number;
  success: boolean;
}

export interface Transcription {
  id: string;
  previousId?: string;
  conversationId: string;
  text: string;
  final: boolean;
  confidence?: number;
  language?: string;
}

export interface ControlStop {
  conversationId: string;
  targetId?: string;
  reason?: string;
  stopType?: StopType;
}

export interface ControlVariation {
  conversationId: string;
  targetId: string;
  mode: VariationType;
  newContent?: string;
}

export interface Configuration {
  conversationId?: string;
  lastSequenceSeen?: number;
  clientVersion?: string;
  preferredLanguage?: string;
  device?: string;
  features?: string[];
}

export interface StartAnswer {
  id: string;
  previousId: string;
  conversationId: string;
  answerType?: AnswerType;
  plannedSentenceCount?: number;
}

export interface MemoryTrace {
  id: string;
  messageId: string;
  conversationId: string;
  memoryId: string;
  content: string;
  relevance: number;
}

export interface Commentary {
  id: string;
  messageId: string;
  conversationId: string;
  content: string;
  commentaryType?: string;
}

export interface AssistantSentence {
  id?: string;
  previousId: string;
  conversationId: string;
  sequence: number;
  text: string;
  isFinal?: boolean;
  audio?: Uint8Array;
}

export type FeedbackTargetType = 'message' | 'tool_use' | 'memory' | 'reasoning';
export type VoteType = 'up' | 'down' | 'critical' | 'remove';

export interface Feedback {
  id: string;
  conversationId: string;
  messageId: string;
  targetType: FeedbackTargetType;
  targetId: string;
  vote: VoteType;
  quickFeedback?: string;
  note?: string;
  timestamp: number;
}

export interface FeedbackAggregates {
  upvotes: number;
  downvotes: number;
  specialVotes?: Record<string, number>;
}

export interface FeedbackConfirmation {
  feedbackId: string;
  targetType: FeedbackTargetType;
  targetId: string;
  aggregates: FeedbackAggregates;
  userVote: 'up' | 'down' | 'critical' | null;
}

export type NoteCategory = 'improvement' | 'correction' | 'context' | 'general';
export type NoteAction = 'create' | 'update' | 'delete';

export interface UserNote {
  id: string;
  messageId: string;
  content: string;
  category: NoteCategory;
  action: NoteAction;
  timestamp: number;
}

export interface NoteConfirmation {
  noteId: string;
  messageId: string;
  success: boolean;
}

export type MemoryCategory = 'preference' | 'fact' | 'context' | 'instruction';
export type MemoryActionType = 'create' | 'update' | 'delete' | 'pin' | 'archive';

export interface MemoryData {
  content: string;
  category: MemoryCategory;
  pinned?: boolean;
}

export interface MemoryAction {
  id: string;
  action: MemoryActionType;
  memory?: MemoryData;
  timestamp: number;
}

export interface MemoryConfirmation {
  memoryId: string;
  action: MemoryActionType;
  success: boolean;
}

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'reconnecting';

export interface ConnectionInfo {
  status: ConnectionStatus;
  latency: number;
}

export interface ModelInfo {
  name: string;
  provider: string;
}

export type MCPServerStatus = 'connected' | 'disconnected' | 'error';

export interface MCPServerInfo {
  name: string;
  status: MCPServerStatus;
}

export interface ServerInfo {
  connection: ConnectionInfo;
  model: ModelInfo;
  mcpServers: MCPServerInfo[];
}

export interface SessionStats {
  messageCount: number;
  toolCallCount: number;
  memoriesUsed: number;
  sessionDuration: number;
}

// Dimension optimization types (Types 29-31)

export interface DimensionWeights {
  successRate: number;
  quality: number;
  efficiency: number;
  robustness: number;
  generalization: number;
  diversity: number;
  innovation: number;
}

export interface DimensionScores {
  successRate: number;
  quality: number;
  efficiency: number;
  robustness: number;
  generalization: number;
  diversity: number;
  innovation: number;
}

export interface DimensionPreference {
  conversationId: string;
  weights: DimensionWeights;
  preset?: 'accuracy' | 'speed' | 'reliable' | 'creative' | 'balanced';
  timestamp: number;
}

export interface EliteSelect {
  conversationId: string;
  eliteId: string;
  timestamp: number;
}

export interface EliteSummary {
  id: string;
  label: string;
  scores: DimensionScores;
  description: string;
  bestFor: string;
}

export interface EliteOptions {
  conversationId: string;
  elites: EliteSummary[];
  currentEliteId: string;
  timestamp: number;
}

// Common features
export const Features = {
  STREAMING: 'streaming',
  PARTIAL_RESPONSES: 'partial_responses',
  AUDIO_OUTPUT: 'audio_output',
  REASONING_STEPS: 'reasoning_steps',
  TOOL_USE: 'tool_use',
} as const;
