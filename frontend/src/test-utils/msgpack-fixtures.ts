import { pack } from 'msgpackr';
import { Message, Conversation } from '../types/models';
import {
  Envelope,
  MessageType,
  UserMessage,
  AssistantMessage,
  Configuration,
  ControlStop,
  ControlVariation
} from '../types/protocol';

/**
 * Test fixture for a basic user message
 */
export const userMessageFixture: Message = {
  id: 'msg-user-1',
  conversation_id: 'conv-test-1',
  sequence_number: 1,
  role: 'user',
  contents: 'Hello, how are you?',
  local_id: 'local-msg-1',
  sync_status: 'pending',
  created_at: '2024-01-01T10:00:00Z',
  updated_at: '2024-01-01T10:00:00Z',
};

/**
 * Test fixture for an assistant message
 */
export const assistantMessageFixture: Message = {
  id: 'msg-assistant-1',
  conversation_id: 'conv-test-1',
  sequence_number: 2,
  role: 'assistant',
  contents: 'I am doing well, thank you for asking!',
  sync_status: 'synced',
  created_at: '2024-01-01T10:00:05Z',
  updated_at: '2024-01-01T10:00:05Z',
};

/**
 * Test fixture for a conversation
 */
export const conversationFixture: Conversation = {
  id: 'conv-test-1',
  title: 'Test Conversation',
  status: 'active',
  created_at: '2024-01-01T09:00:00Z',
  updated_at: '2024-01-01T10:00:00Z',
  last_client_stanza_id: 5,
  last_server_stanza_id: 10,
};

/**
 * Test fixture for a sync request envelope
 */
export const syncRequestEnvelope = {
  type: 'sync_request',
  payload: {
    messages: [userMessageFixture],
  },
};

/**
 * Test fixture for a sync response envelope (wire format uses camelCase)
 */
export const syncResponseEnvelope = {
  type: 'sync_response',
  payload: {
    syncedMessages: [
      {
        localId: userMessageFixture.local_id,
        serverId: userMessageFixture.id,
        status: 'synced',
        message: {
          id: userMessageFixture.id,
          conversationId: userMessageFixture.conversation_id,
          sequenceNumber: userMessageFixture.sequence_number,
          role: userMessageFixture.role,
          contents: userMessageFixture.contents,
          createdAt: userMessageFixture.created_at,
          updatedAt: userMessageFixture.updated_at,
          syncStatus: 'synced',
        },
      },
    ],
    syncedAt: new Date().toISOString(),
  },
};

/**
 * Test fixture for a message envelope
 */
export const messageEnvelope = {
  type: 'message',
  payload: {
    message: assistantMessageFixture,
  },
};

/**
 * Test fixture for an ack envelope
 */
export const ackEnvelope = {
  type: 'ack',
  payload: {
    message_id: 'msg-user-1',
    status: 'received',
  },
};

/**
 * Protocol envelope fixtures
 */
export const protocolFixtures = {
  userMessage: {
    stanzaId: 1,
    conversationId: 'conv-test-1',
    type: MessageType.UserMessage,
    body: {
      id: 'msg-protocol-1',
      conversationId: 'conv-test-1',
      content: 'Test message content',
      timestamp: 1704096000000,
    } as UserMessage,
  } as Envelope,

  assistantMessage: {
    stanzaId: 2,
    conversationId: 'conv-test-1',
    type: MessageType.AssistantMessage,
    body: {
      id: 'msg-assistant-protocol-1',
      conversationId: 'conv-test-1',
      messageId: 'msg-assistant-protocol-1',
      content: 'Assistant response',
      sequence: 2,
      timestamp: 1704096005000,
    } as AssistantMessage,
  } as Envelope,

  configuration: {
    stanzaId: 1,
    conversationId: 'conv-test-1',
    type: MessageType.Configuration,
    body: {
      conversationId: 'conv-test-1',
      clientVersion: '0.1.0',
      features: ['streaming', 'audio_output'],
      device: 'web',
      lastSequenceSeen: 10,
    } as Configuration,
  } as Envelope,

  controlStop: {
    stanzaId: 3,
    conversationId: 'conv-test-1',
    type: MessageType.ControlStop,
    body: {
      conversationId: 'conv-test-1',
      stopType: 'all',
      targetId: 'msg-target-1',
      reason: 'user_requested',
    } as ControlStop,
  } as Envelope,

  controlVariation: {
    stanzaId: 4,
    conversationId: 'conv-test-1',
    type: MessageType.ControlVariation,
    body: {
      conversationId: 'conv-test-1',
      targetId: 'msg-target-2',
      mode: 'regenerate',
    } as ControlVariation,
  } as Envelope,
};

