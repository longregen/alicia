import React, { useState, useEffect } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import AudioAddon from '../atoms/AudioAddon';
import MemoryTraceAddon from '../atoms/MemoryTraceAddon';
import { useConversationStore } from '../../stores/conversationStore';
import { useAudioManager } from '../../hooks/useAudioManager';
import { useAudioStore } from '../../stores/audioStore';
import { MESSAGE_TYPES, MESSAGE_STATES, AUDIO_STATES } from '../../mockData';
import type { MessageId } from '../../types/streaming';
import type { MessageAddon, ToolData, AudioState } from '../../types/components';

/**
 * AssistantMessage organism component.
 *
 * Wraps ChatBubble for assistant messages with:
 * - ComplexAddons integration for tool calls
 * - Memory trace addons
 * - Reasoning blocks in content (handled by ChatBubble)
 * - Data fetched from Zustand store
 */

export interface AssistantMessageProps {
  messageId: MessageId;
  className?: string;
}

const AssistantMessage: React.FC<AssistantMessageProps> = ({ messageId, className = '' }) => {
  const message = useConversationStore((state) => state.messages[messageId]);
  const toolCalls = useConversationStore((state) => state.getMessageToolCalls(messageId));
  const memoryTraces = useConversationStore((state) => state.getMessageMemoryTraces(messageId));
  const sentences = useConversationStore((state) => state.getMessageSentences(messageId));

  const audioManager = useAudioManager();
  const currentlyPlayingId = useAudioStore((state) => state.playback.currentlyPlayingId);
  const isPlaying = useAudioStore((state) => state.playback.isPlaying);
  const playbackProgress = useAudioStore((state) => state.playback.playbackProgress);

  // Track audio state for each audio ref
  const [audioStates, setAudioStates] = useState<Record<string, AudioState>>({});

  // Find sentences with audio - computed before hooks to maintain consistent hook order
  const sentencesWithAudio = sentences.filter(s => s.audioRefId);

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

  // Convert tool calls to ToolData format for ChatBubble
  const tools: ToolData[] = toolCalls.map(toolCall => ({
    id: toolCall.id,
    name: toolCall.toolName,
    description: `Arguments: ${JSON.stringify(toolCall.arguments)}`,
    status: toolCall.status === 'pending' ? 'running' :
            toolCall.status === 'executing' ? 'running' :
            toolCall.status === 'success' ? 'completed' : 'error',
    result: toolCall.status === 'success' ? toolCall.resultContent :
            toolCall.status === 'error' ? toolCall.error : undefined,
  }));

  // Build addons for audio (memory traces handled separately via MemoryTraceAddon)
  const addons: MessageAddon[] = [];

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
      tooltip: 'Play audio response',
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
    <div className={className}>
      <ChatBubble
        type={MESSAGE_TYPES.ASSISTANT}
        content={message.content}
        state={MESSAGE_STATES.COMPLETED}
        timestamp={message.createdAt}
        addons={addons}
        tools={tools}
      />
      {memoryTraces.length > 0 && (
        <MemoryTraceAddon traces={memoryTraces} className="mt-2" />
      )}
    </div>
  );
};

export default AssistantMessage;
