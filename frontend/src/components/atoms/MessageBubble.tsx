import React from 'react';
import { MESSAGE_TYPES, MESSAGE_STATES } from '../../mockData';
import { cls } from '../../utils/cls';
import type { BaseComponentProps, MessageType, MessageState, MessageAddon } from '../../types/components';

/**
 * MessageBubble atom component for displaying basic chat message bubbles.
 *
 * This is a pure presentational component that handles:
 * - Message styling based on type (user, assistant, system)
 * - Basic states (typing, error)
 * - Timestamp display
 *
 * For messages with addons (icons, tools, audio), use the ChatBubble molecule.
 */

// Component props interface
export interface MessageBubbleProps extends BaseComponentProps {
  /** Type of message (user, assistant, system) */
  type?: MessageType;
  /** Message content - can be HTML string or React nodes */
  content?: React.ReactNode;
  /** Current message state */
  state?: MessageState;
  /** Message timestamp */
  timestamp?: Date;
  /** Whether to show typing indicator */
  showTyping?: boolean;
  /** Inline addons to display with timestamp */
  addons?: MessageAddon[];
  /** Whether to hide the timestamp */
  hideTimestamp?: boolean;
}

/**
 * Safely formats text content with basic markdown-like patterns.
 * Returns React elements instead of using dangerouslySetInnerHTML.
 */
const formatContentSafe = (text: string): React.ReactNode => {
  // Split by newlines first
  const lines = text.split('\n');

  return lines.map((line, lineIndex) => {
    // Process inline formatting for each line
    const parts: React.ReactNode[] = [];
    let remaining = line;
    let partIndex = 0;

    // Process bold, italic, and code patterns
    const patterns = [
      { regex: /\*\*(.*?)\*\*/, wrapper: (content: string, key: string) => <strong key={key}>{content}</strong> },
      { regex: /\*(.*?)\*/, wrapper: (content: string, key: string) => <em key={key}>{content}</em> },
      { regex: /`(.*?)`/, wrapper: (content: string, key: string) => <code key={key} className="bg-surface-bg px-1 rounded">{content}</code> },
    ];

    while (remaining.length > 0) {
      let earliestMatch: { index: number; length: number; content: string; wrapper: (c: string, k: string) => React.ReactNode } | null = null;

      for (const pattern of patterns) {
        const match = remaining.match(pattern.regex);
        if (match && match.index !== undefined) {
          if (!earliestMatch || match.index < earliestMatch.index) {
            earliestMatch = {
              index: match.index,
              length: match[0].length,
              content: match[1],
              wrapper: pattern.wrapper
            };
          }
        }
      }

      if (earliestMatch) {
        // Add text before the match
        if (earliestMatch.index > 0) {
          parts.push(remaining.slice(0, earliestMatch.index));
        }
        // Add the formatted element
        parts.push(earliestMatch.wrapper(earliestMatch.content, `${lineIndex}-${partIndex++}`));
        remaining = remaining.slice(earliestMatch.index + earliestMatch.length);
      } else {
        // No more matches, add remaining text
        parts.push(remaining);
        break;
      }
    }

    // Add line break between lines (except after the last line)
    if (lineIndex < lines.length - 1) {
      parts.push(<br key={`br-${lineIndex}`} />);
    }

    return <React.Fragment key={lineIndex}>{parts}</React.Fragment>;
  });
};

