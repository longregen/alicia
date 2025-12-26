import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { Message } from '../types/models';
import {
  ErrorMessage,
  ReasoningStep,
  ToolUseRequest,
  ToolUseResult,
  Acknowledgement,
  MemoryTrace,
  Commentary
} from '../types/protocol';

export interface StreamingMessage {
  id: string;
  role: 'assistant';
  content: string;
  isStreaming: true;
  sequence?: number;
}

export interface ToolUsage {
  request: ToolUseRequest;
  result: ToolUseResult | null;
}

interface MessageContextType {
  // Message state
  messages: Message[];
  streamingMessages: Map<string, string>; // sentence sequence -> content
  currentTranscription: string;
  isGenerating: boolean;
  currentGeneratingMessageId: string | null;

  // New protocol message states
  reasoningSteps: ReasoningStep[];
  toolUsages: ToolUsage[];
  errors: ErrorMessage[];
  memoryTraces: MemoryTrace[];
  commentaries: Commentary[];
  acknowledgements: Acknowledgement[];

  // Message actions
  setMessages: (messages: Message[] | ((prev: Message[]) => Message[])) => void;
  addMessage: (message: Message) => void;
  updateMessage: (id: string, updates: Partial<Message>) => void;
  clearMessages: () => void;
  mergeMessages: (newMessages: Message[]) => void;

  // Streaming actions (for LiveKit)
  updateStreamingSentence: (sequence: number, content: string) => void;
  clearStreamingSentences: () => void;
  finalizeStreamingMessage: (message: Message) => void;

  // Transcription actions
  setTranscription: (text: string) => void;
  clearTranscription: () => void;

  // Generation state actions
  setIsGenerating: (generating: boolean, messageId?: string) => void;

  // New protocol message actions
  addError: (error: ErrorMessage) => void;
  addReasoningStep: (step: ReasoningStep) => void;
  addToolUsage: (usage: ToolUsage) => void;
  updateToolUsageResult: (result: ToolUseResult) => void;
  handleAcknowledgement: (ack: Acknowledgement) => void;
  addMemoryTrace: (trace: MemoryTrace) => void;
  addCommentary: (commentary: Commentary) => void;
  clearProtocolMessages: () => void;
}

const MessageContext = createContext<MessageContextType | null>(null);

