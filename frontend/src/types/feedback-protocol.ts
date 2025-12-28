// Feedback protocol types for Alicia
// Based on Frontend UX Enhancement Plan (docs/frontend-ux-plan.md)

export enum FeedbackEnvelopeType {
  Feedback = 20,
  FeedbackConfirmation = 21,
  UserNote = 22,
  NoteConfirmation = 23,
  MemoryAction = 24,
  MemoryConfirmation = 25,
  ServerInfo = 26,
  SessionStats = 27,
  // Note: Types 28 is reserved
  DimensionPreference = 29,
  EliteSelect = 30,
  EliteOptions = 31,
}

// Vote message sent from client to server
export interface FeedbackMessage {
  id: string;
  conversationId: string;
  messageId: string;
  targetType: 'message' | 'tool_use' | 'memory' | 'reasoning';
  targetId: string;
  vote: 'up' | 'down' | 'critical' | 'remove';
  quickFeedback?: string;
  note?: string;
  timestamp: number;
}

// Server confirmation with aggregate counts
export interface FeedbackConfirmation {
  feedbackId: string;
  targetType: string;
  targetId: string;
  aggregates: {
    upvotes: number;
    downvotes: number;
    specialVotes?: Record<string, number>;
  };
  userVote: 'up' | 'down' | 'critical' | null;
}

// Note message
export interface UserNoteMessage {
  id: string;
  messageId: string;
  content: string;
  category: 'improvement' | 'correction' | 'context' | 'general';
  action: 'create' | 'update' | 'delete';
  timestamp: number;
}

// Note confirmation
export interface NoteConfirmation {
  noteId: string;
  messageId: string;
  success: boolean;
}

// Memory action (global scope)
export interface MemoryActionMessage {
  id: string;
  action: 'create' | 'update' | 'delete' | 'pin' | 'archive';
  memory?: {
    content: string;
    category: 'preference' | 'fact' | 'context' | 'instruction';
    pinned?: boolean;
  };
  timestamp: number;
}

// Memory confirmation
export interface MemoryConfirmation {
  memoryId: string;
  action: string;
  success: boolean;
}

// Server info broadcast
export interface ServerInfoMessage {
  connection: {
    status: 'connected' | 'connecting' | 'disconnected';
    latency: number;
  };
  model: {
    name: string;
    provider: string;
  };
  mcpServers: Array<{
    name: string;
    status: 'connected' | 'disconnected' | 'error';
  }>;
}

// Session statistics
export interface SessionStatsMessage {
  messageCount: number;
  toolCallCount: number;
  memoriesUsed: number;
  sessionDuration: number;
}
