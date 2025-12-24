/**
 * Simple localStorage wrapper for user settings persistence
 */

const STORAGE_KEYS = {
  SELECTED_CONVERSATION_ID: 'alicia.selectedConversationId',
  VOICE_MODE: 'alicia.voiceMode',
} as const;

export const storage = {
  // Selected conversation
  getSelectedConversationId(): string | null {
    return localStorage.getItem(STORAGE_KEYS.SELECTED_CONVERSATION_ID);
  },

  setSelectedConversationId(id: string | null): void {
    if (id === null) {
      localStorage.removeItem(STORAGE_KEYS.SELECTED_CONVERSATION_ID);
    } else {
      localStorage.setItem(STORAGE_KEYS.SELECTED_CONVERSATION_ID, id);
    }
  },

  // Voice mode
  getVoiceMode(): boolean {
    const value = localStorage.getItem(STORAGE_KEYS.VOICE_MODE);
    return value === 'true';
  },

  setVoiceMode(enabled: boolean): void {
    localStorage.setItem(STORAGE_KEYS.VOICE_MODE, enabled.toString());
  },
};
