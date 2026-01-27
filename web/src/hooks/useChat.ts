import { useEffect, useCallback, useRef, useState, useMemo } from 'react';
import { api, ToolUse, MemoryUse } from '../services/api';
import {
  useChatStore,
  selectConversationMessages,
  selectConversationTipMessageId,
  selectConversationStreamingMessage,
  selectConversationActiveBranch,
} from '../stores/chatStore';
import { useWebSocket } from '../contexts/WebSocketContext';
import {
  ChatMessage,
  MessageId,
  ToolCall,
  MemoryTrace,
  createMessageId,
  createConversationId,
  createToolCallId,
  createMemoryTraceId,
  createEmptyMessage,
} from '../types/chat';
import { Message } from '../types/models';
import { withSpanAsync, addSpanEvent, setSpanAttributes } from '../lib/otel';

function convertToolUse(toolUse: ToolUse): ToolCall {
  return {
    id: createToolCallId(toolUse.id),
    tool_name: toolUse.tool_name,
    arguments: toolUse.arguments,
    result: toolUse.result,
    status: toolUse.status,
    error: toolUse.error,
    created_at: toolUse.created_at,
  };
}

function convertMemoryUse(memoryUse: MemoryUse): MemoryTrace {
  return {
    id: createMemoryTraceId(memoryUse.id),
    memory_id: memoryUse.memory_id,
    content: memoryUse.content,
    relevance: memoryUse.similarity,
  };
}

function convertApiMessage(
  msg: Message,
  toolUses: ToolUse[] = [],
  memoryUses: MemoryUse[] = []
): ChatMessage {
  return {
    id: createMessageId(msg.id),
    conversation_id: createConversationId(msg.conversation_id),
    role: msg.role,
    content: msg.content,
    reasoning: msg.reasoning,
    branch_index: msg.branch_index,
    status: msg.status,
    created_at: msg.created_at,
    previous_id: msg.previous_id ? createMessageId(msg.previous_id) : undefined,
    thinking: [],
    reasoning_steps: [],
    tool_calls: toolUses.map(convertToolUse),
    memory_traces: memoryUses.map(convertMemoryUse),
  };
}

async function fetchConversationMessages(conversationId: string): Promise<ChatMessage[]> {
  const serverMessages = await api.getMessages(conversationId);

  const assistantMessages = serverMessages.filter((m) => m.role === 'assistant');
  const [toolUsesResults, memoryUsesResults] = await Promise.all([
    Promise.all(assistantMessages.map((m) => api.getToolUsesByMessage(m.id).catch(() => [] as ToolUse[]))),
    Promise.all(assistantMessages.map((m) => api.getMemoryUsesByMessage(m.id).catch(() => [] as MemoryUse[]))),
  ]);

  const toolUsesByMessage = new Map<string, ToolUse[]>();
  const memoryUsesByMessage = new Map<string, MemoryUse[]>();
  assistantMessages.forEach((msg, index) => {
    toolUsesByMessage.set(msg.id, toolUsesResults[index]);
    memoryUsesByMessage.set(msg.id, memoryUsesResults[index]);
  });

  return serverMessages.map((msg) =>
    convertApiMessage(msg, toolUsesByMessage.get(msg.id) || [], memoryUsesByMessage.get(msg.id) || [])
  );
}