export function MessageProvider({ children }: { children: ReactNode }) {
  const [messages, setMessagesState] = useState<Message[]>([]);
  const [streamingMessages, setStreamingMessages] = useState<Map<string, string>>(new Map());
  const [currentTranscription, setCurrentTranscription] = useState<string>('');
  const [isGenerating, setIsGeneratingState] = useState<boolean>(false);
  const [currentGeneratingMessageId, setCurrentGeneratingMessageId] = useState<string | null>(null);

  // New protocol message states
  const [reasoningSteps, setReasoningSteps] = useState<ReasoningStep[]>([]);
  const [toolUsages, setToolUsages] = useState<ToolUsage[]>([]);
  const [errors, setErrors] = useState<ErrorMessage[]>([]);
  const [memoryTraces, setMemoryTraces] = useState<MemoryTrace[]>([]);
  const [commentaries, setCommentaries] = useState<Commentary[]>([]);
  const [acknowledgements, setAcknowledgements] = useState<Acknowledgement[]>([]);

  const setMessages = useCallback((newMessages: Message[] | ((prev: Message[]) => Message[])) => {
    if (typeof newMessages === 'function') {
      setMessagesState(newMessages);
    } else {
      setMessagesState(newMessages);
    }
  }, []);

  const addMessage = useCallback((message: Message) => {
    setMessagesState(prev => {
      // Avoid duplicates - check by id
      if (prev.some(m => m.id === message.id)) {
        return prev;
      }
      // Also check by local_id if the incoming message has one
      if (message.local_id && prev.some(m => m.local_id === message.local_id)) {
        return prev;
      }
      // Check for content match on very recent messages (within 5 seconds)
      // to catch duplicates from different sources with different IDs
      const hasRecentDuplicate = prev.some(m => {
        if (m.role !== message.role || m.contents !== message.contents) return false;
        return Date.now() - new Date(m.created_at).getTime() < 5000;
      });
      if (hasRecentDuplicate) {
        return prev;
      }
      return [...prev, message];
    });
  }, []);

  const updateMessage = useCallback((id: string, updates: Partial<Message>) => {
    setMessagesState(prev =>
      prev.map(msg => msg.id === id ? { ...msg, ...updates } : msg)
    );
  }, []);

  const clearMessages = useCallback(() => {
    setMessagesState([]);
    setStreamingMessages(new Map());
    setCurrentTranscription('');
    setIsGeneratingState(false);
    setCurrentGeneratingMessageId(null);
    setReasoningSteps([]);
    setToolUsages([]);
    setErrors([]);
    setMemoryTraces([]);
    setCommentaries([]);
    setAcknowledgements([]);
  }, []);

  const mergeMessages = useCallback((newMessages: Message[]) => {
    setMessagesState(prev => {
      // Create sets for deduplication - check by id and local_id
      const existingIds = new Set(prev.map(m => m.id));
      const existingLocalIds = new Set(prev.filter(m => m.local_id).map(m => m.local_id));

      // Also create a content+role fingerprint for messages within last 5 seconds
      // to catch duplicates from different sources with different IDs
      const recentFingerprints = new Set(
        prev
          .filter(m => Date.now() - new Date(m.created_at).getTime() < 5000)
          .map(m => `${m.role}:${m.contents}`)
      );

      // Filter out messages that already exist (by id, local_id, or recent content match)
      const uniqueNewMessages = newMessages.filter(m => {
        // Check by server id
        if (existingIds.has(m.id)) return false;
        // Check by local_id if present
        if (m.local_id && existingLocalIds.has(m.local_id)) return false;
        // Check by content fingerprint for very recent messages
        const fingerprint = `${m.role}:${m.contents}`;
        if (recentFingerprints.has(fingerprint)) return false;
        return true;
      });

      // If no new messages, return previous state
      if (uniqueNewMessages.length === 0) {
        return prev;
      }

      // Merge and sort by sequence number to maintain order
      const merged = [...prev, ...uniqueNewMessages].sort((a, b) => {
        // Sort by sequence_number if available
        if (a.sequence_number !== undefined && b.sequence_number !== undefined) {
          return a.sequence_number - b.sequence_number;
        }
        // Fallback to created_at timestamp
        return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
      });

      return merged;
    });
  }, []);

  const updateStreamingSentence = useCallback((sequence: number, content: string) => {
    setStreamingMessages(prev => {
      const newMap = new Map(prev);
      newMap.set(String(sequence), content);
      return newMap;
    });
  }, []);

  const clearStreamingSentences = useCallback(() => {
    setStreamingMessages(new Map());
  }, []);

  const finalizeStreamingMessage = useCallback((message: Message) => {
    // Add the finalized message and clear streaming state
    addMessage(message);
    clearStreamingSentences();
    setIsGeneratingState(false);
    setCurrentGeneratingMessageId(null);
  }, [addMessage, clearStreamingSentences]);

  const setTranscription = useCallback((text: string) => {
    setCurrentTranscription(text);
  }, []);

  const clearTranscription = useCallback(() => {
    setCurrentTranscription('');
  }, []);

  const setIsGenerating = useCallback((generating: boolean, messageId?: string) => {
    setIsGeneratingState(generating);
    setCurrentGeneratingMessageId(messageId || null);
  }, []);

  // New protocol message handlers
  const addError = useCallback((error: ErrorMessage) => {
    setErrors(prev => [...prev, error]);
  }, []);

  const addReasoningStep = useCallback((step: ReasoningStep) => {
    setReasoningSteps(prev => {
      // Avoid duplicates
      if (prev.some(s => s.id === step.id)) {
        return prev;
      }
      // Sort by sequence to maintain order
      return [...prev, step].sort((a, b) => a.sequence - b.sequence);
    });
  }, []);

  const addToolUsage = useCallback((usage: ToolUsage) => {
    setToolUsages(prev => [...prev, usage]);
  }, []);

  const updateToolUsageResult = useCallback((result: ToolUseResult) => {
    setToolUsages(prev =>
      prev.map(usage =>
        usage.request.id === result.requestId
          ? { ...usage, result }
          : usage
      )
    );
  }, []);

  const handleAcknowledgement = useCallback((ack: Acknowledgement) => {
    setAcknowledgements(prev => [...prev, ack]);
  }, []);

  const addMemoryTrace = useCallback((trace: MemoryTrace) => {
    setMemoryTraces(prev => {
      // Avoid duplicates
      if (prev.some(t => t.id === trace.id)) {
        return prev;
      }
      return [...prev, trace];
    });
  }, []);

  const addCommentary = useCallback((commentary: Commentary) => {
    setCommentaries(prev => {
      // Avoid duplicates
      if (prev.some(c => c.id === commentary.id)) {
        return prev;
      }
      return [...prev, commentary];
    });
  }, []);

  const clearProtocolMessages = useCallback(() => {
    setReasoningSteps([]);
    setToolUsages([]);
    setErrors([]);
    setMemoryTraces([]);
    setCommentaries([]);
    setAcknowledgements([]);
  }, []);

  const value: MessageContextType = {
    messages,
    streamingMessages,
    currentTranscription,
    isGenerating,
    currentGeneratingMessageId,
    reasoningSteps,
    toolUsages,
    errors,
    memoryTraces,
    commentaries,
    acknowledgements,
    setMessages,
    addMessage,
    updateMessage,
    clearMessages,
    mergeMessages,
    updateStreamingSentence,
    clearStreamingSentences,
    finalizeStreamingMessage,
    setTranscription,
    clearTranscription,
    setIsGenerating,
    addError,
    addReasoningStep,
    addToolUsage,
    updateToolUsageResult,
    handleAcknowledgement,
    addMemoryTrace,
    addCommentary,
    clearProtocolMessages,
  };

  return (
    <MessageContext.Provider value={value}>
      {children}
    </MessageContext.Provider>
  );
}

export function useMessageContext() {
  const context = useContext(MessageContext);
  if (!context) {
    throw new Error('useMessageContext must be used within MessageProvider');
  }
  return context;
}
