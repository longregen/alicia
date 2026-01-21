import React, { useState, useEffect, useCallback } from 'react';
import MessageBubble from '../atoms/MessageBubble';
import ComplexAddons, { type ToolDetail } from '../atoms/ComplexAddons';
import FeedbackControls from '../atoms/FeedbackControls';
import { Textarea } from '../atoms/Textarea';
import Button from '../atoms/Button';
import Badge from '../atoms/Badge';
import ConflictResolutionDialog from './ConflictResolutionDialog';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { useFeedback } from '../../hooks/useFeedback';
import { useBranchStore } from '../../stores/branchStore';
import type {
  BaseComponentProps,
  MessageRole,
  MessageState,
  MessageAddon,
  ToolData,
} from '../../types/components';
import type { MessageId } from '../../types/streaming';
import type { SyncStatus } from '../../types/models';
import type { ConflictDetails } from '../../types/sync';

/**
 * Collapsible reasoning block component.
 * Renders reasoning steps as blue-bordered blocks that start expanded.
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
  /** Array of tools attached to the message */
  tools?: ToolData[];
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
  /** Sync status for offline sync support */
  syncStatus?: SyncStatus;
  /** Server version content for conflict resolution */
  serverContent?: string;
  /** Conflict details if sync status is 'conflict' */
  conflictDetails?: ConflictDetails;
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
  conversationId,
  onEditMessage,
  onBranchSwitch,
  onContinueFromHere,
  onRetry,
  syncStatus,
  serverContent,
  conflictDetails,
  className = ''
}) => {
  const [displayedContent, setDisplayedContent] = useState<string>('');
  const [typingIndex, setTypingIndex] = useState<number>(0);
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [editedContent, setEditedContent] = useState<string>('');
  const [conflictDialogOpen, setConflictDialogOpen] = useState<boolean>(false);

  // Branch store for managing message siblings.
  // Siblings are now initialized from the main message fetch in useMessages,
  // so we don't need to fetch per-message anymore.
  const {
    switchBranch,
    getSiblingCount,
    getCurrentIndex,
    isLoading: isBranchLoading,
  } = useBranchStore();

  // Get current branch state
  const siblingCount = messageId ? getSiblingCount(messageId) : 0;
  const currentIndex = messageId ? getCurrentIndex(messageId) : 0;
  const branchLoading = messageId ? isBranchLoading(messageId) : false;

  // Use prop content (backend is source of truth for displayed message)
  const effectiveContent = content;

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
   * Process content to extract and render reasoning/thinking blocks as React elements.
   * Returns safe React nodes.
   * Reasoning blocks are sorted by sequence number when multiple exist.
   * Thinking summary blocks are rendered at the top as a subtle header.
   */
  const processContent = (text: string): React.ReactNode => {
    // Match thinking-summary tags
    const thinkingSummaryPattern = /<thinking-summary([^>]*)>([\s\S]*?)<\/thinking-summary>/g;
    // Match reasoning tags with optional data-sequence and data-id attributes
    const reasoningPattern = /<reasoning([^>]*)>([\s\S]*?)<\/reasoning>/g;

    interface ThinkingSummaryData {
      content: string;
      id?: string;
      startIndex: number;
      endIndex: number;
    }

    interface ReasoningBlockData {
      sequence: number;
      content: string;
      id?: string;
      startIndex: number;
      endIndex: number;
    }

    const thinkingSummaries: ThinkingSummaryData[] = [];
    const reasoningBlocks: ReasoningBlockData[] = [];
    let match: RegExpExecArray | null;

    // First pass: collect thinking summaries
    while ((match = thinkingSummaryPattern.exec(text)) !== null) {
      const attrsStr = match[1];
      const idMatch = /data-id="([^"]+)"/.exec(attrsStr);
      const id = idMatch ? idMatch[1] : undefined;
      thinkingSummaries.push({
        content: match[2],
        id,
        startIndex: match.index,
        endIndex: thinkingSummaryPattern.lastIndex,
      });
    }

    // Second pass: collect all reasoning blocks with their positions, sequences, and ids
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

    // If no special blocks, return text as-is
    if (reasoningBlocks.length === 0 && thinkingSummaries.length === 0) {
      return text;
    }

    // Sort reasoning blocks by sequence number
    reasoningBlocks.sort((a, b) => a.sequence - b.sequence);

    // Build the output with text segments and sorted blocks
    const parts: React.ReactNode[] = [];
    let keyIndex = 0;

    // Render thinking summaries at the top as a subtle header
    for (const summary of thinkingSummaries) {
      parts.push(
        <div key={`thinking-summary-${keyIndex++}`} className="text-xs text-muted italic mb-2 pb-2 border-b border-muted/30">
          <span className="text-accent font-medium">Thinking:</span> {summary.content}
        </div>
      );
    }

    // Find all block positions for text extraction
    const allBlocks = [
      ...thinkingSummaries.map(b => ({ startIndex: b.startIndex, endIndex: b.endIndex })),
      ...reasoningBlocks.map(b => ({ startIndex: b.startIndex, endIndex: b.endIndex })),
    ].sort((a, b) => a.startIndex - b.startIndex);

    // Extract text segments between blocks
    let currentPos = 0;
    for (const block of allBlocks) {
      if (block.startIndex > currentPos) {
        const textSegment = text.slice(currentPos, block.startIndex).trim();
        if (textSegment) {
          parts.push(<span key={`text-${keyIndex++}`}>{textSegment}</span>);
        }
      }
      currentPos = block.endIndex;
    }

    // Render all reasoning blocks in sequence order
    for (const block of reasoningBlocks) {
      parts.push(
        <ReasoningBlock key={`reasoning-${keyIndex}`} content={block.content} keyId={keyIndex++} id={block.id} />
      );
    }

    // Add remaining text after last block
    if (currentPos < text.length) {
      const remainingText = text.slice(currentPos).trim();
      if (remainingText) {
        parts.push(<span key={`text-${keyIndex++}`}>{remainingText}</span>);
      }
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
    if (messageId && editedContent !== effectiveContent) {
      // Notify parent to trigger new agent response with edited content
      // The backend will create a new sibling message
      if (onEditMessage) {
        onEditMessage(messageId, editedContent);
      }
    }
    setIsEditing(false);
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditedContent('');
  };

  // Navigate to prev/next sibling via backend API
  const handleNavigateBranch = useCallback(async (direction: 'prev' | 'next') => {
    if (!messageId || !conversationId) return;

    const targetSibling = await switchBranch(conversationId, messageId, direction);

    // If branch switched successfully, notify parent to reload messages
    if (targetSibling && onBranchSwitch) {
      onBranchSwitch(targetSibling.id);
    }
  }, [messageId, conversationId, switchBranch, onBranchSwitch]);

  const handleConflictClick = () => {
    if (syncStatus === 'conflict') {
      setConflictDialogOpen(true);
    }
  };

  const handleKeepLocal = () => {
    console.log('Keeping local version for message:', messageId);
    // Note: Correction/conflict resolution requires backend API implementation
    // For now, these are UI-only operations
  };

  const handleKeepServer = () => {
    console.log('Keeping server version for message:', messageId);
    // Note: Correction/conflict resolution requires backend API implementation
    // For now, these are UI-only operations
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
  const isUser = type === MESSAGE_TYPES.USER;

  // Don't allow editing while streaming or during branch switch
  const canEdit = state !== MESSAGE_STATES.STREAMING && !branchLoading;

  return (
    <div className={cls('flex flex-col gap-1', roleClass, className)}>
      {/* Status badges */}
      {state === MESSAGE_STATES.STREAMING && type === MESSAGE_TYPES.ASSISTANT && (
        <span className="badge badge-default w-fit text-xs">
          <span className="w-1.5 h-1.5 rounded-full bg-accent mr-1.5 animate-pulse" />
          Streaming
        </span>
      )}

      {branchLoading && (
        <span className="badge badge-default w-fit text-xs">
          <span className="w-1.5 h-1.5 rounded-full bg-accent mr-1.5 animate-pulse" />
          Switching branch...
        </span>
      )}

      {syncStatus === 'conflict' && (
        <Badge
          variant="destructive"
          className="w-fit cursor-pointer hover:opacity-80"
          onClick={handleConflictClick}
        >
          <svg className="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          Sync Conflict
        </Badge>
      )}

      {syncStatus === 'pending' && (
        <Badge variant="secondary" className="w-fit text-xs">
          <svg className="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          Pending
        </Badge>
      )}

      {/* Main message bubble or edit mode */}
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
          {/* Hover actions - CSS-only visibility */}
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
        </div>
      )}

      {/* Footer: addons, branch navigation, timestamp */}
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

      {/* Below addons (audio, etc.) */}
      {belowAddons.length > 0 && (
        <div className="flex flex-col gap-2 pl-1">
          {belowAddons.map(addon => (
            <div key={addon.id}>{addon.content}</div>
          ))}
        </div>
      )}

      {/* Conflict resolution dialog */}
      {syncStatus === 'conflict' && serverContent && (
        <ConflictResolutionDialog
          open={conflictDialogOpen}
          onOpenChange={setConflictDialogOpen}
          localContent={effectiveContent}
          serverContent={serverContent}
          conflict={conflictDetails}
          onKeepLocal={handleKeepLocal}
          onKeepServer={handleKeepServer}
        />
      )}
    </div>
  );
};

export default ChatBubble;