/**
 * Pre-encoded MessagePack test data
 */
export const encodedFixtures = {
  syncRequest: pack(syncRequestEnvelope),
  syncResponse: pack(syncResponseEnvelope),
  message: pack(messageEnvelope),
  ack: pack(ackEnvelope),

  userMessage: pack(protocolFixtures.userMessage),
  assistantMessage: pack(protocolFixtures.assistantMessage),
  configuration: pack(protocolFixtures.configuration),
  controlStop: pack(protocolFixtures.controlStop),
  controlVariation: pack(protocolFixtures.controlVariation),
};

/**
 * Create a custom user message fixture
 */
export function createUserMessage(
  overrides: Partial<Message> = {}
): Message {
  return {
    ...userMessageFixture,
    id: `msg-user-${Date.now()}`,
    local_id: `local-${Date.now()}`,
    ...overrides,
  };
}

/**
 * Create a custom assistant message fixture
 */
export function createAssistantMessage(
  overrides: Partial<Message> = {}
): Message {
  return {
    ...assistantMessageFixture,
    id: `msg-assistant-${Date.now()}`,
    ...overrides,
  };
}

/**
 * Create a custom conversation fixture
 */
export function createConversation(
  overrides: Partial<Conversation> = {}
): Conversation {
  return {
    ...conversationFixture,
    id: `conv-${Date.now()}`,
    ...overrides,
  };
}

/**
 * Create a batch of messages for testing
 */
export function createMessageBatch(
  count: number,
  conversationId = 'conv-test-1',
  role: 'user' | 'assistant' = 'user'
): Message[] {
  const messages: Message[] = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    messages.push({
      id: `msg-${role}-${i}`,
      conversation_id: conversationId,
      sequence_number: i + 1,
      role,
      contents: `Message ${i + 1}`,
      local_id: role === 'user' ? `local-${i}` : undefined,
      sync_status: i % 2 === 0 ? 'synced' : 'pending',
      created_at: new Date(now + i * 1000).toISOString(),
      updated_at: new Date(now + i * 1000).toISOString(),
    });
  }

  return messages;
}

/**
 * Create a conflict scenario fixture (wire format uses camelCase)
 */
export function createConflictFixture() {
  const localMessage = createUserMessage({
    id: 'msg-conflict',
    local_id: 'local-conflict',
    contents: 'Local version',
    sync_status: 'conflict',
  });

  const serverMessage = createUserMessage({
    id: 'msg-conflict-server',
    local_id: 'local-conflict',
    contents: 'Server version',
    sync_status: 'synced',
  });

  // Wire format syncResponse uses camelCase
  return {
    local: localMessage,
    server: serverMessage,
    syncResponse: {
      syncedMessages: [
        {
          localId: 'local-conflict',
          serverId: 'msg-conflict-server',
          status: 'conflict',
          conflict: {
            reason: 'Content mismatch',
            serverMessage: {
              id: serverMessage.id,
              conversationId: serverMessage.conversation_id,
              sequenceNumber: serverMessage.sequence_number,
              role: serverMessage.role,
              contents: serverMessage.contents,
              createdAt: serverMessage.created_at,
              updatedAt: serverMessage.updated_at,
              localId: serverMessage.local_id,
              syncStatus: serverMessage.sync_status,
            },
            resolution: 'server_wins',
          },
        },
      ],
      syncedAt: new Date().toISOString(),
    },
  };
}
