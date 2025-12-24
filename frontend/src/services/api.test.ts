import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { api } from './api';

describe('api service', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    (global as any).fetch = mockFetch;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('createConversation', () => {
    it('should create a conversation successfully', async () => {
      const mockConversation = {
        id: 'conv-123',
        title: 'Test Conversation',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockConversation,
      });

      const result = await api.createConversation({ title: 'Test Conversation' });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'Test Conversation' }),
      });
      expect(result).toEqual(mockConversation);
    });

    it('should handle errors when creating conversation', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        text: async () => 'Bad request',
      });

      await expect(api.createConversation({ title: 'Test' })).rejects.toThrow('Bad request');
    });
  });

  describe('getConversations', () => {
    it('should fetch conversations successfully', async () => {
      const mockConversations = [
        {
          id: 'conv-1',
          title: 'Conversation 1',
          status: 'active',
          last_client_stanza_id: 0,
          last_server_stanza_id: 0,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'conv-2',
          title: 'Conversation 2',
          status: 'active',
          last_client_stanza_id: 0,
          last_server_stanza_id: 0,
          created_at: '2024-01-02T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
        },
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ conversations: mockConversations }),
      });

      const result = await api.getConversations();

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations', undefined);
      expect(result).toEqual(mockConversations);
    });

    it('should return empty array when conversations is undefined', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

      const result = await api.getConversations();

      expect(result).toEqual([]);
    });

    it('should handle fetch errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        text: async () => 'Internal server error',
      });

      await expect(api.getConversations()).rejects.toThrow('Internal server error');
    });
  });

  describe('getConversation', () => {
    it('should fetch a single conversation', async () => {
      const mockConversation = {
        id: 'conv-123',
        title: 'Test Conversation',
        status: 'active',
        last_client_stanza_id: 5,
        last_server_stanza_id: 10,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockConversation,
      });

      const result = await api.getConversation('conv-123');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123', undefined);
      expect(result).toEqual(mockConversation);
    });

    it('should handle not found error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        text: async () => 'Conversation not found',
      });

      await expect(api.getConversation('conv-999')).rejects.toThrow('Conversation not found');
    });
  });

  describe('deleteConversation', () => {
    it('should delete a conversation successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
      });

      await api.deleteConversation('conv-123');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123', {
        method: 'DELETE',
      });
    });

    it('should handle delete errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        text: async () => 'Not found',
      });

      await expect(api.deleteConversation('conv-999')).rejects.toThrow('Not found');
    });
  });

  describe('updateConversation', () => {
    it('should update a conversation successfully', async () => {
      const updatedConversation = {
        id: 'conv-123',
        title: 'Updated Title',
        status: 'active',
        last_client_stanza_id: 0,
        last_server_stanza_id: 0,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => updatedConversation,
      });

      const result = await api.updateConversation('conv-123', {
        title: 'Updated Title',
      });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'Updated Title' }),
      });
      expect(result).toEqual(updatedConversation);
    });
  });

  describe('getMessages', () => {
    it('should fetch messages for a conversation', async () => {
      const mockMessages = [
        {
          id: 'msg-1',
          conversation_id: 'conv-123',
          sequence_number: 1,
          role: 'user',
          contents: 'Hello',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 'msg-2',
          conversation_id: 'conv-123',
          sequence_number: 2,
          role: 'assistant',
          contents: 'Hi there!',
          created_at: '2024-01-01T00:01:00Z',
          updated_at: '2024-01-01T00:01:00Z',
        },
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ messages: mockMessages }),
      });

      const result = await api.getMessages('conv-123');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123/messages', undefined);
      expect(result).toEqual(mockMessages);
    });

    it('should return empty array when messages is undefined', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

      const result = await api.getMessages('conv-123');

      expect(result).toEqual([]);
    });
  });

  describe('sendMessage', () => {
    it('should send a message successfully', async () => {
      const mockMessage = {
        id: 'msg-123',
        conversation_id: 'conv-123',
        sequence_number: 1,
        role: 'user',
        contents: 'Hello, assistant!',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockMessage,
      });

      const result = await api.sendMessage('conv-123', {
        contents: 'Hello, assistant!',
      });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contents: 'Hello, assistant!' }),
      });
      expect(result).toEqual(mockMessage);
    });
  });

  describe('getLiveKitToken', () => {
    it('should get a LiveKit token successfully', async () => {
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ token: mockToken }),
      });

      const result = await api.getLiveKitToken('conv-123', 'Test User');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/conversations/conv-123/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: expect.stringContaining('Test User'),
      });

      // Verify the participant_id is generated
      const callBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(callBody.participant_id).toMatch(/^user_device_\d+_[a-z0-9]+$/);
      expect(callBody.participant_name).toBe('Test User');

      expect(result).toBe(mockToken);
    });

    it('should use default participant name if not provided', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ token: 'token-123' }),
      });

      await api.getLiveKitToken('conv-123');

      const callBody = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(callBody.participant_name).toBe('Web User');
    });

    it('should handle token generation errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        text: async () => 'Failed to generate token',
      });

      await expect(api.getLiveKitToken('conv-123')).rejects.toThrow(
        'Failed to generate token'
      );
    });
  });

  describe('error handling', () => {
    it('should throw error with status when response text is empty', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 503,
        text: async () => '',
      });

      await expect(api.getConversations()).rejects.toThrow('HTTP error! status: 503');
    });

    it('should prioritize response text over status in error message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        text: async () => 'Validation error: title is required',
      });

      await expect(api.createConversation({})).rejects.toThrow(
        'Validation error: title is required'
      );
    });
  });
});