export function useChat(conversationId: string | null) {
  const convId = conversationId ? createConversationId(conversationId) : null;

  const setActiveConversation = useChatStore((s) => s.setActiveConversation);
  const setMessages = useChatStore((s) => s.setMessages);
  const setTipMessageId = useChatStore((s) => s.setTipMessageId);
  const addMessage = useChatStore((s) => s.addMessage);

  const messagesSelector = useMemo(() => selectConversationMessages(convId), [convId]);
  const tipMessageIdSelector = useMemo(() => selectConversationTipMessageId(convId), [convId]);
  const streamingMessageSelector = useMemo(
    () => selectConversationStreamingMessage(convId),
    [convId]
  );
  const activeBranchSelector = useMemo(() => selectConversationActiveBranch(convId), [convId]);

  const messages = useChatStore(messagesSelector);
  const tipMessageId = useChatStore(tipMessageIdSelector);
  const streamingMessage = useChatStore(streamingMessageSelector);
  const activeBranch = useChatStore(activeBranchSelector);

  const { subscribe, unsubscribe, isConnected } = useWebSocket();
  const [loading, setLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const prevConversationRef = useRef<string | null>(null);

  useEffect(() => {
    if (conversationId === prevConversationRef.current) return;
    const prevConversation = prevConversationRef.current;
    prevConversationRef.current = conversationId;

    if (prevConversation) {
      unsubscribe(prevConversation);
    }

    if (!conversationId) {
      setActiveConversation(null);
      return;
    }

    const switchConvId = createConversationId(conversationId);
    setActiveConversation(switchConvId);

    setLoading(true);
    setError(null);

    Promise.all([fetchConversationMessages(conversationId), api.getConversation(conversationId)])
      .then(([chatMessages, conversation]) => {
        setMessages(switchConvId, chatMessages);

        // Always set tipMessageId from conversation (single source of truth)
        if (conversation.tip_message_id) {
          setTipMessageId(switchConvId, createMessageId(conversation.tip_message_id));
        } else {
          setTipMessageId(switchConvId, null);
        }
      })
      .catch((err) => {
        setError(err.message || 'Failed to load messages');
      })
      .finally(() => {
        setLoading(false);
      });

    return () => {
      if (conversationId) {
        unsubscribe(conversationId);
      }
    };
  }, [conversationId, setActiveConversation, setMessages, setTipMessageId, unsubscribe]);

  useEffect(() => {
    if (conversationId && isConnected) {
      subscribe(conversationId).catch((err) => {
        console.error('Failed to subscribe to conversation:', err);
      });
    }
  }, [isConnected, conversationId, subscribe]);

  const findLeafTip = useCallback(
    (messageId: MessageId): MessageId => {
      if (!messages) return messageId;
      const messageArray = Array.from(messages.values());

      const childrenOf = new Map<string, ChatMessage[]>();
      for (const msg of messageArray) {
        if (msg.previous_id) {
          const children = childrenOf.get(msg.previous_id) || [];
          children.push(msg);
          childrenOf.set(msg.previous_id, children);
        }
      }

      const descendants: ChatMessage[] = [];
      const queue = [messageId as string];

      while (queue.length > 0) {
        const current = queue.shift()!;
        const children = childrenOf.get(current) || [];
        for (const child of children) {
          descendants.push(child);
          queue.push(child.id);
        }
      }

      const leaves = descendants.filter(
        (d) => !childrenOf.has(d.id) || childrenOf.get(d.id)!.length === 0
      );

      if (leaves.length === 0) {
        return messageId;
      }

      const newest = leaves.reduce((a, b) =>
        new Date(a.created_at).getTime() > new Date(b.created_at).getTime() ? a : b
      );
      return newest.id;
    },
    [messages]
  );

  const sendMessage = useCallback(
    async (content: string): Promise<boolean> => {
      if (!conversationId || !convId || !content.trim()) return false;

      return withSpanAsync(
        'user.send_message',
        async () => {
          setSpanAttributes({
            'conversation.id': conversationId,
            'message.content_length': content.trim().length,
          });

          setSending(true);

          try {
            // Prevents race conditions with WebSocket messages that reference server IDs
            const serverMessage = await api.sendMessage(conversationId, {
              content: content,
              previous_id: tipMessageId || undefined,
              use_pareto: true,
            });

            const serverId = createMessageId(serverMessage.id);

            const confirmedMessage = createEmptyMessage(serverId, convId, 'user');
            confirmedMessage.content = content.trim();
            confirmedMessage.status = 'completed';
            confirmedMessage.previous_id = tipMessageId || undefined;
            confirmedMessage.created_at = serverMessage.created_at || new Date().toISOString();

            addMessage(convId, confirmedMessage);
            setTipMessageId(convId, serverId);
            addSpanEvent('message_confirmed', { 'message.server_id': serverMessage.id });

            return true;
          } catch (error) {
            addSpanEvent('message_failed');
            throw error; // Re-throw to let withSpanAsync record the error
          } finally {
            setSending(false);
          }
        },
        {
          'conversation.id': conversationId,
        }
      ).catch(() => false); // Convert thrown errors to false return
    },
    [conversationId, convId, tipMessageId, addMessage, setTipMessageId]
  );

  const switchBranch = useCallback(
    (targetMessageId: MessageId) => {
      if (!convId) return;
      const leafTipId = findLeafTip(targetMessageId);
      setTipMessageId(convId, leafTipId);
    },
    [convId, findLeafTip, setTipMessageId]
  );

  const refetch = useCallback(async () => {
    if (!conversationId || !convId) return;

    setLoading(true);
    await withSpanAsync(
      'chat.refetch_messages',
      async () => {
        const chatMessages = await fetchConversationMessages(conversationId);
        setMessages(convId, chatMessages);
        addSpanEvent('messages_loaded', { 'message.count': chatMessages.length });
      },
      { 'conversation.id': conversationId }
    ).catch(() => {
      setError('Failed to refresh messages');
    }).finally(() => {
      setLoading(false);
    });
  }, [conversationId, convId, setMessages]);

  return {
    messages: activeBranch,
    streamingMessage,
    isStreaming: !!streamingMessage,
    loading,
    sending,
    error,
    isConnected,
    sendMessage,
    switchBranch,
    refetch,
  };
}
