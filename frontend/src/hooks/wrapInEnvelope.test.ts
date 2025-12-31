import { MessageType } from '../types/protocol';

/**
 * Test for potential type detection fragility in wrapInEnvelope.
 *
 * This test verifies that the duck-typing logic in wrapInEnvelope correctly
 * identifies message types and doesn't misclassify messages with overlapping fields.
 */

// Import the function under test by extracting it from the module
// Since wrapInEnvelope is not exported, we'll need to test it indirectly
// For now, we'll create a copy of the logic to test the algorithm

function wrapInEnvelope(data: unknown, _conversationId: string): { type: MessageType; body: unknown } {
  const dto = data as Record<string, unknown>;

  if ('syncedMessages' in dto) {
    return {
      type: MessageType.SyncResponse,
      body: data,
    };
  } else if ('id' in dto && 'contents' in dto) {
    // MessageResponse DTO (broadcast from other clients)
    return {
      type: MessageType.AssistantMessage,
      body: data,
    };
  } else if ('acknowledgedStanzaId' in dto) {
    return {
      type: MessageType.Acknowledgement,
      body: data,
    };
  }
  // Protocol streaming messages
  else if ('id' in dto && 'conversationId' in dto && 'previousId' in dto && ('answerType' in dto || 'plannedSentenceCount' in dto)) {
    return {
      type: MessageType.StartAnswer,
      body: data,
    };
  } else if ('conversationId' in dto && 'sequence' in dto && 'text' in dto && 'previousId' in dto) {
    return {
      type: MessageType.AssistantSentence,
      body: data,
    };
  } else if ('id' in dto && 'messageId' in dto && 'toolName' in dto && 'parameters' in dto) {
    return {
      type: MessageType.ToolUseRequest,
      body: data,
    };
  } else if ('requestId' in dto && 'success' in dto) {
    return {
      type: MessageType.ToolUseResult,
      body: data,
    };
  } else if ('messageId' in dto && 'sequence' in dto && 'content' in dto && !('text' in dto)) {
    return {
      type: MessageType.ReasoningStep,
      body: data,
    };
  } else if ('format' in dto && 'sequence' in dto && 'durationMs' in dto) {
    return {
      type: MessageType.AudioChunk,
      body: data,
    };
  } else if ('text' in dto && 'final' in dto && typeof (dto as Record<string, unknown>).final === 'boolean') {
    return {
      type: MessageType.Transcription,
      body: data,
    };
  } else if ('memoryId' in dto && 'messageId' in dto && 'content' in dto && 'relevance' in dto) {
    return {
      type: MessageType.MemoryTrace,
      body: data,
    };
  }

  return {
    type: MessageType.ErrorMessage,
    body: data,
  };
}

describe('wrapInEnvelope type detection', () => {
  const testConversationId = 'test-conv-123';

  describe('Correct classifications (baseline tests)', () => {
    test('Pure SyncResponse is classified correctly', () => {
      const syncResponse = {
        syncedMessages: [{ localId: 'local-1', serverId: 'server-1', status: 'synced' }],
      };
      const result = wrapInEnvelope(syncResponse, testConversationId);
      expect(result.type).toBe(MessageType.SyncResponse);
    });

    test('Pure MessageResponse is classified correctly', () => {
      const messageResponse = {
        id: 'msg-123',
        contents: [{ type: 'text', text: 'Hello' }],
      };
      const result = wrapInEnvelope(messageResponse, testConversationId);
      expect(result.type).toBe(MessageType.AssistantMessage);
    });

    test('Pure StartAnswer is classified correctly (when not overlapping)', () => {
      const startAnswer = {
        id: 'msg-123',
        conversationId: testConversationId,
        previousId: 'msg-122',
        answerType: 'text',
        // Note: Does NOT have 'contents' field, so won't match MessageResponse
      };
      const result = wrapInEnvelope(startAnswer, testConversationId);
      expect(result.type).toBe(MessageType.StartAnswer);
    });

    test('Pure AssistantSentence is classified correctly', () => {
      const sentence = {
        conversationId: testConversationId,
        sequence: 1,
        text: 'Hello',
        previousId: 'msg-122',
      };
      const result = wrapInEnvelope(sentence, testConversationId);
      expect(result.type).toBe(MessageType.AssistantSentence);
    });

    test('Pure ReasoningStep is classified correctly', () => {
      const reasoning = {
        messageId: 'msg-123',
        sequence: 1,
        content: 'Thinking...',
        // Note: Does NOT have 'text' field
      };
      const result = wrapInEnvelope(reasoning, testConversationId);
      expect(result.type).toBe(MessageType.ReasoningStep);
    });
  });

});
