import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
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
  // Streaming state (used for real-time message streaming from any source)
  streamingMessages: Map<string, string>; // sentence sequence -> content
  currentTranscription: string;
  isGenerating: boolean;
  currentGeneratingMessageId: string | null;

  // Protocol message states
  reasoningSteps: ReasoningStep[];
  toolUsages: ToolUsage[];
  errors: ErrorMessage[];
  memoryTraces: MemoryTrace[];
  commentaries: Commentary[];
  acknowledgements: Acknowledgement[];

  // Legacy compatibility field - always empty, use store for actual messages
  messages: never[];

  // Streaming actions
  updateStreamingSentence: (sequence: number, content: string) => void;
  clearStreamingSentences: () => void;
  finalizeStreamingMessage: (message: unknown) => void; // Streaming finalization handler

  // Message actions
  addMessage: (message: unknown) => void; // Message addition handler

  // Transcription actions
  setTranscription: (text: string) => void; // Transcription handler
  clearTranscription: () => void;

  // Generation state actions
  setIsGenerating: (generating: boolean, messageId?: string) => void;

  // Protocol message actions
  addError: (error: ErrorMessage) => void;
  addReasoningStep: (step: ReasoningStep) => void;
  addToolUsage: (usage: ToolUsage) => void;
  updateToolUsageResult: (result: ToolUseResult) => void;
  handleAcknowledgement: (ack: Acknowledgement) => void;
  addMemoryTrace: (trace: MemoryTrace) => void;
  addCommentary: (commentary: Commentary) => void;
  clearProtocolMessages: () => void;
  clearMessages: () => void; // For compatibility - clears protocol messages only
}

const MessageContext = createContext<MessageContextType | null>(null);

export function MessageProvider({ children }: { children: ReactNode }) {
  const [streamingMessages, setStreamingMessages] = useState<Map<string, string>>(new Map());
  const [currentTranscription, setCurrentTranscription] = useState<string>('');
  const [isGenerating, setIsGeneratingState] = useState<boolean>(false);
  const [currentGeneratingMessageId, setCurrentGeneratingMessageId] = useState<string | null>(null);

  // Protocol message states
  const [reasoningSteps, setReasoningSteps] = useState<ReasoningStep[]>([]);
  const [toolUsages, setToolUsages] = useState<ToolUsage[]>([]);
  const [errors, setErrors] = useState<ErrorMessage[]>([]);
  const [memoryTraces, setMemoryTraces] = useState<MemoryTrace[]>([]);
  const [commentaries, setCommentaries] = useState<Commentary[]>([]);
  const [acknowledgements, setAcknowledgements] = useState<Acknowledgement[]>([]);

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

  // No-op: Message operations are handled by the store, not context
  // These exist for interface compatibility with legacy components
  const finalizeStreamingMessage = useCallback((_message: unknown) => {
    clearStreamingSentences();
    setIsGeneratingState(false);
    setCurrentGeneratingMessageId(null);
  }, [clearStreamingSentences]);

  const addMessage = useCallback((_message: unknown) => {
    // No-op
  }, []);

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

  // Protocol message handlers
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

  const clearMessages = useCallback(() => {
    // For compatibility - clear all transient state
    setStreamingMessages(new Map());
    setCurrentTranscription('');
    setIsGeneratingState(false);
    setCurrentGeneratingMessageId(null);
    clearProtocolMessages();
  }, [clearProtocolMessages]);

  const value: MessageContextType = {
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
    messages: [], // Empty array for compatibility
    updateStreamingSentence,
    clearStreamingSentences,
    finalizeStreamingMessage,
    addMessage,
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
    clearMessages,
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
