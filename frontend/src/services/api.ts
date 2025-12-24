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
  } catch (err) {
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
};
