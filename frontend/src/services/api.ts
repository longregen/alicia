import {
  Conversation,
  Message,
  CreateConversationRequest,
  CreateMessageRequest,
  ConversationsResponse,
  MessagesResponse,
} from '../types/models';
import {
  MCPServer,
  MCPServerConfig,
  MCPTool,
  MCPServersResponse,
  MCPToolsResponse,
} from '../types/mcp';
import {
  SyncRequest,
  SyncResponse,
  SyncStatusResponse,
} from '../types/sync';
import { getDeviceId } from '../utils/deviceId';

const API_BASE = '/api/v1';

async function fetchWithErrorHandling(url: string, options?: RequestInit): Promise<Response> {
  try {
    return await fetch(url, options);
  } catch (err) {
    if (err instanceof TypeError && err.message.includes('fetch')) {
      throw new Error('Network error: Unable to connect to the server. Please check your connection.');
    }
    throw err;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const text = await response.text();
    const errorMessage = text || `HTTP error! status: ${response.status}`;
    throw new Error(errorMessage);
  }

  try {
    return await response.json();
  } catch {
    throw new Error('Failed to parse response: invalid JSON');
  }
}

export const api = {
  // Conversations
  async createConversation(data: CreateConversationRequest): Promise<Conversation> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Conversation>(response);
  },

  async getConversations(): Promise<Conversation[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations`);
    const data = await handleResponse<ConversationsResponse>(response);
    return data.conversations || [];
  },

  async getConversation(id: string): Promise<Conversation> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${id}`);
    return handleResponse<Conversation>(response);
  },

  async deleteConversation(id: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || `Failed to delete conversation: ${response.status}`);
    }
  },

  async updateConversation(id: string, data: Partial<Conversation>): Promise<Conversation> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Conversation>(response);
  },

  // Messages
  async getMessages(conversationId: string): Promise<Message[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/messages`);
    const data = await handleResponse<MessagesResponse>(response);
    return data.messages || [];
  },

  async sendMessage(conversationId: string, data: CreateMessageRequest): Promise<Message> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Message>(response);
  },

  // LiveKit
  async getLiveKitToken(conversationId: string, participantName?: string): Promise<string> {
    // Use persistent device ID instead of random ID
    const deviceId = getDeviceId();
    const participantId = `user_${deviceId}`;

    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        participant_id: participantId,
        participant_name: participantName || 'Web User',
      }),
    });
    const data = await handleResponse<{ token: string }>(response);
    return data.token;
  },

  // MCP Server management
  async getMCPServers(): Promise<MCPServer[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers`);
    const data = await handleResponse<MCPServersResponse>(response);
    return data.servers || [];
  },

  async addMCPServer(server: MCPServerConfig): Promise<MCPServer> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(server),
    });
    return handleResponse<MCPServer>(response);
  },

  async removeMCPServer(name: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || `Failed to delete MCP server: ${response.status}`);
    }
  },

  async getMCPTools(): Promise<MCPTool[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/tools`);
    const data = await handleResponse<MCPToolsResponse>(response);
    return Object.values(data.tools || {}).flat();
  },

  // Sync
  async syncConversation(conversationId: string, request: SyncRequest): Promise<SyncResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/sync`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    return handleResponse<SyncResponse>(response);
  },

  async getSyncStatus(conversationId: string): Promise<SyncStatusResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/sync/status`);
    return handleResponse<SyncStatusResponse>(response);
  },

  // Config
  async getConfig(): Promise<PublicConfig> {
    const response = await fetchWithErrorHandling(`${API_BASE}/config`);
    return handleResponse<PublicConfig>(response);
  },

  // Voting
  async voteOnMessage(messageId: string, vote: 'up' | 'down', quickFeedback?: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vote, quick_feedback: quickFeedback }),
    });
    return handleResponse<VoteResponse>(response);
  },

  async removeMessageVote(messageId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/vote`, {
      method: 'DELETE',
    });
    return handleResponse<VoteResponse>(response);
  },

  async getMessageVotes(messageId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/votes`);
    return handleResponse<VoteResponse>(response);
  },

  async getToolUseVotes(toolUseId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/votes`);
    return handleResponse<VoteResponse>(response);
  },

  async voteOnToolUse(toolUseId: string, vote: 'up' | 'down', quickFeedback?: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vote, quick_feedback: quickFeedback }),
    });
    return handleResponse<VoteResponse>(response);
  },

  async removeToolUseVote(toolUseId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/vote`, {
      method: 'DELETE',
    });
    return handleResponse<VoteResponse>(response);
  },

  async submitToolUseQuickFeedback(toolUseId: string, feedback: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/quick-feedback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ feedback }),
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || `Failed to submit quick feedback: ${response.status}`);
    }
  },

  async voteOnMemory(memoryId: string, vote: 'up' | 'down' | 'critical'): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vote }),
    });
    return handleResponse<VoteResponse>(response);
  },

  async removeMemoryVote(memoryId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/vote`, {
      method: 'DELETE',
    });
    return handleResponse<VoteResponse>(response);
  },

  async getMemoryVotes(memoryId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/votes`);
    return handleResponse<VoteResponse>(response);
  },

  // Memory CRUD operations
  async createMemory(content: string, category: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, category }),
    });
    return handleResponse<MemoryResponse>(response);
  },

  async listMemories(): Promise<MemoryListResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories`);
    return handleResponse<MemoryListResponse>(response);
  },

  async getMemory(memoryId: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}`);
    return handleResponse<MemoryResponse>(response);
  },

  async updateMemory(memoryId: string, content: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    });
    return handleResponse<MemoryResponse>(response);
  },

  async deleteMemory(memoryId: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || `Failed to delete memory: ${response.status}`);
    }
  },

  async addMemoryTags(memoryId: string, tags: string[]): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tags }),
    });
    return handleResponse<MemoryResponse>(response);
  },

  async removeMemoryTag(memoryId: string, tag: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/tags/${encodeURIComponent(tag)}`, {
      method: 'DELETE',
    });
    return handleResponse<MemoryResponse>(response);
  },

  async pinMemory(memoryId: string, pinned: boolean): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/pin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pinned }),
    });
    return handleResponse<MemoryResponse>(response);
  },

  async archiveMemory(memoryId: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/archive`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
    return handleResponse<MemoryResponse>(response);
  },

  async setMemoryImportance(memoryId: string, importance: number): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/importance`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ importance }),
    });
    return handleResponse<MemoryResponse>(response);
  },

  async searchMemories(query: string, limit?: number): Promise<MemoryListResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query, limit: limit || 10 }),
    });
    return handleResponse<MemoryListResponse>(response);
  },

  async getMemoriesByTags(tags: string[]): Promise<MemoryListResponse> {
    const params = new URLSearchParams();
    tags.forEach(tag => params.append('tags', tag));
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/by-tags?${params.toString()}`);
    return handleResponse<MemoryListResponse>(response);
  },

  async voteOnReasoning(reasoningId: string, vote: 'up' | 'down'): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/reasoning/${reasoningId}/vote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vote }),
    });
    return handleResponse<VoteResponse>(response);
  },

  async removeReasoningVote(reasoningId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/reasoning/${reasoningId}/vote`, {
      method: 'DELETE',
    });
    return handleResponse<VoteResponse>(response);
  },

  async getReasoningVotes(reasoningId: string): Promise<VoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/reasoning/${reasoningId}/votes`);
    return handleResponse<VoteResponse>(response);
  },

  // Notes
  async createMessageNote(messageId: string, content: string, category?: string): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, category: category || 'general' }),
    });
    return handleResponse<NoteResponse>(response);
  },

  async getMessageNotes(messageId: string): Promise<NoteListResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/notes`);
    return handleResponse<NoteListResponse>(response);
  },

  async createToolUseNote(toolUseId: string, content: string, category?: string): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, category: category || 'general' }),
    });
    return handleResponse<NoteResponse>(response);
  },

  async createReasoningNote(reasoningId: string, content: string, category?: string): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/reasoning/${reasoningId}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content, category: category || 'general' }),
    });
    return handleResponse<NoteResponse>(response);
  },

  async updateNote(noteId: string, content: string): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes/${noteId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    });
    return handleResponse<NoteResponse>(response);
  },

  async deleteNote(noteId: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes/${noteId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const errorMessage = await response.text();
      throw new Error(errorMessage || `Failed to delete note: ${response.status}`);
    }
  },

  // Server Info
  async getServerInfo(): Promise<ServerInfoResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/server/info`);
    return handleResponse<ServerInfoResponse>(response);
  },

  async getGlobalStats(): Promise<SessionStatsResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/server/stats`);
    return handleResponse<SessionStatsResponse>(response);
  },

  async getConversationStats(conversationId: string): Promise<SessionStatsResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/stats`);
    return handleResponse<SessionStatsResponse>(response);
  },
};

// Feedback types
export interface VoteResponse {
  target_id: string;
  target_type: string;
  upvotes: number;
  downvotes: number;
  user_vote: string | null;
  special?: Record<string, number>;
}

export interface NoteResponse {
  id: string;
  message_id?: string;
  target_id: string;
  target_type: string;
  content: string;
  category: string;
  created_at: number;
  updated_at: number;
}

export interface NoteListResponse {
  notes: NoteResponse[];
  total: number;
}

export interface Voice {
  id: string;
  name: string;
  category: string;
}

export interface TTSConfig {
  endpoint: string;
  model: string;
  default_voice: string;
  default_speed: number;
  speed_min: number;
  speed_max: number;
  speed_step: number;
  voices: Voice[];
}

export interface PublicConfig {
  livekit_url?: string;
  tts_enabled: boolean;
  asr_enabled: boolean;
  tts?: TTSConfig;
}

// Server info types
export interface ConnectionInfoResponse {
  status: string;
  latency: number;
}

export interface ModelInfoResponse {
  name: string;
  provider: string;
}

export interface MCPServerInfoResponse {
  name: string;
  status: string;
}

export interface ServerInfoResponse {
  connection: ConnectionInfoResponse;
  model: ModelInfoResponse;
  mcpServers: MCPServerInfoResponse[];
}

export interface SessionStatsResponse {
  messageCount: number;
  toolCallCount: number;
  memoriesUsed: number;
  sessionDuration: number;
  conversationId?: string;
}

// Memory types
export interface MemoryResponse {
  id: string;
  content: string;
  category: 'preference' | 'fact' | 'context' | 'instruction';
  importance: number;
  tags: string[];
  pinned: boolean;
  archived: boolean;
  createdAt: number;
  updatedAt: number;
}

export interface MemoryListResponse {
  memories: MemoryResponse[];
  total: number;
}
