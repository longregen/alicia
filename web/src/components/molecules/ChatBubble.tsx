import React, { useState, useCallback, useMemo } from 'react';
import MessageBubble from '../atoms/MessageBubble';
import ComplexAddons, { type ToolDetail } from '../atoms/ComplexAddons';
import FeedbackControls from '../atoms/FeedbackControls';
import { Textarea } from '../atoms/Textarea';
import Button from '../atoms/Button';
const MESSAGE_TYPES = { USER: 'user', ASSISTANT: 'assistant' } as const;
const MESSAGE_STATES = { STREAMING: 'streaming', COMPLETED: 'completed' } as const;
import { cls } from '../../utils/cls';
import { useFeedback } from '../../hooks/useFeedback';
import { useChatStore } from '../../stores/chatStore';
import { useVoiceConnectionStore } from '../../stores/voiceConnectionStore';
import type {
  BaseComponentProps,
  MessageRole,
  MessageState,
  MessageAddon,
} from '../../types/components';
import type { MessageId, ConversationId, ThinkingEntry, ReasoningEntry } from '../../types/chat';

interface ReasoningBlockProps {
  content: string;
  keyId: number;
  id?: string;
}

const ReasoningBlock: React.FC<ReasoningBlockProps> = ({ content, keyId, id }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('reasoning', id || '');

  const previewContent = content.length > 100 ? content.slice(0, 100) + '...' : content;
  const hasMore = content.length > 100;

  return (
    <div
      key={`reasoning-${keyId}`}
      className="my-2 p-3 bg-reasoning border-l-4 border-accent rounded-r-lg"
    >
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className={cls(
          'w-full layout-between',
          'text-xs text-reasoning font-medium mb-1',
          'hover:text-accent transition-colors'
        )}
        aria-expanded={isExpanded}
        aria-label={isExpanded ? 'Collapse reasoning' : 'Expand reasoning'}
      >
        <span>Reasoning</span>
        <svg
          className={cls(
            'w-4 h-4 transition-transform duration-200',
            isExpanded ? 'rotate-180' : 'rotate-0'
          )}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      <div className="text-sm text-default whitespace-pre-wrap">
        {isExpanded || !hasMore ? content : previewContent}
      </div>
      {hasMore && !isExpanded && (
        <button
          onClick={() => setIsExpanded(true)}
          className="text-xs text-accent hover:text-accent-hover mt-1"
        >
          Show more
        </button>
      )}
      {/* Feedback controls - only show when expanded and id is available */}
      {isExpanded && id && (
        <div className="mt-2 pt-2 border-t border-muted">
          <div className="text-xs text-muted mb-1">Was this reasoning helpful?</div>
          <FeedbackControls
            currentVote={currentVote as 'up' | 'down' | null}
            onVote={vote}
            upvotes={counts.up}
            downvotes={counts.down}
            isLoading={feedbackLoading}
            compact
          />
        </div>
      )}
    </div>
  );
};

export interface ChatBubbleProps extends BaseComponentProps {
  /** Type of message (user, assistant, system) */
  type?: MessageRole;
  /** Message content text */
  content?: string;
  /** Current message state */
  state?: MessageState;
  /** Message timestamp */
  timestamp?: Date;
  /** Whether to show typing indicator */
  showTyping?: boolean;
  /** Streaming text content */
  streamingText?: string;
  /** Array of addons to display with the message */
  addons?: MessageAddon[];
  /** Array of tool details attached to the message */
  toolDetails?: ToolDetail[];
  /** Message ID for branch tracking */
  messageId?: MessageId;
  /** Conversation ID for branch API calls */
  conversationId?: string;
  /** Callback when user edits a message - triggers new agent response */
  onEditMessage?: (messageId: MessageId, newContent: string) => void;
  /** Callback when user navigates to a different branch - triggers reload of messages */
  onBranchSwitch?: (targetMessageId: string) => void;
  /** Callback when user clicks "Continue from here" on an assistant message */
  onContinueFromHere?: (messageId: MessageId) => void;
  /** Callback when user clicks "Retry" to regenerate an assistant message */
  onRetry?: (messageId: MessageId) => void;
  /** Structured thinking entries */
  thinkingEntries?: ThinkingEntry[];
  /** Structured reasoning steps */
  reasoningSteps?: ReasoningEntry[];
}

