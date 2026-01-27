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
import { getUserId } from '../utils/deviceId';

const API_BASE = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL}/api/v1`
  : '/api/v1';

async function fetchWithErrorHandling(url: string, options?: RequestInit): Promise<Response> {
  try {
    const headers = new Headers(options?.headers);
    headers.set('X-User-ID', getUserId());

    return await fetch(url, {
      ...options,
      headers,
    });
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

async function handleVoidResponse(response: Response): Promise<void> {
  if (!response.ok) {
    const errorMessage = await response.text();
    throw new Error(errorMessage || `Request failed: ${response.status}`);
  }
}

export interface UserPreferencesResponse {
  user_id: string;
  theme: 'light' | 'dark' | 'system';
  audio_output_enabled: boolean;
  voice_speed: number;
  memory_min_importance: number;
  memory_min_historical: number;
  memory_min_personal: number;
  memory_min_factual: number;
  memory_retrieval_count: number;
  max_tokens: number;
  max_tool_iterations: number;
  temperature: number;
  pareto_target_score: number;
  pareto_max_generations: number;
  pareto_branches_per_gen: number;
  pareto_archive_size: number;
  pareto_enable_crossover: boolean;
  notes_similarity_threshold: number;
  notes_max_count: number;
  confirm_delete_memory: boolean;
  show_relevance_scores: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdatePreferencesRequest {
  theme?: 'light' | 'dark' | 'system';
  audio_output_enabled?: boolean;
  voice_speed?: number;
  memory_min_importance?: number;
  memory_min_historical?: number;
  memory_min_personal?: number;
  memory_min_factual?: number;
  memory_retrieval_count?: number;
  max_tokens?: number;
  pareto_target_score?: number;
  pareto_max_generations?: number;
  pareto_branches_per_gen?: number;
  pareto_archive_size?: number;
  pareto_enable_crossover?: boolean;
  notes_similarity_threshold?: number;
  notes_max_count?: number;
  confirm_delete_memory?: boolean;
  show_relevance_scores?: boolean;
}

export const api = {
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
    await handleVoidResponse(response);
  },

  async updateConversation(id: string, data: Partial<Conversation>): Promise<Conversation> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Conversation>(response);
  },

  async getMessages(conversationId: string, all?: boolean): Promise<Message[]> {
    const url = all
      ? `${API_BASE}/conversations/${conversationId}/messages?all=true`
      : `${API_BASE}/conversations/${conversationId}/messages`;
    const response = await fetchWithErrorHandling(url);
    const data = await handleResponse<MessagesResponse>(response);
    return data.messages || [];
  },

  async getMessage(messageId: string): Promise<Message> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}`);
    return handleResponse<Message>(response);
  },

  async getMessageSiblings(messageId: string): Promise<Message[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/siblings`);
    const data = await handleResponse<{ siblings: Message[] }>(response);
    return data.siblings || [];
  },

  async sendMessage(conversationId: string, data: CreateMessageRequest): Promise<Message> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Message>(response);
  },

  async listToolUses(limit = 50, offset = 0): Promise<ToolUsesResponse> {
    const response = await fetchWithErrorHandling(
      `${API_BASE}/tool-uses?limit=${limit}&offset=${offset}`
    );
    return handleResponse<ToolUsesResponse>(response);
  },

  async getToolUse(id: string): Promise<ToolUse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${id}`);
    return handleResponse<ToolUse>(response);
  },

  async getToolUsesByMessage(messageId: string): Promise<ToolUse[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/tool-uses`);
    const data = await handleResponse<{ tool_uses: ToolUse[] }>(response);
    return data.tool_uses || [];
  },

  async getMemoryUsesByMessage(messageId: string): Promise<MemoryUse[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/memory-uses`);
    const data = await handleResponse<{ memory_uses: MemoryUse[] }>(response);
    return data.memory_uses || [];
  },

  async createMessageFeedback(messageId: string, rating: FeedbackRating, note?: string): Promise<MessageFeedback> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/feedback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rating, note: note || '' }),
    });
    return handleResponse<MessageFeedback>(response);
  },

  async getMessageFeedback(messageId: string): Promise<MessageFeedback | null> {
    const response = await fetchWithErrorHandling(`${API_BASE}/messages/${messageId}/feedback`);
    return handleResponse<MessageFeedback | null>(response);
  },

  async createToolUseFeedback(toolUseId: string, rating: FeedbackRating, note?: string): Promise<ToolUseFeedback> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/feedback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rating, note: note || '' }),
    });
    return handleResponse<ToolUseFeedback>(response);
  },

  async getToolUseFeedback(toolUseId: string): Promise<ToolUseFeedback | null> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tool-uses/${toolUseId}/feedback`);
    return handleResponse<ToolUseFeedback | null>(response);
  },

  async createMemoryUseFeedback(memoryUseId: string, rating: FeedbackRating, note?: string): Promise<MemoryUseFeedback> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memory-uses/${memoryUseId}/feedback`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rating, note: note || '' }),
    });
    return handleResponse<MemoryUseFeedback>(response);
  },

  async getMemoryUseFeedback(memoryUseId: string): Promise<MemoryUseFeedback | null> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memory-uses/${memoryUseId}/feedback`);
    return handleResponse<MemoryUseFeedback | null>(response);
  },

  async getLiveKitToken(conversationId: string, participantName?: string): Promise<string> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        participant_name: participantName || 'Web User',
      }),
    });
    const data = await handleResponse<{ token: string }>(response);
    return data.token;
  },

  async createLiveKitRoom(conversationId: string): Promise<{ room_name: string }> {
    const response = await fetchWithErrorHandling(`${API_BASE}/conversations/${conversationId}/room`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
    return handleResponse<{ room_name: string }>(response);
  },

  async getMCPServers(): Promise<MCPServer[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers`);
    const data = await handleResponse<MCPServersResponse>(response);
    return data.servers || [];
  },

  async getMCPServer(name: string): Promise<MCPServer> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers/${encodeURIComponent(name)}`);
    return handleResponse<MCPServer>(response);
  },

  async addMCPServer(server: MCPServerConfig): Promise<MCPServer> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(server),
    });
    return handleResponse<MCPServer>(response);
  },

  async updateMCPServer(name: string, updates: Partial<MCPServerConfig>): Promise<MCPServer> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers/${encodeURIComponent(name)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates),
    });
    return handleResponse<MCPServer>(response);
  },

  async removeMCPServer(name: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/mcp/servers/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
    await handleVoidResponse(response);
  },

  async getMCPTools(): Promise<Record<string, MCPTool[]>> {
    const response = await fetchWithErrorHandling(`${API_BASE}/tools`);
    const data = await handleResponse<MCPToolsResponse>(response);
    return data.tools || {};
  },

  async createMemory(content: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
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
    await handleVoidResponse(response);
  },

  async addMemoryTag(memoryId: string, tag: string): Promise<MemoryResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tag }),
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
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/${memoryId}`, {
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

  async getMemoriesByTag(tag: string): Promise<MemoryListResponse> {
    const params = new URLSearchParams();
    params.append('tag', tag);
    const response = await fetchWithErrorHandling(`${API_BASE}/memories/by-tags?${params.toString()}`);
    return handleResponse<MemoryListResponse>(response);
  },

  async createNote(data: { id?: string; title: string; content: string }): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<NoteResponse>(response);
  },

  async listNotes(): Promise<NoteResponse[]> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes`);
    const data = await handleResponse<{ notes: NoteResponse[] }>(response);
    return data.notes || [];
  },

  async getNote(id: string): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes/${id}`);
    return handleResponse<NoteResponse>(response);
  },

  async updateNote(id: string, data: { title?: string; content?: string }): Promise<NoteResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<NoteResponse>(response);
  },

  async deleteNote(id: string): Promise<void> {
    const response = await fetchWithErrorHandling(`${API_BASE}/notes/${id}`, {
      method: 'DELETE',
    });
    await handleVoidResponse(response);
  },

  async getPreferences(): Promise<UserPreferencesResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/preferences`);
    return handleResponse<UserPreferencesResponse>(response);
  },

  async updatePreferences(data: UpdatePreferencesRequest): Promise<UserPreferencesResponse> {
    const response = await fetchWithErrorHandling(`${API_BASE}/preferences`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<UserPreferencesResponse>(response);
  },
};

export type FeedbackRating = -1 | 0 | 1;

export interface ToolUse {
  id: string;
  message_id: string;
  tool_name: string;
  arguments: Record<string, unknown>;
  result?: unknown;
  status: 'pending' | 'success' | 'error';
  error?: string;
  created_at: string;
}

export interface ToolUsesResponse {
  tool_uses: ToolUse[];
  total: number;
  limit: number;
  offset: number;
}

export interface MessageFeedback {
  id: string;
  message_id: string;
  rating: FeedbackRating;
  note: string;
  created_at: string;
}

export interface ToolUseFeedback {
  id: string;
  tool_use_id: string;
  rating: FeedbackRating;
  note: string;
  created_at: string;
}

export interface MemoryUseFeedback {
  id: string;
  memory_use_id: string;
  rating: FeedbackRating;
  note: string;
  created_at: string;
}

export interface MemoryUse {
  id: string;
  memory_id: string;
  message_id: string;
  content: string;
  similarity: number;
  created_at: string;
}

export interface MemoryResponse {
  id: string;
  content: string;
  importance: number;
  tags: string[];
  pinned: boolean;
  archived: boolean;
  source_message_id?: string;
  created_at: string;
  updated_at: string;
}

export interface MemoryListResponse {
  memories: MemoryResponse[];
  total: number;
}

export interface NoteResponse {
  id: string;
  user_id: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}
