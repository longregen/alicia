import { pack, unpack } from 'msgpackr';
import { Envelope, MessageType, UserMessage } from '../types/protocol';

export class ProtocolService {
  encode(envelope: Envelope): Uint8Array {
    return pack(envelope);
  }

  decode(data: Uint8Array): Envelope {
    return unpack(data) as Envelope;
  }

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
    };

    return {
      conversationId,
      type: MessageType.UserMessage,
      body,
    };
  }

  private generateId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
  }
}
