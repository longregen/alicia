export enum MessageType {
  ErrorMessage = 1,
  UserMessage = 2,
  AssistantMessage = 3,
  ReasoningStep = 5,
  ToolUseRequest = 6,
  ToolUseResult = 7,
  Acknowledgement = 8,
  StartAnswer = 13,
  MemoryTrace = 14,
  AssistantSentence = 16,
  GenerationRequest = 33,
  ThinkingSummary = 34,
  ConversationTitleUpdate = 35,
  Subscribe = 40,
  Unsubscribe = 41,
  SubscribeAck = 42,
  UnsubscribeAck = 43,
  BranchUpdate = 50,
  VoiceJoinRequest = 51,
  VoiceJoinAck = 52,
  VoiceLeaveRequest = 53,
  VoiceLeaveAck = 54,
  VoiceStatus = 55,
  VoiceSpeaking = 56,
  GenerationComplete = 80,
}

export interface Envelope {
  conversationId: string;
  type: MessageType;
  meta?: Record<string, unknown>;
  body: unknown;
  // Trace context (W3C Trace Context compatible)
  trace_id?: string;
  span_id?: string;
  trace_flags?: number;
  session_id?: string;
  user_id?: string;
}

export interface ErrorMessage {
  code: string;
  message: string;
  messageId?: string;
  conversationId?: string;
}

export interface UserMessage {
  id: string;
  conversationId: string;
  content: string;
  previousId?: string;
}

export interface AssistantMessage {
  id: string;
  conversationId: string;
  content: string;
  previousId?: string;
  reasoning?: string;
}

export interface ToolUseRequest {
  id: string;
  messageId: string;
  conversationId: string;
  toolName: string;
  arguments: Record<string, unknown>;
}

export interface ToolUseResult {
  id: string;
  requestId: string;
  messageId?: string;
  conversationId: string;
  success: boolean;
  result?: unknown;
  error?: string;
}

export type Acknowledgement = Record<string, never>;

export interface StartAnswer {
  messageId: string;
  conversationId: string;
  previousId?: string;
}

export interface MemoryTrace {
  id: string;
  memoryId: string;
  messageId: string;
  conversationId: string;
  content: string;
  relevance: number;
}

export interface AssistantSentence {
  id: string;
  messageId: string;
  conversationId: string;
  sequence: number;
  text: string;
  isFinal: boolean;
}

export interface SubscribeRequest {
  conversationId?: string;
  agentMode?: boolean;
}

export interface UnsubscribeRequest {
  conversationId: string;
}

export interface SubscribeAck {
  conversationId?: string;
  agentMode?: boolean;
  success: boolean;
  error?: string;
}

export interface UnsubscribeAck {
  conversationId: string;
  success: boolean;
}

export interface ReasoningStep {
  id: string;
  messageId: string;
  conversationId: string;
  sequence: number;
  content: string;
}

export interface ThinkingSummary {
  id: string;
  messageId: string;
  conversationId: string;
  content: string;
  progress?: number; // 0-100 percentage
}

export interface ConversationTitleUpdate {
  conversationId: string;
  title: string;
}

export interface SiblingInfo {
  id: string;
  content: string;
  createdAt: string;
}

export interface BranchUpdate {
  conversationId: string;
  parentMessageId: string;
  newSibling: SiblingInfo;
  allSiblings: SiblingInfo[];
  totalCount: number;
}

export interface VoiceJoinRequest {
  conversationId: string;
  userId: string;
}

export interface VoiceJoinAck {
  conversationId: string;
  success: boolean;
  error?: string;
  sampleRate?: number;
}

export interface VoiceLeaveRequest {
  conversationId: string;
}

export interface VoiceLeaveAck {
  conversationId: string;
  success: boolean;
  error?: string;
}

export interface VoiceSpeaking {
  conversationId: string;
  messageId: string;
  speaking: boolean;
  sentenceSeq?: number;
}

export interface VoiceStatus {
  conversationId: string;
  status: 'queue_full' | 'queue_ok' | 'speaking' | 'idle';
  queueLength: number;
  error?: string;
}

export interface GenerationComplete {
  messageId: string;
  conversationId: string;
  success: boolean;
  error?: string;
}