const MessageBubble: React.FC<MessageBubbleProps> = ({
  type = MESSAGE_TYPES.USER,
  content = '',
  state = MESSAGE_STATES.COMPLETED,
  timestamp = new Date(),
  showTyping = false,
  addons = [],
  hideTimestamp = false,
  className = ''
}) => {
  const getBubbleClasses = (): string => {
    const baseClasses = [
      'message-bubble',
      'relative',
      'max-w-xs',
      'sm:max-w-sm',
      'md:max-w-md',
      'lg:max-w-lg',
      'px-4',
      'py-3',
      'rounded-2xl',
      'text-sm',
      'break-words',
      'transition-all',
      'duration-300',
      'ease-in-out',
    ];

    switch (type) {
      case MESSAGE_TYPES.USER:
        return cls([
          ...baseClasses,
          'user',
          'bg-accent-subtle',
          'text-default',
          'ml-auto',
          'rounded-br-md',
          'shadow-lg',
        ]);

      case MESSAGE_TYPES.ASSISTANT:
        return cls([
          ...baseClasses,
          'assistant',
          'bg-surface',
          'text-default',
          'mr-auto',
          'rounded-bl-md',
          'shadow-lg',
        ]);

      case MESSAGE_TYPES.SYSTEM:
        return cls([
          ...baseClasses,
          'system',
          'bg-surface',
          'text-muted',
          'border',
          'border-accent',
          'mx-auto',
          'text-center',
          'text-xs',
        ]);

      default:
        return cls(baseClasses);
    }
  };

  const renderContent = () => {
    if (typeof content === 'string') {
      return <div>{formatContentSafe(content)}</div>;
    }
    return content;
  };

  const renderAddonIcon = (addon: MessageAddon) => (
    <div
      key={addon.id}
      className={cls(
        'relative',
        'group',
        'cursor-pointer',
        'text-sm'
      )}
      title={addon.tooltip}
    >
      <span>{addon.emoji}</span>
      {/* Tooltip */}
      <div className={cls(
        'absolute',
        'bottom-full',
        'left-1/2',
        'transform',
        '-translate-x-1/2',
        'mb-2',
        'px-3',
        'py-2',
        'text-xs',
        'text-on-emphasis',
        'bg-overlay',
        'backdrop-blur-sm',
        'rounded-md',
        'opacity-0',
        'group-hover:opacity-100',
        'transition-opacity',
        'duration-200',
        'pointer-events-none',
        'whitespace-nowrap',
        'z-10',
        'shadow-lg',
        'border',
        'border-border-emphasis',
        'min-w-[8rem]'
      )}>
        {addon.tooltip}
        {/* Tooltip arrow */}
        <div className={cls(
          'absolute',
          'top-full',
          'left-1/2',
          'transform',
          '-translate-x-1/2',
          'border-4',
          'border-transparent',
          'border-t-overlay'
        )} />
      </div>
    </div>
  );

  const inlineAddons = addons.filter(addon => addon.position === 'inline' || !addon.position);

  return (
    <div className={cls('flex', 'flex-col', 'gap-2', className)}>
      <div className={getBubbleClasses()}>
        {/* Main content */}
        {renderContent()}

        {/* Typing indicator */}
        {(showTyping || state === MESSAGE_STATES.TYPING) && (
          <div className={cls('flex', 'items-center', 'gap-1', 'mt-2')}>
            <div className={cls('flex', 'space-x-1')}>
              <div
                className={cls('w-2', 'h-2', 'bg-accent', 'rounded-full', 'animate-bounce')}
                style={{ animationDelay: '0ms' }}
              />
              <div
                className={cls('w-2', 'h-2', 'bg-accent', 'rounded-full', 'animate-bounce')}
                style={{ animationDelay: '150ms' }}
              />
              <div
                className={cls('w-2', 'h-2', 'bg-accent', 'rounded-full', 'animate-bounce')}
                style={{ animationDelay: '300ms' }}
              />
            </div>
          </div>
        )}

        {/* Error state indicator */}
        {state === MESSAGE_STATES.ERROR && (
          <div className={cls('flex', 'items-center', 'gap-2', 'mt-2', 'text-error')}>
            <svg className={cls('w-4', 'h-4')} fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            <span className={cls('text-xs')}>Failed to send</span>
          </div>
        )}
      </div>

      {/* External footer with addons and timestamp */}
      {(inlineAddons.length > 0 || (!showTyping && !hideTimestamp)) && (
        <div className={cls(
          'flex',
          'items-center',
          'justify-between',
          type === MESSAGE_TYPES.USER ? 'ml-auto' : 'mr-auto',
          'max-w-xs sm:max-w-sm md:max-w-md lg:max-w-lg'
        )}>
          {/* Left side: Addons */}
          <div className={cls('flex', 'items-center', 'gap-2')}>
            {inlineAddons.map(renderAddonIcon)}
          </div>

          {/* Right side: Timestamp */}
          {!hideTimestamp && (
            <div className={cls('text-xs', 'text-muted')}>
              {timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default MessageBubble;
