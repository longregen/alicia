import { pack, unpack } from 'msgpackr';
import { Envelope, MessageType, UserMessage, Configuration, ControlStop, ControlVariation, VariationType } from '../types/protocol';

/**
 * Protocol service for encoding and decoding MessagePack messages
 */
export class ProtocolService {
  private clientStanzaId = 0;

  /**
   * Get next client stanza ID (positive, incrementing)
   */
  nextStanzaId(): number {
    return ++this.clientStanzaId;
  }

  /**
   * Encode an envelope to MessagePack binary
   */
  encode(envelope: Envelope): Uint8Array {
    return pack(envelope);
  }

  /**
   * Decode MessagePack binary to envelope
   */
  decode(data: Uint8Array): Envelope {
    return unpack(data) as Envelope;
  }

  /**
   * Create a UserMessage envelope
   */
  createUserMessage(
    conversationId: string,
    content: string,
    previousId?: string
  ): Envelope {
    const body: UserMessage = {
      id: this.generateId(),
      conversation_id: conversationId,
      content,
      previous_id: previousId,
      timestamp: Date.now(),
    };

    return {
      stanza_id: this.nextStanzaId(),
      conversation_id: conversationId,
      type: MessageType.UserMessage,
      body,
    };
  }

  /**
   * Create a Configuration envelope
   */
  createConfiguration(
    conversationId: string,
    features: string[],
    lastSequenceSeen?: number
  ): Envelope {
    const body: Configuration = {
      conversation_id: conversationId,
      client_version: '0.1.0',
      features,
      device: 'web',
      last_sequence_seen: lastSequenceSeen,
    };

    return {
      stanza_id: this.nextStanzaId(),
      conversation_id: conversationId,
      type: MessageType.Configuration,
      body,
    };
  }

  /**
   * Create a ControlStop envelope
   */
  createControlStop(conversationId: string, targetId?: string): Envelope {
    const body: ControlStop = {
      conversation_id: conversationId,
      stop_type: 'all',
      target_id: targetId,
      reason: 'user_requested',
    };

    return {
      stanza_id: this.nextStanzaId(),
      conversation_id: conversationId,
      type: MessageType.ControlStop,
      body,
    };
  }

  /**
   * Create a ControlVariation envelope
   */
  createControlVariation(
    conversationId: string,
    targetId: string,
    variationType: VariationType = 'regenerate',
    newContent?: string
  ): Envelope {
    const body: ControlVariation = {
      conversation_id: conversationId,
      target_id: targetId,
      mode: variationType,
      new_content: newContent,
    };

    return {
      stanza_id: this.nextStanzaId(),
      conversation_id: conversationId,
      type: MessageType.ControlVariation,
      body,
    };
  }

  /**
   * Generate a unique message ID
   */
  private generateId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}
