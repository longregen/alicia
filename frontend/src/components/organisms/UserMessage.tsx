import React, { useState, useEffect, useMemo } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import AudioAddon from '../atoms/AudioAddon';
import { useConversationStore, selectSentences } from '../../stores/conversationStore';
import { useAudioManager } from '../../hooks/useAudioManager';
import { useAudioStore } from '../../stores/audioStore';
import { MESSAGE_TYPES, MESSAGE_STATES, AUDIO_STATES } from '../../mockData';
import type { MessageId } from '../../types/streaming';
import type { MessageAddon, AudioState } from '../../types/components';

/**
 * UserMessage organism component.
 *
 * Wraps ChatBubble for user messages with:
 * - User transcription as content
 * - Microphone icon for voice messages
 * - Data fetched from Zustand store
 */

export interface UserMessageProps {
  messageId: MessageId;
  className?: string;
}

const UserMessage: React.FC<UserMessageProps> = ({ messageId, className = '' }) => {
  const message = useConversationStore((state) => state.messages[messageId]);
  const sentencesMap = useConversationStore(selectSentences);

  // Memoize sentences to avoid creating new arrays on every render
  const sentences = useMemo(() => {
    if (!message) return [];
    return message.sentenceIds
      .map(id => sentencesMap[id])
      .filter(Boolean)
      .sort((a, b) => a.sequence - b.sequence);
  }, [message, sentencesMap]);

  const audioManager = useAudioManager();
  const currentlyPlayingId = useAudioStore((state) => state.playback.currentlyPlayingId);
  const isPlaying = useAudioStore((state) => state.playback.isPlaying);
  const playbackProgress = useAudioStore((state) => state.playback.playbackProgress);

  // Track audio state for each audio ref
  const [audioStates, setAudioStates] = useState<Record<string, AudioState>>({});

  // Find sentences with audio - memoized to avoid new array on every render
  const sentencesWithAudio = useMemo(
    () => sentences.filter(s => s.audioRefId),
    [sentences]
  );
  const hasAudio = sentencesWithAudio.length > 0;

  // Update audio states based on playback - must be called unconditionally
  useEffect(() => {
    if (!message) return; // Guard inside effect, not before

    const newStates: Record<string, AudioState> = {};

    sentencesWithAudio.forEach(sentence => {
      if (!sentence.audioRefId) return;

      if (currentlyPlayingId === sentence.audioRefId) {
        newStates[sentence.audioRefId] = isPlaying ? AUDIO_STATES.PLAYING : AUDIO_STATES.PAUSED;
      } else {
        newStates[sentence.audioRefId] = AUDIO_STATES.IDLE;
      }
    });

    setAudioStates(newStates);
  }, [message, currentlyPlayingId, isPlaying, sentencesWithAudio]);

  // Early return after all hooks
  if (!message) {
    return null;
  }

  // Build addons for voice messages
  const addons: MessageAddon[] = hasAudio ? [
    {
      id: `${messageId}-voice`,
      type: 'icon',
      position: 'inline',
      emoji: '🎤',
      tooltip: 'Voice message',
    }
  ] : [];

  // Add audio addons for sentences with audio
  sentencesWithAudio.forEach(sentence => {
    if (!sentence.audioRefId) return;

    const audioRef = audioManager.getMetadata(sentence.audioRefId);
    const audioRefId = sentence.audioRefId;

    addons.push({
      id: `audio-${sentence.id}`,
      type: 'audio',
      position: 'below',
      emoji: '🔊',
      tooltip: 'Play recorded audio',
      content: (
        <AudioAddon
          state={audioStates[audioRefId] || AUDIO_STATES.IDLE}
          duration={audioRef ? audioRef.durationMs / 1000 : 0}
          currentTime={currentlyPlayingId === audioRefId ? (playbackProgress * (audioRef?.durationMs || 0) / 1000) : 0}
          onPlay={() => audioManager.play(audioRefId)}
          onPause={() => audioManager.stop()}
          onStop={() => audioManager.stop()}
          mode="full"
        />
      ),
    });
  });

  return (
    <ChatBubble
      type={MESSAGE_TYPES.USER}
      content={message.content}
      state={MESSAGE_STATES.COMPLETED}
      timestamp={message.createdAt}
      addons={addons}
      className={className}
    />
  );
};

export default UserMessage;
