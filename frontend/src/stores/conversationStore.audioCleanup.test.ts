import { describe, it, expect, beforeEach } from 'vitest';
import { useConversationStore } from './conversationStore';
import {
  createMessageId,
  createSentenceId,
  createAudioRefId,
  createConversationId,
  MessageStatus,
  type NormalizedMessage,
  type MessageSentence,
  type AudioRef,
} from '../types/streaming';

describe('conversationStore - audioRef cleanup in mergeMessages', () => {
  beforeEach(() => {
    useConversationStore.getState().clearConversation();
  });

  it('HYPOTHESIS #7: should clean up orphaned audioRefs when sentences are deleted during mergeMessages', () => {
    const conversationId = createConversationId('conv-1');

    // Step 1: Create a message with sentences that have audio
    const message1: NormalizedMessage = {
      id: createMessageId('msg-1'),
      conversationId,
      role: 'assistant',
      content: 'Hello world',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const sentence1: MessageSentence = {
      id: createSentenceId('sent-1'),
      messageId: message1.id,
      content: 'Hello',
      sequence: 0,
      audioRefId: createAudioRefId('audio-1'),
      isComplete: true,
    };

    const sentence2: MessageSentence = {
      id: createSentenceId('sent-2'),
      messageId: message1.id,
      content: 'world',
      sequence: 1,
      audioRefId: createAudioRefId('audio-2'),
      isComplete: true,
    };

    const audioRef1: AudioRef = {
      id: createAudioRefId('audio-1'),
      sizeBytes: 1024,
      durationMs: 500,
      sampleRate: 44100,
    };

    const audioRef2: AudioRef = {
      id: createAudioRefId('audio-2'),
      sizeBytes: 2048,
      durationMs: 800,
      sampleRate: 44100,
    };

    // Add initial state
    useConversationStore.getState().addMessage(message1);
    useConversationStore.getState().addSentence(sentence1);
    useConversationStore.getState().addSentence(sentence2);
    useConversationStore.getState().addAudioRef(audioRef1);
    useConversationStore.getState().addAudioRef(audioRef2);

    // Verify initial state
    let state = useConversationStore.getState();
    expect(Object.keys(state.messages)).toHaveLength(1);
    expect(Object.keys(state.sentences)).toHaveLength(2);
    expect(Object.keys(state.audioRefs)).toHaveLength(2);
    expect(state.sentences['sent-1'].audioRefId).toBe('audio-1');
    expect(state.sentences['sent-2'].audioRefId).toBe('audio-2');

    // Step 2: Call mergeMessages with an empty array (simulating all messages deleted)
    // This should remove the message, its sentences, AND the audioRefs
    useConversationStore.getState().mergeMessages(conversationId, []);

    // Step 3: Verify cleanup
    state = useConversationStore.getState();

    // Message should be deleted
    expect(Object.keys(state.messages)).toHaveLength(0);

    // Sentences should be deleted
    expect(Object.keys(state.sentences)).toHaveLength(0);

    // BUG: audioRefs should be deleted but they are NOT
    // This is the memory leak - orphaned audioRefs remain in the store
    expect(Object.keys(state.audioRefs)).toHaveLength(0); // WILL FAIL - audioRefs are not cleaned up
  });

  it('should clean up orphaned audioRefs when a specific message is removed during mergeMessages', () => {
    const conversationId = createConversationId('conv-1');

    // Create two messages - one will be kept, one will be removed
    const message1: NormalizedMessage = {
      id: createMessageId('msg-1'),
      conversationId,
      role: 'assistant',
      content: 'First message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const message2: NormalizedMessage = {
      id: createMessageId('msg-2'),
      conversationId,
      role: 'assistant',
      content: 'Second message',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const sentence1: MessageSentence = {
      id: createSentenceId('sent-1'),
      messageId: message1.id,
      content: 'First',
      sequence: 0,
      audioRefId: createAudioRefId('audio-1'),
      isComplete: true,
    };

    const sentence2: MessageSentence = {
      id: createSentenceId('sent-2'),
      messageId: message2.id,
      content: 'Second',
      sequence: 0,
      audioRefId: createAudioRefId('audio-2'),
      isComplete: true,
    };

    const audioRef1: AudioRef = {
      id: createAudioRefId('audio-1'),
      sizeBytes: 1024,
      durationMs: 500,
      sampleRate: 44100,
    };

    const audioRef2: AudioRef = {
      id: createAudioRefId('audio-2'),
      sizeBytes: 2048,
      durationMs: 800,
      sampleRate: 44100,
    };

    // Add initial state
    useConversationStore.getState().addMessage(message1);
    useConversationStore.getState().addMessage(message2);
    useConversationStore.getState().addSentence(sentence1);
    useConversationStore.getState().addSentence(sentence2);
    useConversationStore.getState().addAudioRef(audioRef1);
    useConversationStore.getState().addAudioRef(audioRef2);

    // Verify initial state
    let state = useConversationStore.getState();
    expect(Object.keys(state.messages)).toHaveLength(2);
    expect(Object.keys(state.sentences)).toHaveLength(2);
    expect(Object.keys(state.audioRefs)).toHaveLength(2);

    // Step 2: mergeMessages with only message1 (message2 is removed)
    useConversationStore.getState().mergeMessages(conversationId, [message1]);

    // Step 3: Verify cleanup
    state = useConversationStore.getState();

    // message1 should remain, message2 should be deleted
    expect(Object.keys(state.messages)).toHaveLength(1);
    expect(state.messages['msg-1']).toBeDefined();
    expect(state.messages['msg-2']).toBeUndefined();

    // sentence1 should remain, sentence2 should be deleted
    expect(Object.keys(state.sentences)).toHaveLength(1);
    expect(state.sentences['sent-1']).toBeDefined();
    expect(state.sentences['sent-2']).toBeUndefined();

    // BUG: audioRef1 should remain, audioRef2 should be deleted
    // But audioRef2 is NOT cleaned up (memory leak)
    expect(Object.keys(state.audioRefs)).toHaveLength(1);
    expect(state.audioRefs['audio-1']).toBeDefined();
    expect(state.audioRefs['audio-2']).toBeUndefined(); // WILL FAIL - audio-2 is not cleaned up
  });

  it('should handle sentences without audioRefs during cleanup', () => {
    const conversationId = createConversationId('conv-1');

    const message1: NormalizedMessage = {
      id: createMessageId('msg-1'),
      conversationId,
      role: 'assistant',
      content: 'Message without audio',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const sentence1: MessageSentence = {
      id: createSentenceId('sent-1'),
      messageId: message1.id,
      content: 'No audio here',
      sequence: 0,
      // No audioRefId
      isComplete: true,
    };

    useConversationStore.getState().addMessage(message1);
    useConversationStore.getState().addSentence(sentence1);

    let state = useConversationStore.getState();
    expect(Object.keys(state.messages)).toHaveLength(1);
    expect(Object.keys(state.sentences)).toHaveLength(1);
    expect(Object.keys(state.audioRefs)).toHaveLength(0);

    // Remove the message
    useConversationStore.getState().mergeMessages(conversationId, []);

    state = useConversationStore.getState();
    expect(Object.keys(state.messages)).toHaveLength(0);
    expect(Object.keys(state.sentences)).toHaveLength(0);
    expect(Object.keys(state.audioRefs)).toHaveLength(0);
  });

  it('should NOT delete shared audioRefs that are still referenced by other sentences', () => {
    const conversationId = createConversationId('conv-1');

    // Two messages sharing the same audioRef (edge case but theoretically possible)
    const message1: NormalizedMessage = {
      id: createMessageId('msg-1'),
      conversationId,
      role: 'assistant',
      content: 'First',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const message2: NormalizedMessage = {
      id: createMessageId('msg-2'),
      conversationId,
      role: 'assistant',
      content: 'Second',
      status: MessageStatus.Complete,
      createdAt: new Date(),
      sentenceIds: [],
      toolCallIds: [],
      memoryTraceIds: [],
    };

    const sharedAudioRefId = createAudioRefId('shared-audio');

    const sentence1: MessageSentence = {
      id: createSentenceId('sent-1'),
      messageId: message1.id,
      content: 'First',
      sequence: 0,
      audioRefId: sharedAudioRefId,
      isComplete: true,
    };

    const sentence2: MessageSentence = {
      id: createSentenceId('sent-2'),
      messageId: message2.id,
      content: 'Second',
      sequence: 0,
      audioRefId: sharedAudioRefId, // Same audio ref
      isComplete: true,
    };

    const sharedAudioRef: AudioRef = {
      id: sharedAudioRefId,
      sizeBytes: 1024,
      durationMs: 500,
      sampleRate: 44100,
    };

    useConversationStore.getState().addMessage(message1);
    useConversationStore.getState().addMessage(message2);
    useConversationStore.getState().addSentence(sentence1);
    useConversationStore.getState().addSentence(sentence2);
    useConversationStore.getState().addAudioRef(sharedAudioRef);

    let state = useConversationStore.getState();
    expect(Object.keys(state.audioRefs)).toHaveLength(1);

    // Remove only message1 (message2 still references the same audio)
    useConversationStore.getState().mergeMessages(conversationId, [message2]);

    state = useConversationStore.getState();

    // The shared audioRef should NOT be deleted because sentence2 still references it
    expect(Object.keys(state.audioRefs)).toHaveLength(1);
    expect(state.audioRefs['shared-audio']).toBeDefined();
  });
});
