import React, { useState, useEffect, useMemo, useCallback } from 'react';
import ChatBubble from '../molecules/ChatBubble';
import AudioAddon from '../atoms/AudioAddon';
import { useConversationStore, selectSentences } from '../../stores/conversationStore';
import { useAudioManager } from '../../hooks/useAudioManager';
import { useAudioStore } from '../../stores/audioStore';
import { useFeedback } from '../../hooks/useFeedback';
import { api } from '../../services/api';
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
  const currentConversationId = useConversationStore((state) => state.currentConversationId);
  const toolCallsMap = useConversationStore((state) => state.toolCalls);
  const memoryTracesMap = useConversationStore((state) => state.memoryTraces);
  const sentencesMap = useConversationStore(selectSentences);

  // Memoize derived data to avoid creating new arrays on every render
  const toolCalls = useMemo(() => {
    if (!message) return [];
    return message.toolCallIds
      .map(id => toolCallsMap[id])
      .filter(Boolean);
  }, [message, toolCallsMap]);

  const memoryTraces = useMemo(() => {
    if (!message) return [];
    return message.memoryTraceIds
      .map(id => memoryTracesMap[id])
      .filter(Boolean)
      .sort((a, b) => b.relevance - a.relevance);
  }, [message, memoryTracesMap]);

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

  // Feedback hook for message voting
  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('message', messageId);

  // Track audio state for each audio ref
  const [audioStates, setAudioStates] = useState<Record<string, AudioState>>({});

  // Find sentences with audio - memoized to avoid new array on every render
  const sentencesWithAudio = useMemo(
    () => sentences.filter(s => s.audioRefId),
    [sentences]
  );

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

  // Get the refresh action from conversation store
  const requestMessagesRefresh = useConversationStore((state) => state.requestMessagesRefresh);

  // Handle message edit - calls REST API to update assistant message content
  const handleEditMessage = useCallback(async (editedMessageId: MessageId, newContent: string) => {
    if (currentConversationId) {
      try {
        await api.editAssistantMessage(currentConversationId, editedMessageId, newContent);
        // The backend will update the message in place
        // A refresh may be needed to see the updated content
        requestMessagesRefresh();
      } catch (error) {
        console.error('Failed to edit assistant message:', error);
      }
    }
  }, [currentConversationId, requestMessagesRefresh]);

  // Handle branch switch - reloads messages after backend updates the conversation tip
  const handleBranchSwitch = useCallback(() => {
    // The branchStore already called switch-branch API to update the tip
    // Now we need to reload messages to reflect the new branch
    // Request a refresh which will be handled by App.tsx
    requestMessagesRefresh();
  }, [requestMessagesRefresh]);

  // Handle retry (regenerate) - calls REST API to regenerate the assistant response
  const handleRetry = useCallback(async (retryMessageId: MessageId) => {
    if (currentConversationId) {
      try {
        await api.regenerateResponse(currentConversationId, retryMessageId);
        // The backend will delete the old message and create a new one
        // A refresh is needed to see the new message
        requestMessagesRefresh();
      } catch (error) {
        console.error('Failed to regenerate response:', error);
      }
    }
  }, [currentConversationId, requestMessagesRefresh]);

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

  // Build addons for tools and audio (memory traces handled separately via MemoryTraceAddon)
  const addons: MessageAddon[] = [];

  // Add tool addons for each tool call
  tools.forEach(tool => {
    // Map tool names to appropriate emojis
    const getToolEmoji = (toolName: string) => {
      const name = toolName.toLowerCase();
      // Specific tool mappings
      if (name === 'memory_query' || name.includes('memory')) return '🧠';
      if (name === 'web_search' || name.includes('web')) return '🌐';
      // Generic mappings
      if (name.includes('search') || name.includes('find') || name.includes('query')) return '🔍';
      if (name.includes('calculate') || name.includes('math') || name.includes('compute')) return '🔢';
      if (name.includes('file') || name.includes('read') || name.includes('write')) return '📄';
      if (name.includes('code') || name.includes('execute') || name.includes('run')) return '⚙️';
      return '🔧'; // Default tool icon
    };

    addons.push({
      id: tool.id,
      type: 'tool',
      position: 'inline',
      emoji: getToolEmoji(tool.name),
      tooltip: tool.name,
    });
  });

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

  // Add memory traces as addon
  if (memoryTraces.length > 0) {
    addons.push({
      id: 'memory-traces',
      type: 'memory',
      position: 'inline',
      memoryData: memoryTraces.map(trace => ({
        id: trace.id,
        content: trace.content,
        relevance: trace.relevance,
      })),
    });
  }

  // Add feedback controls as addon
  addons.push({
    id: 'message-feedback',
    type: 'feedback',
    position: 'inline',
    feedbackData: {
      currentVote: currentVote as 'up' | 'down' | null,
      onVote: vote,
      upvotes: counts.up,
      downvotes: counts.down,
      isLoading: feedbackLoading,
    },
  });

  return (
    <div className={`flex flex-col items-start gap-2 ${className}`}>
      <ChatBubble
        type={MESSAGE_TYPES.ASSISTANT}
        content={message.content}
        state={MESSAGE_STATES.COMPLETED}
        timestamp={message.createdAt}
        addons={addons}
        tools={tools}
        messageId={messageId}
        conversationId={currentConversationId || undefined}
        onEditMessage={handleEditMessage}
        onBranchSwitch={handleBranchSwitch}
        onRetry={handleRetry}
        syncStatus={message.sync_status}
      />
    </div>
  );
};

export default AssistantMessage;
