import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { messageRepository, conversationRepository } from './repository';
import { Message, Conversation } from '../types/models';
import * as sqlite from './sqlite';

// Mock the sqlite module
vi.mock('./sqlite', () => ({
  getDatabase: vi.fn(),
  scheduleSave: vi.fn(),
}));

describe('repository', () => {
  let mockDb: {
    exec: ReturnType<typeof vi.fn>;
    run: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    vi.clearAllMocks();

    mockDb = {
      exec: vi.fn(() => []),
      run: vi.fn(),
    };

    (sqlite.getDatabase as any).mockReturnValue(mockDb);
  });

  describe('messageRepository', () => {
    const mockMessage: Message = {
      id: 'msg-1',
      conversation_id: 'conv-1',
      sequence_number: 1,
      role: 'user',
      contents: 'Hello, world!',
      local_id: 'local-1',
      sync_status: 'pending',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    };

    describe('findByConversation', () => {
      it('should return messages for a conversation', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'sync_status', 'created_at', 'updated_at'],
            values: [
              ['msg-1', 'conv-1', 1, 'user', 'Hello', 'local-1', 'synced', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
              ['msg-2', 'conv-1', 2, 'assistant', 'Hi there!', null, 'synced', '2024-01-01T00:01:00Z', '2024-01-01T00:01:00Z'],
            ],
          },
        ]);

        const messages = messageRepository.findByConversation('conv-1');

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('SELECT'),
          ['conv-1']
        );

        expect(messages).toHaveLength(2);
        expect(messages[0].id).toBe('msg-1');
        expect(messages[0].role).toBe('user');
        expect(messages[1].id).toBe('msg-2');
        expect(messages[1].role).toBe('assistant');
      });

      it('should return empty array when no messages found', () => {
        mockDb.exec.mockReturnValue([]);

        const messages = messageRepository.findByConversation('conv-1');

        expect(messages).toEqual([]);
      });

      it('should order messages by sequence_number', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'sync_status', 'created_at', 'updated_at'],
            values: [
              ['msg-1', 'conv-1', 1, 'user', 'First', null, 'synced', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
              ['msg-2', 'conv-1', 2, 'assistant', 'Second', null, 'synced', '2024-01-01T00:01:00Z', '2024-01-01T00:01:00Z'],
            ],
          },
        ]);

        messageRepository.findByConversation('conv-1');

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('ORDER BY sequence_number ASC'),
          ['conv-1']
        );
      });
    });

    describe('findById', () => {
      it('should return a message by id', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'sync_status', 'created_at', 'updated_at'],
            values: [
              ['msg-1', 'conv-1', 1, 'user', 'Hello', 'local-1', 'synced', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
            ],
          },
        ]);

        const message = messageRepository.findById('msg-1');

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('WHERE id = ?'),
          ['msg-1']
        );

        expect(message).not.toBeNull();
        expect(message?.id).toBe('msg-1');
        expect(message?.contents).toBe('Hello');
      });

      it('should return null when message not found', () => {
        mockDb.exec.mockReturnValue([]);

        const message = messageRepository.findById('msg-999');

        expect(message).toBeNull();
      });
    });

    describe('insert', () => {
      it('should insert a new message', () => {
        messageRepository.insert(mockMessage);

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('INSERT INTO messages'),
          [
            'msg-1',
            'conv-1',
            1,
            'user',
            'Hello, world!',
            'local-1',
            null, // server_id
            'pending',
            0, // retry_count
            '2024-01-01T00:00:00Z',
            '2024-01-01T00:00:00Z',
          ]
        );

        expect(sqlite.scheduleSave).toHaveBeenCalled();
      });

      it('should insert message with null local_id', () => {
        const messageWithoutLocalId = { ...mockMessage, local_id: undefined };

        messageRepository.insert(messageWithoutLocalId);

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.any(String),
          expect.arrayContaining([null]) // local_id should be null
        );
      });

      it('should default sync_status to synced', () => {
        const messageWithoutStatus = { ...mockMessage, sync_status: undefined };

        messageRepository.insert(messageWithoutStatus);

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.any(String),
          expect.arrayContaining(['synced'])
        );
      });
    });

    describe('update', () => {
      beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(new Date('2024-01-02T00:00:00Z'));
      });

      afterEach(() => {
        vi.useRealTimers();
      });

      it('should update message contents', () => {
        messageRepository.update('msg-1', { contents: 'Updated content' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('UPDATE messages SET'),
          expect.arrayContaining(['Updated content', 'msg-1'])
        );

        expect(sqlite.scheduleSave).toHaveBeenCalled();
      });

      it('should update sync_status', () => {
        messageRepository.update('msg-1', { sync_status: 'synced' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('sync_status = ?'),
          expect.arrayContaining(['synced'])
        );
      });

      it('should update sequence_number', () => {
        messageRepository.update('msg-1', { sequence_number: 5 });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('sequence_number = ?'),
          expect.arrayContaining([5])
        );
      });

      it('should always update updated_at timestamp', () => {
        messageRepository.update('msg-1', { contents: 'New content' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('updated_at = ?'),
          expect.arrayContaining(['2024-01-02T00:00:00.000Z'])
        );
      });

      it('should update multiple fields', () => {
        messageRepository.update('msg-1', {
          contents: 'New content',
          sync_status: 'synced',
          sequence_number: 3,
        });

        const call = mockDb.run.mock.calls[0];
        const sql = call[0];

        expect(sql).toContain('contents = ?');
        expect(sql).toContain('sequence_number = ?');
        expect(sql).toContain('sync_status = ?');
        expect(sql).toContain('updated_at = ?');
      });
    });

    describe('delete', () => {
      it('should delete a message by id', () => {
        messageRepository.delete('msg-1');

        expect(mockDb.run).toHaveBeenCalledWith(
          'DELETE FROM messages WHERE id = ?',
          ['msg-1']
        );

        expect(sqlite.scheduleSave).toHaveBeenCalled();
      });
    });

    describe('getPending', () => {
      it('should return pending messages', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'server_id', 'sync_status', 'retry_count', 'created_at', 'updated_at'],
            values: [
              ['msg-1', 'conv-1', 1, 'user', 'Hello', 'local-1', null, 'pending', 0, '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
              ['msg-2', 'conv-1', 2, 'user', 'World', 'local-2', null, 'pending', 0, '2024-01-01T00:01:00Z', '2024-01-01T00:01:00Z'],
            ],
          },
        ]);

        const pending = messageRepository.getPending('conv-1');

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining("WHERE sync_status = ? AND conversation_id = ?"),
          ['pending', 'conv-1']
        );

        expect(pending).toHaveLength(2);
        expect(pending[0].sync_status).toBe('pending');
        expect(pending[1].sync_status).toBe('pending');
      });

      it('should return empty array when no pending messages', () => {
        mockDb.exec.mockReturnValue([]);

        const pending = messageRepository.getPending('conv-1');

        expect(pending).toEqual([]);
      });
    });

    describe('upsert', () => {
      it('should insert when message does not exist', () => {
        mockDb.exec.mockReturnValue([]);

        const insertSpy = vi.spyOn(messageRepository, 'insert');
        const updateSpy = vi.spyOn(messageRepository, 'update');

        messageRepository.upsert(mockMessage);

        expect(insertSpy).toHaveBeenCalledWith(mockMessage);
        expect(updateSpy).not.toHaveBeenCalled();
      });

      it('should update when message exists', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'conversation_id', 'sequence_number', 'role', 'contents', 'local_id', 'sync_status', 'created_at', 'updated_at'],
            values: [
              ['msg-1', 'conv-1', 1, 'user', 'Old content', 'local-1', 'pending', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
            ],
          },
        ]);

        const insertSpy = vi.spyOn(messageRepository, 'insert');
        const updateSpy = vi.spyOn(messageRepository, 'update');

        messageRepository.upsert(mockMessage);

        expect(insertSpy).not.toHaveBeenCalled();
        expect(updateSpy).toHaveBeenCalledWith(mockMessage.id, mockMessage);
      });
    });
  });

  describe('conversationRepository', () => {
    const mockConversation: Conversation = {
      id: 'conv-1',
      title: 'Test Conversation',
      status: 'active',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      last_client_stanza_id: 0,
      last_server_stanza_id: 0,
    };

    describe('findAll', () => {
      it('should return all conversations', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'title', 'status', 'created_at', 'updated_at'],
            values: [
              ['conv-1', 'Conversation 1', 'active', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
              ['conv-2', 'Conversation 2', 'archived', '2024-01-02T00:00:00Z', '2024-01-02T00:00:00Z'],
            ],
          },
        ]);

        const conversations = conversationRepository.findAll();

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('SELECT id, title, status, created_at, updated_at FROM conversations')
        );

        expect(conversations).toHaveLength(2);
        expect(conversations[0].title).toBe('Conversation 1');
        expect(conversations[1].title).toBe('Conversation 2');
      });

      it('should order by updated_at DESC', () => {
        mockDb.exec.mockReturnValue([]);

        conversationRepository.findAll();

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('ORDER BY updated_at DESC')
        );
      });

      it('should return empty array when no conversations', () => {
        mockDb.exec.mockReturnValue([]);

        const conversations = conversationRepository.findAll();

        expect(conversations).toEqual([]);
      });
    });

    describe('findById', () => {
      it('should return a conversation by id', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'title', 'status', 'created_at', 'updated_at'],
            values: [
              ['conv-1', 'Test Conversation', 'active', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
            ],
          },
        ]);

        const conversation = conversationRepository.findById('conv-1');

        expect(mockDb.exec).toHaveBeenCalledWith(
          expect.stringContaining('WHERE id = ?'),
          ['conv-1']
        );

        expect(conversation).not.toBeNull();
        expect(conversation?.id).toBe('conv-1');
        expect(conversation?.title).toBe('Test Conversation');
      });

      it('should return null when conversation not found', () => {
        mockDb.exec.mockReturnValue([]);

        const conversation = conversationRepository.findById('conv-999');

        expect(conversation).toBeNull();
      });
    });

    describe('insert', () => {
      it('should insert a new conversation', () => {
        conversationRepository.insert(mockConversation);

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('INSERT INTO conversations'),
          [
            'conv-1',
            'Test Conversation',
            'active',
            '2024-01-01T00:00:00Z',
            '2024-01-01T00:00:00Z',
          ]
        );

        expect(sqlite.scheduleSave).toHaveBeenCalled();
      });
    });

    describe('update', () => {
      beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(new Date('2024-01-02T00:00:00Z'));
      });

      afterEach(() => {
        vi.useRealTimers();
      });

      it('should update conversation title', () => {
        conversationRepository.update('conv-1', { title: 'Updated Title' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('UPDATE conversations SET'),
          expect.arrayContaining(['Updated Title', 'conv-1'])
        );

        expect(sqlite.scheduleSave).toHaveBeenCalled();
      });

      it('should update conversation status', () => {
        conversationRepository.update('conv-1', { status: 'archived' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('status = ?'),
          expect.arrayContaining(['archived'])
        );
      });

      it('should always update updated_at timestamp', () => {
        conversationRepository.update('conv-1', { title: 'New Title' });

        expect(mockDb.run).toHaveBeenCalledWith(
          expect.stringContaining('updated_at = ?'),
          expect.arrayContaining(['2024-01-02T00:00:00.000Z'])
        );
      });
    });

    describe('upsert', () => {
      it('should insert when conversation does not exist', () => {
        mockDb.exec.mockReturnValue([]);

        const insertSpy = vi.spyOn(conversationRepository, 'insert');
        const updateSpy = vi.spyOn(conversationRepository, 'update');

        conversationRepository.upsert(mockConversation);

        expect(insertSpy).toHaveBeenCalledWith(mockConversation);
        expect(updateSpy).not.toHaveBeenCalled();
      });

      it('should update when conversation exists', () => {
        mockDb.exec.mockReturnValue([
          {
            columns: ['id', 'title', 'status', 'created_at', 'updated_at'],
            values: [
              ['conv-1', 'Old Title', 'active', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z'],
            ],
          },
        ]);

        const insertSpy = vi.spyOn(conversationRepository, 'insert');
        const updateSpy = vi.spyOn(conversationRepository, 'update');

        conversationRepository.upsert(mockConversation);

        expect(insertSpy).not.toHaveBeenCalled();
        expect(updateSpy).toHaveBeenCalledWith(mockConversation.id, mockConversation);
      });
    });
  });
});
