/**
 * Shared test utilities for MessagePack + WebSocket sync protocol
 */

// WebSocket mocking
export {
  MockWebSocket,
  createMockWebSocket,
  installWebSocketMock,
  uninstallWebSocketMock,
  type WebSocketMessage,
  type MockWebSocketOptions,
} from './websocket-mock';

// MessagePack fixtures
export {
  userMessageFixture,
  assistantMessageFixture,
  conversationFixture,
  syncRequestEnvelope,
  syncResponseEnvelope,
  messageEnvelope,
  ackEnvelope,
  protocolFixtures,
  encodedFixtures,
  createUserMessage,
  createAssistantMessage,
  createConversation,
  createMessageBatch,
  createConflictFixture,
} from './msgpack-fixtures';

// Sync protocol builders
export {
  SyncProtocolBuilder,
  SyncFlowSimulator,
  createSyncProtocolBuilder,
  createSyncFlowSimulator,
} from './sync-protocol';

// SQLite mocking
export {
  TestDatabase,
  createTestDatabase,
  mockDatabaseFunctions,
  setupTestDatabase,
} from './sqlite-mock';
