import { pack, unpack } from 'msgpackr';
import { Envelope, MessageType, UserMessage, Configuration, ControlStop, ControlVariation, VariationType, DimensionPreference, DimensionWeights, EliteSelect } from '../types/protocol';

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
      conversationId,
      content,
      previousId,
      timestamp: Date.now(),
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
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
      conversationId,
      clientVersion: '0.1.0',
      features,
      device: 'web',
      lastSequenceSeen,
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
      type: MessageType.Configuration,
      body,
    };
  }

  /**
   * Create a ControlStop envelope
   */
  createControlStop(conversationId: string, targetId?: string): Envelope {
    const body: ControlStop = {
      conversationId,
      stopType: 'all',
      targetId,
      reason: 'user_requested',
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
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
      conversationId,
      targetId,
      mode: variationType,
      newContent,
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
      type: MessageType.ControlVariation,
      body,
    };
  }

  /**
   * Create a DimensionPreference envelope
   */
  createDimensionPreference(
    conversationId: string,
    weights: DimensionWeights,
    preset?: string
  ): Envelope {
    const body: DimensionPreference = {
      conversationId,
      weights,
      preset: preset as 'accuracy' | 'speed' | 'reliable' | 'creative' | 'balanced' | undefined,
      timestamp: Date.now(),
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
      type: MessageType.DimensionPreference,
      body,
    };
  }

  /**
   * Create an EliteSelect envelope
   */
  createEliteSelect(
    conversationId: string,
    eliteId: string
  ): Envelope {
    const body: EliteSelect = {
      conversationId,
      eliteId,
      timestamp: Date.now(),
    };

    return {
      stanzaId: this.nextStanzaId(),
      conversationId,
      type: MessageType.EliteSelect,
      body,
    };
  }

  /**
   * Generate a unique message ID
   */
  private generateId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
  }
}
