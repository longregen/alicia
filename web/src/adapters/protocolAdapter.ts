import { Envelope } from '../types/protocol';

let messageSender: ((envelope: Envelope) => void) | null = null;

export function setMessageSender(sender: ((envelope: Envelope) => void) | null): void {
  messageSender = sender;
}

