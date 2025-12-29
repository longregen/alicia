import React, { useState, useEffect } from 'react';
import MessageBubble from '../atoms/MessageBubble';
import ComplexAddons, { type ToolDetail } from '../atoms/ComplexAddons';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type {
  BaseComponentProps,
  MessageType,
  MessageState,
  MessageAddon,
  ToolData,
} from '../../types/components';

/**
 * Collapsible reasoning block component.
 * Renders reasoning steps as blue-bordered blocks that start collapsed.
 */
interface ReasoningBlockProps {
  content: string;
  keyId: number;
}

const ReasoningBlock: React.FC<ReasoningBlockProps> = ({ content, keyId }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  // Truncate content for preview (first 100 chars)
  const previewContent = content.length > 100 ? content.slice(0, 100) + '...' : content;
  const hasMore = content.length > 100;

  return (
    <div
      key={`reasoning-${keyId}`}
      className="my-2 p-3 bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500 rounded-r-lg"
    >
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className={cls(
          'w-full flex items-center justify-between',
          'text-xs text-blue-600 dark:text-blue-400 font-medium mb-1',
          'hover:text-blue-700 dark:hover:text-blue-300 transition-colors'
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
      <div className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
        {isExpanded || !hasMore ? content : previewContent}
      </div>
      {hasMore && !isExpanded && (
        <button
          onClick={() => setIsExpanded(true)}
          className="text-xs text-blue-500 hover:text-blue-600 mt-1"
        >
          Show more
        </button>
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
  className = ''
}) => {
  const [displayedContent, setDisplayedContent] = useState<string>('');
  const [typingIndex, setTypingIndex] = useState<number>(0);

  // Handle streaming/typing animation
  useEffect(() => {
    if (state === MESSAGE_STATES.STREAMING) {
      // Use streamingText for streaming mode, fallback to content
      const textToAnimate = streamingText || content;

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
      setDisplayedContent(content);
      setTypingIndex(0); // Reset typing index when not streaming
    }
  }, [content, streamingText, state, typingIndex]);

  /**
   * Process content to extract and render reasoning blocks as React elements.
   * Returns safe React nodes instead of using dangerouslySetInnerHTML.
   * Reasoning blocks are sorted by sequence number when multiple exist.
   */
  const processContent = (text: string): React.ReactNode => {
    // Match reasoning tags with optional data-sequence attribute
    const reasoningPattern = /<reasoning(?:\s+data-sequence="(\d+)")?>([\s\S]*?)<\/reasoning>/g;

    interface ReasoningBlockData {
      sequence: number;
      content: string;
      startIndex: number;
      endIndex: number;
    }

    const reasoningBlocks: ReasoningBlockData[] = [];
    let match: RegExpExecArray | null;

    // First pass: collect all reasoning blocks with their positions and sequences
    while ((match = reasoningPattern.exec(text)) !== null) {
      const sequence = match[1] ? parseInt(match[1], 10) : 0;
      reasoningBlocks.push({
        sequence,
        content: match[2],
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
        <ReasoningBlock key={`reasoning-${keyIndex}`} content={block.content} keyId={keyIndex++} />
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
        <div className={cls(CSS.flex, CSS.itemsCenter)}>
          <span>{processContent(displayedContent)}</span>
          <span className={cls('inline-block', 'w-0.5', 'h-4', 'bg-current', 'ml-1', CSS.animatePulse)} />
        </div>
      );
    }

    return processContent(content);
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

  return (
    <div className={cls('message-bubble', roleClass, CSS.flex, CSS.flexCol, CSS.gap2, className)}>
      {/* Streaming status badge */}
      {state === MESSAGE_STATES.STREAMING && type === MESSAGE_TYPES.ASSISTANT && (
        <div className={cls(
          'w-full max-w-xs sm:max-w-sm md:max-w-lg lg:max-w-xl',
          'mr-auto',
          'mb-1'
        )}>
          <span className={cls(
            'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
            'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
          )}>
            <span className={cls('w-1.5 h-1.5 rounded-full bg-blue-500 mr-1.5', CSS.animatePulse)} />
            Streaming
          </span>
        </div>
      )}

      {/* Main message bubble */}
      <MessageBubble
        type={type}
        content={getContentToDisplay()}
        state={state}
        timestamp={timestamp}
        showTyping={showTyping}
        addons={[]}
        hideTimestamp={true}
      />

      {/* Addons footer with timestamp (always shown) */}
      <div className={cls(
        'w-full max-w-xs sm:max-w-sm md:max-w-md lg:max-w-lg',
        type === MESSAGE_TYPES.USER ? 'ml-auto' : 'mr-auto'
      )}>
        <ComplexAddons
          addons={inlineAddons}
          toolDetails={toolDetails}
          timestamp={timestamp}
          className="w-full"
        />
      </div>


      {/* Below addons */}
      {belowAddons.length > 0 && (
        <div className={cls(
          CSS.flex,
          CSS.flexCol,
          CSS.gap2,
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
