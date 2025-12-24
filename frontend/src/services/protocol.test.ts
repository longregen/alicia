import { describe, it, expect, beforeEach } from 'vitest';
import { ProtocolService } from './protocol';
import { MessageType } from '../types/protocol';

describe('ProtocolService', () => {
  let service: ProtocolService;

  beforeEach(() => {
    service = new ProtocolService();
  });

  describe('nextStanzaId', () => {
    it('should start at 1 and increment', () => {
      expect(service.nextStanzaId()).toBe(1);
      expect(service.nextStanzaId()).toBe(2);
      expect(service.nextStanzaId()).toBe(3);
    });

    it('should maintain separate counters for different instances', () => {
      const service2 = new ProtocolService();

      expect(service.nextStanzaId()).toBe(1);
      expect(service2.nextStanzaId()).toBe(1);
      expect(service.nextStanzaId()).toBe(2);
      expect(service2.nextStanzaId()).toBe(2);
    });
  });

  describe('encode and decode', () => {
    it('should encode and decode a simple envelope', () => {
      const envelope = {
        stanzaId: 1,
        conversationId: 'conv-123',
        type: MessageType.UserMessage,
        body: {
          id: 'msg-1',
          conversationId: 'conv-123',
          content: 'Hello, world!',
          timestamp: 1234567890,
        },
      };

      const encoded = service.encode(envelope);
      // In Node.js, msgpackr returns Buffer which extends Uint8Array
      expect(encoded).toBeInstanceOf(Buffer);
      expect(encoded.length).toBeGreaterThan(0);

      const decoded = service.decode(encoded);
      expect(decoded).toEqual(envelope);
    });

    it('should handle envelopes with metadata', () => {
      const envelope = {
        stanzaId: 1,
        conversationId: 'conv-123',
        type: MessageType.Configuration,
        meta: { foo: 'bar', num: 42 },
        body: {
          conversationId: 'conv-123',
          clientVersion: '1.0.0',
        },
      };

      const encoded = service.encode(envelope);
      const decoded = service.decode(encoded);

      expect(decoded.meta).toEqual({ foo: 'bar', num: 42 });
    });

    it('should handle binary data in body', () => {
      const binaryData = new Uint8Array([1, 2, 3, 4, 5]);
      const envelope = {
        stanzaId: 1,
        conversationId: 'conv-123',
        type: MessageType.AudioChunk,
        body: {
          conversationId: 'conv-123',
          format: 'pcm',
          sequence: 1,
          durationMs: 100,
          data: binaryData,
        },
      };

      const encoded = service.encode(envelope);
      const decoded = service.decode(encoded);

      // msgpackr may return Buffer instead of Uint8Array
      // Compare the actual byte values
      expect(Array.from(decoded.body.data)).toEqual(Array.from(binaryData));
    });
  });

  describe('createUserMessage', () => {
    it('should create a valid UserMessage envelope', () => {
      const envelope = service.createUserMessage(
        'conv-123',
        'Hello, assistant!',
        'prev-msg-1'
      );

      expect(envelope.stanzaId).toBe(1);
      expect(envelope.conversationId).toBe('conv-123');
      expect(envelope.type).toBe(MessageType.UserMessage);
      expect(envelope.body.id).toMatch(/^msg_/);
      expect(envelope.body.conversationId).toBe('conv-123');
      expect(envelope.body.content).toBe('Hello, assistant!');
      expect(envelope.body.previousId).toBe('prev-msg-1');
      expect(envelope.body.timestamp).toBeTypeOf('number');
    });

    it('should create UserMessage without previousId', () => {
      const envelope = service.createUserMessage('conv-123', 'First message');

      expect(envelope.body.previousId).toBeUndefined();
      expect(envelope.body.content).toBe('First message');
    });

    it('should generate unique message IDs', () => {
      const envelope1 = service.createUserMessage('conv-123', 'Message 1');
      const envelope2 = service.createUserMessage('conv-123', 'Message 2');

      expect(envelope1.body.id).not.toBe(envelope2.body.id);
    });

    it('should increment stanzaId for each message', () => {
      const envelope1 = service.createUserMessage('conv-123', 'Message 1');
      const envelope2 = service.createUserMessage('conv-123', 'Message 2');

      expect(envelope1.stanzaId).toBe(1);
      expect(envelope2.stanzaId).toBe(2);
    });
  });

  describe('createConfiguration', () => {
    it('should create a valid Configuration envelope', () => {
      const features = ['streaming', 'audio_output'];
      const envelope = service.createConfiguration('conv-123', features, 42);

      expect(envelope.stanzaId).toBe(1);
      expect(envelope.conversationId).toBe('conv-123');
      expect(envelope.type).toBe(MessageType.Configuration);
      expect(envelope.body.conversationId).toBe('conv-123');
      expect(envelope.body.clientVersion).toBe('0.1.0');
      expect(envelope.body.features).toEqual(features);
      expect(envelope.body.device).toBe('web');
      expect(envelope.body.lastSequenceSeen).toBe(42);
    });

    it('should create Configuration without lastSequenceSeen', () => {
      const features = ['streaming'];
      const envelope = service.createConfiguration('conv-123', features);

      expect(envelope.body.lastSequenceSeen).toBeUndefined();
    });
  });

  describe('createControlStop', () => {
    it('should create a valid ControlStop envelope', () => {
      const envelope = service.createControlStop('conv-123', 'msg-456');

      expect(envelope.stanzaId).toBe(1);
      expect(envelope.conversationId).toBe('conv-123');
      expect(envelope.type).toBe(MessageType.ControlStop);
      expect(envelope.body.conversationId).toBe('conv-123');
      expect(envelope.body.stopType).toBe('all');
      expect(envelope.body.targetId).toBe('msg-456');
      expect(envelope.body.reason).toBe('user_requested');
    });

    it('should create ControlStop without targetId', () => {
      const envelope = service.createControlStop('conv-123');

      expect(envelope.body.targetId).toBeUndefined();
      expect(envelope.body.stopType).toBe('all');
    });
  });

  describe('createControlVariation', () => {
    it('should create a valid ControlVariation envelope with regenerate', () => {
      const envelope = service.createControlVariation(
        'conv-123',
        'msg-456',
        'regenerate'
      );

      expect(envelope.stanzaId).toBe(1);
      expect(envelope.conversationId).toBe('conv-123');
      expect(envelope.type).toBe(MessageType.ControlVariation);
      expect(envelope.body.conversationId).toBe('conv-123');
      expect(envelope.body.targetId).toBe('msg-456');
      expect(envelope.body.mode).toBe('regenerate');
      expect(envelope.body.newContent).toBeUndefined();
    });

    it('should create ControlVariation with edit and new content', () => {
      const envelope = service.createControlVariation(
        'conv-123',
        'msg-456',
        'edit',
        'New edited content'
      );

      expect(envelope.body.mode).toBe('edit');
      expect(envelope.body.newContent).toBe('New edited content');
    });

    it('should default to regenerate variation type', () => {
      const envelope = service.createControlVariation('conv-123', 'msg-456');

      expect(envelope.body.mode).toBe('regenerate');
    });

    it('should support continue variation type', () => {
      const envelope = service.createControlVariation(
        'conv-123',
        'msg-456',
        'continue'
      );

      expect(envelope.body.mode).toBe('continue');
    });
  });

  describe('round-trip encoding', () => {
    it('should preserve UserMessage through encode/decode', () => {
      const original = service.createUserMessage(
        'conv-123',
        'Test message',
        'prev-1'
      );

      const encoded = service.encode(original);
      const decoded = service.decode(encoded);

      expect(decoded).toEqual(original);
    });

    it('should preserve Configuration through encode/decode', () => {
      const original = service.createConfiguration(
        'conv-123',
        ['streaming', 'audio'],
        10
      );

      const encoded = service.encode(original);
      const decoded = service.decode(encoded);

      expect(decoded).toEqual(original);
    });

    it('should preserve ControlStop through encode/decode', () => {
      const original = service.createControlStop('conv-123', 'target-1');

      const encoded = service.encode(original);
      const decoded = service.decode(encoded);

      expect(decoded).toEqual(original);
    });

    it('should preserve ControlVariation through encode/decode', () => {
      const original = service.createControlVariation(
        'conv-123',
        'msg-1',
        'edit',
        'New content'
      );

      const encoded = service.encode(original);
      const decoded = service.decode(encoded);

      expect(decoded).toEqual(original);
    });
  });
});
