import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import {
  ErrorMessage,
  ToolUseRequest,
  ToolUseResult,
  Acknowledgement,
  MemoryTrace,
} from '../types/protocol';

export interface ToolUsage {
  request: ToolUseRequest;
  result: ToolUseResult | null;
}

interface MessageContextType {
  streamingMessages: Map<string, string>;
  currentTranscription: string;
  isGenerating: boolean;
  currentGeneratingMessageId: string | null;

  toolUsages: ToolUsage[];
  errors: ErrorMessage[];
  memoryTraces: MemoryTrace[];
  acknowledgements: Acknowledgement[];

  messages: never[];

  updateStreamingSentence: (sequence: number, content: string) => void;
  clearStreamingSentences: () => void;
  finalizeStreamingMessage: (message: unknown) => void;
  addMessage: (message: unknown) => void;
  setTranscription: (text: string) => void;
  clearTranscription: () => void;
  setIsGenerating: (generating: boolean, messageId?: string) => void;
  addError: (error: ErrorMessage) => void;
  addToolUsage: (usage: ToolUsage) => void;
  updateToolUsageResult: (result: ToolUseResult) => void;
  handleAcknowledgement: (ack: Acknowledgement) => void;
  addMemoryTrace: (trace: MemoryTrace) => void;
  clearProtocolMessages: () => void;
  clearMessages: () => void;
}

const MessageContext = createContext<MessageContextType | null>(null);

export function MessageProvider({ children }: { children: ReactNode }) {
  const [streamingMessages, setStreamingMessages] = useState<Map<string, string>>(new Map());
  const [currentTranscription, setCurrentTranscription] = useState<string>('');
  const [isGenerating, setIsGeneratingState] = useState<boolean>(false);
  const [currentGeneratingMessageId, setCurrentGeneratingMessageId] = useState<string | null>(null);

  const [toolUsages, setToolUsages] = useState<ToolUsage[]>([]);
  const [errors, setErrors] = useState<ErrorMessage[]>([]);
  const [memoryTraces, setMemoryTraces] = useState<MemoryTrace[]>([]);
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

  const finalizeStreamingMessage = useCallback((_message: unknown) => {
    clearStreamingSentences();
    setIsGeneratingState(false);
    setCurrentGeneratingMessageId(null);
  }, [clearStreamingSentences]);

  const addMessage = useCallback((_message: unknown) => {
    // No-op - messages are managed by store
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

  const addError = useCallback((error: ErrorMessage) => {
    setErrors(prev => [...prev, error]);
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
      if (prev.some(t => t.memoryId === trace.memoryId)) {
        return prev;
      }
      return [...prev, trace];
    });
  }, []);

  const clearProtocolMessages = useCallback(() => {
    setToolUsages([]);
    setErrors([]);
    setMemoryTraces([]);
    setAcknowledgements([]);
  }, []);

  const clearMessages = useCallback(() => {
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
    toolUsages,
    errors,
    memoryTraces,
    acknowledgements,
    messages: [],
    updateStreamingSentence,
    clearStreamingSentences,
    finalizeStreamingMessage,
    addMessage,
    setTranscription,
    clearTranscription,
    setIsGenerating,
    addError,
    addToolUsage,
    updateToolUsageResult,
    handleAcknowledgement,
    addMemoryTrace,
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
