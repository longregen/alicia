import React, { useState, useEffect } from 'react';
import MessageBubble from '../atoms/MessageBubble';
import ComplexAddons, { type ToolDetail } from '../atoms/ComplexAddons';
import FeedbackControls from '../atoms/FeedbackControls';
import BranchNavigator from '../atoms/BranchNavigator';
import { Textarea } from '../atoms/Textarea';
import Button from '../atoms/Button';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { cn } from '../../lib/utils';
import { useFeedback } from '../../hooks/useFeedback';
import { useBranchStore } from '../../stores/branchStore';
import type {
  BaseComponentProps,
  MessageType,
  MessageState,
  MessageAddon,
  ToolData,
} from '../../types/components';
import type { MessageId } from '../../types/streaming';

/**
 * Collapsible reasoning block component.
 * Renders reasoning steps as blue-bordered blocks that start collapsed.
 * Supports voting via useFeedback hook when an id is provided.
 */
interface ReasoningBlockProps {
  content: string;
  keyId: number;
  id?: string;
}

const ReasoningBlock: React.FC<ReasoningBlockProps> = ({ content, keyId, id }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Feedback hook for reasoning voting (only works if id is provided)
  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('reasoning', id || '');

  // Truncate content for preview (first 100 chars)
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

/**
 * ChatBubble molecule component that extends MessageBubble with addon support.
 *
 * Supports various addons:
 * - Icons: Render inline with the message (e.g., translation flags)
 * - Tools: Render below with expand/collapse functionality
 * - Audio: Render as compact player that expands when playing
 *
 * Also handles streaming animation and tool expansion state.
 */

// Component props interface
export interface ChatBubbleProps extends BaseComponentProps {
  /** Type of message (user, assistant, system) */
  type?: MessageType;
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
  /** Array of tools attached to the message */
  tools?: ToolData[];
  /** Message ID for branch tracking */
  messageId?: MessageId;
  /** Callback when user clicks "Continue from here" on an assistant message */
  onContinueFromHere?: (messageId: MessageId) => void;
}

const ChatBubble: React.FC<ChatBubbleProps> = ({
  type = MESSAGE_TYPES.USER,
  content = '',
  state = MESSAGE_STATES.COMPLETED,
  timestamp = new Date(),
  showTyping = false,
  streamingText = '',
  addons = [],
  tools = [],
  messageId,
  onContinueFromHere,
  className = ''
}) => {
  const [displayedContent, setDisplayedContent] = useState<string>('');
  const [typingIndex, setTypingIndex] = useState<number>(0);
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [editedContent, setEditedContent] = useState<string>('');
  const [isHovering, setIsHovering] = useState<boolean>(false);

  // Branch store for managing message versions
  const {
    initializeBranch,
    createBranch,
    navigateToBranch,
    getCurrentBranch,
    getBranchCount,
    getCurrentIndex,
  } = useBranchStore();

  // Initialize branch for user messages on mount
  useEffect(() => {
    if (messageId && type === MESSAGE_TYPES.USER && content) {
      initializeBranch(messageId, content);
    }
  }, [messageId, type, content, initializeBranch]);

  // Get current branch content if available
  const currentBranch = messageId ? getCurrentBranch(messageId) : null;
  const branchCount = messageId ? getBranchCount(messageId) : 0;
  const currentIndex = messageId ? getCurrentIndex(messageId) : 0;

  // Use branch content if available, otherwise use prop content
  const effectiveContent = currentBranch ? currentBranch.content : content;

  // Handle streaming/typing animation
  useEffect(() => {
    if (state === MESSAGE_STATES.STREAMING) {
      // Use streamingText for streaming mode, fallback to effectiveContent
      const textToAnimate = streamingText || effectiveContent;

      if (textToAnimate) {
        const timer = setTimeout(() => {
          if (typingIndex < textToAnimate.length) {
            setDisplayedContent(textToAnimate.slice(0, typingIndex + 1));
            setTypingIndex(prev => prev + 1);
          }
        }, 30); // Typing speed

        return () => clearTimeout(timer);
      }
    } else {
      setDisplayedContent(effectiveContent);
      setTypingIndex(0); // Reset typing index when not streaming
    }
  }, [effectiveContent, streamingText, state, typingIndex]);

  /**
   * Process content to extract and render reasoning blocks as React elements.
   * Returns safe React nodes instead of using dangerouslySetInnerHTML.
   * Reasoning blocks are sorted by sequence number when multiple exist.
   */
  const processContent = (text: string): React.ReactNode => {
    // Match reasoning tags with optional data-sequence and data-id attributes
    const reasoningPattern = /<reasoning([^>]*)>([\s\S]*?)<\/reasoning>/g;

    interface ReasoningBlockData {
      sequence: number;
      content: string;
      id?: string;
      startIndex: number;
      endIndex: number;
    }

    const reasoningBlocks: ReasoningBlockData[] = [];
    let match: RegExpExecArray | null;

    // First pass: collect all reasoning blocks with their positions, sequences, and ids
    while ((match = reasoningPattern.exec(text)) !== null) {
      const attrsStr = match[1];
      const sequenceMatch = /data-sequence="(\d+)"/.exec(attrsStr);
      const idMatch = /data-id="([^"]+)"/.exec(attrsStr);
      const sequence = sequenceMatch ? parseInt(sequenceMatch[1], 10) : 0;
      const id = idMatch ? idMatch[1] : undefined;
      reasoningBlocks.push({
        sequence,
        content: match[2],
        id,
        startIndex: match.index,
        endIndex: reasoningPattern.lastIndex,
      });
    }

    // If no reasoning blocks, return text as-is
    if (reasoningBlocks.length === 0) {
      return text;
    }

    // Sort reasoning blocks by sequence number
    reasoningBlocks.sort((a, b) => a.sequence - b.sequence);

    // Build the output with text segments and sorted reasoning blocks
    const parts: React.ReactNode[] = [];
    let keyIndex = 0;

    // Find the first text segment before any reasoning block
    const firstBlockStart = Math.min(...reasoningBlocks.map(b => b.startIndex));
    if (firstBlockStart > 0) {
      parts.push(<span key={`text-${keyIndex++}`}>{text.slice(0, firstBlockStart)}</span>);
    }

    // Render all reasoning blocks in sequence order
    for (const block of reasoningBlocks) {
      parts.push(
        <ReasoningBlock key={`reasoning-${keyIndex}`} content={block.content} keyId={keyIndex++} id={block.id} />
      );
    }

    // Find text after the last reasoning block
    const lastBlockEnd = Math.max(...reasoningBlocks.map(b => b.endIndex));
    if (lastBlockEnd < text.length) {
      parts.push(<span key={`text-${keyIndex++}`}>{text.slice(lastBlockEnd)}</span>);
    }

    return parts;
  };

  const getContentToDisplay = (): React.ReactNode => {
    if (state === MESSAGE_STATES.STREAMING) {
      return (
        <div className="layout-center">
          <span>{processContent(displayedContent)}</span>
          <span className="inline-block w-0.5 h-4 bg-current ml-1 animate-pulse" />
        </div>
      );
    }

    return processContent(effectiveContent);
  };

  const handleEditClick = () => {
    setEditedContent(effectiveContent);
    setIsEditing(true);
  };

  const handleSaveEdit = () => {
    if (type === MESSAGE_TYPES.USER && messageId) {
      // For user messages: create new branch (UI-only)
      createBranch(messageId, editedContent);
    } else if (type === MESSAGE_TYPES.ASSISTANT) {
      // For assistant messages: submit as correction feedback
      console.log('Assistant message edited - would submit feedback:', editedContent);
      // TODO: feedbackApi.submitCorrection(editedContent);
    }
    setIsEditing(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditedContent('');
  };

  const handleNavigateBranch = (direction: 'prev' | 'next') => {
    if (messageId) {
      navigateToBranch(messageId, direction);
    }
  };

  const inlineAddons = addons.filter(addon => addon.position === 'inline' || !addon.position);
  const belowAddons = addons.filter(addon => addon.position === 'below');

  // Convert tools to ToolDetails for ComplexAddons
  const toolDetails: ToolDetail[] = tools.map(tool => ({
    id: tool.id,
    name: tool.name,
    description: tool.description,
    result: tool.result,
    status: tool.status
  }));

  // Determine role-based class for e2e testing
  const roleClass = type === MESSAGE_TYPES.USER ? 'user' : type === MESSAGE_TYPES.ASSISTANT ? 'assistant' : 'system';

  // Don't allow editing while streaming
  const canEdit = state !== MESSAGE_STATES.STREAMING;

  return (
    <div
      className={cls('layout-stack-gap', roleClass, className)}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {/* Streaming status badge */}
      {state === MESSAGE_STATES.STREAMING && type === MESSAGE_TYPES.ASSISTANT && (
        <div className="w-full max-w-xs sm:max-w-sm md:max-w-lg lg:max-w-xl mr-auto mb-1">
          <span className="badge badge-neutral">
            <span className="w-1.5 h-1.5 rounded-full bg-accent mr-1.5 animate-pulse" />
            Streaming
          </span>
        </div>
      )}

      {/* Main message bubble or edit mode */}
      {isEditing ? (
        <div className={cn(
          'w-full message-max-width',
          type === MESSAGE_TYPES.USER ? 'ml-auto' : 'mr-auto'
        )}>
          <Textarea
            value={editedContent}
            onChange={(e) => setEditedContent(e.target.value)}
            className="w-full mb-2 min-h-24"
            autoFocus
          />
          <div className="flex gap-2 justify-end">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCancelEdit}
            >
              Cancel
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={handleSaveEdit}
            >
              Save
            </Button>
          </div>
        </div>
      ) : (
        <div className="relative group">
          <MessageBubble
            type={type}
            content={getContentToDisplay()}
            state={state}
            timestamp={timestamp}
            showTyping={showTyping}
            addons={[]}
            hideTimestamp={true}
          />
          {/* Edit button - show on hover */}
          {canEdit && isHovering && (
            <div className={cn(
              'absolute top-2 flex gap-1',
              type === MESSAGE_TYPES.USER ? 'left-2' : 'right-2'
            )}>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={handleEditClick}
                className="opacity-70 hover:opacity-100"
                aria-label="Edit message"
              >
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
              </Button>
              {/* Continue from here button - only for assistant messages */}
              {type === MESSAGE_TYPES.ASSISTANT && messageId && onContinueFromHere && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onContinueFromHere(messageId)}
                  className="opacity-70 hover:opacity-100"
                  aria-label="Continue from here"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 5l7 7-7 7"
                    />
                  </svg>
                </Button>
              )}
            </div>
          )}
        </div>
      )}

      {/* Addons footer with timestamp and branch navigator */}
      <div className={cls(
        'w-full message-max-width',
        type === MESSAGE_TYPES.USER ? 'ml-auto' : 'mr-auto'
      )}>
        <div className="layout-between-gap">
          <div className="flex-1">
            <ComplexAddons
              addons={inlineAddons}
              toolDetails={toolDetails}
              timestamp={timestamp}
              className="w-full"
              showFeedback={true}
            />
          </div>
          {/* Branch navigator - only for user messages with branches */}
          {type === MESSAGE_TYPES.USER && messageId && branchCount > 0 && (
            <BranchNavigator
              messageId={messageId}
              currentIndex={currentIndex}
              totalBranches={branchCount}
              onNavigate={handleNavigateBranch}
            />
          )}
        </div>
      </div>


      {/* Below addons */}
      {belowAddons.length > 0 && (
        <div className={cls(
          'layout-stack-gap',
          type === MESSAGE_TYPES.USER ? 'ml-4' : 'mr-4'
        )}>
          {belowAddons.map(addon => (
            <div key={addon.id}>
              {addon.content}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default ChatBubble;
