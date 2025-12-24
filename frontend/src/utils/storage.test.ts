import { describe, it, expect, beforeEach } from 'vitest';
import { storage } from './storage';

describe('storage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('selectedConversationId', () => {
    it('returns null when no value is stored', () => {
      expect(storage.getSelectedConversationId()).toBeNull();
    });

    it('stores and retrieves conversation ID', () => {
      storage.setSelectedConversationId('conv-123');
      expect(storage.getSelectedConversationId()).toBe('conv-123');
    });

    it('removes conversation ID when set to null', () => {
      storage.setSelectedConversationId('conv-123');
      storage.setSelectedConversationId(null);
      expect(storage.getSelectedConversationId()).toBeNull();
    });
  });

  describe('voiceMode', () => {
    it('returns false when no value is stored', () => {
      expect(storage.getVoiceMode()).toBe(false);
    });

    it('stores and retrieves voice mode enabled', () => {
      storage.setVoiceMode(true);
      expect(storage.getVoiceMode()).toBe(true);
    });

    it('stores and retrieves voice mode disabled', () => {
      storage.setVoiceMode(false);
      expect(storage.getVoiceMode()).toBe(false);
    });
  });
});