const ChatBubble: React.FC<ChatBubbleProps> = ({
  type = MESSAGE_TYPES.USER,
  content = '',
  state = MESSAGE_STATES.COMPLETED,
  timestamp = new Date(),
  showTyping = false,
  streamingText = '',
  addons = [],
  toolDetails = [],
  messageId,
  conversationId,
  onEditMessage,
  onBranchSwitch,
  onContinueFromHere,
  onRetry,
  thinkingEntries = [],
  reasoningSteps = [],
  className = ''
}) => {
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [editedContent, setEditedContent] = useState<string>('');

  const branchKey = useChatStore((state) => {
    if (!conversationId || !messageId) return '';
    const messages = state.conversations.get(conversationId as ConversationId)?.messages;
    if (!messages) return '';
    const msg = messages.get(messageId);
    if (!msg?.previous_id) return '';
    const siblings: string[] = [];
    for (const [, m] of messages) {
      if (m.previous_id === msg.previous_id) {
        siblings.push(m.id as string);
      }
    }
    if (siblings.length <= 1) return '';
    siblings.sort((a, b) => {
      const ma = messages.get(a as MessageId);
      const mb = messages.get(b as MessageId);
      return (ma?.branch_index ?? 0) - (mb?.branch_index ?? 0);
    });
    return siblings.join('|');
  });

  const { siblingCount, currentIndex, siblingIds } = useMemo(() => {
    if (!branchKey) return { siblingCount: 0, currentIndex: 0, siblingIds: [] as string[] };
    const ids = branchKey.split('|');
    return {
      siblingCount: ids.length,
      currentIndex: ids.indexOf(messageId as string),
      siblingIds: ids,
    };
  }, [branchKey, messageId]);

  const messageStatus = useChatStore((state) => {
    if (!conversationId || !messageId) return undefined;
    return state.conversations.get(conversationId as ConversationId)?.messages?.get(messageId)?.status;
  });

  const isVoiceSpeaking = useVoiceConnectionStore((state) => {
    if (type !== MESSAGE_TYPES.ASSISTANT || !messageId) return false;
    return state.speakingState.speaking && state.speakingState.messageId === messageId;
  });

  const getContentToDisplay = (): React.ReactNode => {
    const isStreaming = state === MESSAGE_STATES.STREAMING;
    const rawText = isStreaming ? (streamingText || content) : content;
    let text = rawText;

    if (isStreaming) {
      text = text.replace(/<[^>]*$/, '');
      text = text.replace(/<\/[^>]*$/, '');
    }

    // Strip embedded reasoning XML tags — structured props replace them
    text = text.replace(/<reasoning[^>]*>[\s\S]*?<\/reasoning>/g, '');

    const hasThinking = thinkingEntries.length > 0;
    const hasReasoning = reasoningSteps.length > 0;

    // Simple case: just text, let MessageBubble handle markdown
    if (!hasThinking && !hasReasoning) {
      return text;
    }

    // Build parts with thinking/reasoning blocks around text
    const sortedReasoning = [...reasoningSteps].sort((a, b) => a.sequence - b.sequence);
    const parts: React.ReactNode[] = [];
    let keyIndex = 0;

    for (const entry of thinkingEntries) {
      parts.push(
        <div key={`thinking-summary-${keyIndex++}`} className="text-xs text-muted italic mb-2 whitespace-pre-line">
          {entry.content}
        </div>
      );
    }

    if (text.trim()) {
      parts.push(<span key={`text-${keyIndex++}`}>{text}</span>);
    }

    for (const step of sortedReasoning) {
      parts.push(
        <ReasoningBlock key={`reasoning-${keyIndex}`} content={step.content} keyId={keyIndex++} id={step.id} />
      );
    }

    return parts;
  };

  const handleEditClick = () => {
    setEditedContent(content);
    setIsEditing(true);
  };

  const handleSaveEdit = () => {
    if (messageId && editedContent !== content && onEditMessage) {
      onEditMessage(messageId, editedContent);
    }
    setIsEditing(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditedContent('');
  };

  const handleNavigateBranch = useCallback((direction: 'prev' | 'next') => {
    if (!siblingIds || siblingIds.length <= 1) return;

    const targetIndex = direction === 'prev'
      ? (currentIndex - 1 + siblingIds.length) % siblingIds.length
      : (currentIndex + 1) % siblingIds.length;

    const targetId = siblingIds[targetIndex];
    if (targetId && onBranchSwitch) {
      onBranchSwitch(targetId);
    }
  }, [siblingIds, currentIndex, onBranchSwitch]);

  const inlineAddons = addons.filter(addon => addon.position === 'inline' || !addon.position);
  const belowAddons = addons.filter(addon => addon.position === 'below');

  const roleClass = type === MESSAGE_TYPES.USER ? 'user' : type === MESSAGE_TYPES.ASSISTANT ? 'assistant' : 'system';
  const isUser = type === MESSAGE_TYPES.USER;
  const canEdit = state !== MESSAGE_STATES.STREAMING && messageStatus !== 'error';

  return (
    <div className={cls('flex flex-col gap-1', roleClass, className)}>
      {isEditing ? (
        <div className="message-max-width">
          <Textarea
            value={editedContent}
            onChange={(e) => setEditedContent(e.target.value)}
            className="w-full mb-2 min-h-24"
            autoFocus
          />
          <div className="flex gap-2 justify-end">
            <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
              Cancel
            </Button>
            <Button variant="default" size="sm" onClick={handleSaveEdit}>
              Save
            </Button>
          </div>
        </div>
      ) : (
        <div className="relative group/bubble">
          <MessageBubble
            type={type}
            content={getContentToDisplay()}
            state={state}
            timestamp={timestamp}
            showTyping={showTyping}
            addons={[]}
            hideTimestamp={true}
          />
          {canEdit && (
            <div className={cls(
              'absolute top-1 flex gap-0.5',
              'opacity-0 group-hover/bubble:opacity-100 transition-opacity',
              isUser ? '-left-8' : '-right-8'
            )}>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={handleEditClick}
                className="h-6 w-6 text-muted-foreground hover:text-foreground"
                aria-label="Edit message"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </Button>
              {type === MESSAGE_TYPES.ASSISTANT && messageId && onRetry && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onRetry(messageId)}
                  className="h-6 w-6 text-muted-foreground hover:text-foreground"
                  aria-label="Retry (regenerate response)"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </Button>
              )}
              {type === MESSAGE_TYPES.ASSISTANT && messageId && onContinueFromHere && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onContinueFromHere(messageId)}
                  className="h-6 w-6 text-muted-foreground hover:text-foreground"
                  aria-label="Continue from here"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </Button>
              )}
            </div>
          )}
          {isVoiceSpeaking && (
            <div className="flex items-center gap-1.5 mt-1 px-2 text-xs text-accent" aria-label="Speaking">
              <svg className="w-3.5 h-3.5 animate-pulse" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 3a1 1 0 0 1 1 1v16a1 1 0 1 1-2 0V4a1 1 0 0 1 1-1zM6 8a1 1 0 0 1 1 1v6a1 1 0 1 1-2 0V9a1 1 0 0 1 1-1zM18 8a1 1 0 0 1 1 1v6a1 1 0 1 1-2 0V9a1 1 0 0 1 1-1zM3 11a1 1 0 0 1 1 1v0a1 1 0 1 1-2 0v0a1 1 0 0 1 1-1zM21 11a1 1 0 0 1 1 1v0a1 1 0 1 1-2 0v0a1 1 0 0 1 1-1zM9 6a1 1 0 0 1 1 1v10a1 1 0 1 1-2 0V7a1 1 0 0 1 1-1zM15 6a1 1 0 0 1 1 1v10a1 1 0 1 1-2 0V7a1 1 0 0 1 1-1z" />
              </svg>
              <span>Speaking</span>
            </div>
          )}
        </div>
      )}

      <div className="px-1">
        <ComplexAddons
          addons={inlineAddons}
          toolDetails={toolDetails}
          timestamp={timestamp}
          showFeedback={false}
          branchData={messageId && siblingCount > 1 ? {
            currentIndex,
            totalBranches: siblingCount,
            onNavigate: handleNavigateBranch,
          } : undefined}
        />
      </div>

      {belowAddons.length > 0 && (
        <div className="flex flex-col gap-2 pl-1">
          {belowAddons.map(addon => (
            <div key={addon.id}>{addon.content}</div>
          ))}
        </div>
      )}

    </div>
  );
};

export default ChatBubble;
