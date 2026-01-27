import React from 'react';
import { cls } from '../../utils/cls';

const MESSAGE_TYPES = { USER: 'user', ASSISTANT: 'assistant' } as const;
const MESSAGE_STATES = {
  STREAMING: 'streaming',
  COMPLETED: 'completed',
  TYPING: 'typing',
  ERROR: 'error',
} as const;
import type { BaseComponentProps, MessageRole, MessageState, MessageAddon } from '../../types/components';

export interface MessageBubbleProps extends BaseComponentProps {
  type?: MessageRole;
  content?: React.ReactNode;
  state?: MessageState;
  timestamp?: Date;
  showTyping?: boolean;
  addons?: MessageAddon[];
  hideTimestamp?: boolean;
}

const formatContentSafe = (text: string): React.ReactNode => {
  // Split by newlines first
  const lines = text.split('\n');

  return lines.map((line, lineIndex) => {
    // Process inline formatting for each line
    const parts: React.ReactNode[] = [];
    let remaining = line;
    let partIndex = 0;

    // Process bold, italic, code, and link patterns
    // Link pattern matches [text](url) - must come before other patterns to avoid conflicts
    const patterns: Array<{
      regex: RegExp;
      wrapper: (content: string, key: string, extra?: string) => React.ReactNode;
      hasExtra?: boolean;
    }> = [
      {
        regex: /\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/,
        wrapper: (content: string, key: string, url?: string) => (
          <a
            key={key}
            href={url}
            onClick={(e) => {
              e.preventDefault();
              if (url) window.open(url, '_blank', 'noopener,noreferrer');
            }}
            className="text-accent hover:text-accent-hover underline cursor-pointer"
          >
            {content}
          </a>
        ),
        hasExtra: true,
      },
      { regex: /\*\*(.*?)\*\*/, wrapper: (content: string, key: string) => <strong key={key}>{content}</strong> },
      { regex: /\*(.*?)\*/, wrapper: (content: string, key: string) => <em key={key}>{content}</em> },
      { regex: /`(.*?)`/, wrapper: (content: string, key: string) => <code key={key} className="bg-surface-bg px-1 rounded">{content}</code> },
    ];

    while (remaining.length > 0) {
      let earliestMatch: {
        index: number;
        length: number;
        content: string;
        wrapper: (c: string, k: string, extra?: string) => React.ReactNode;
        extra?: string;
      } | null = null;

      for (const pattern of patterns) {
        const match = remaining.match(pattern.regex);
        if (match && match.index !== undefined) {
          if (!earliestMatch || match.index < earliestMatch.index) {
            earliestMatch = {
              index: match.index,
              length: match[0].length,
              content: match[1],
              wrapper: pattern.wrapper,
              extra: pattern.hasExtra ? match[2] : undefined,
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
        parts.push(earliestMatch.wrapper(earliestMatch.content, `${lineIndex}-${partIndex++}`, earliestMatch.extra));
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
      'message-max-width',
      'w-fit',
      'px-4',
      'py-3',
      'rounded-2xl',
      'text-sm',
      'break-words',
    ];

    switch (type) {
      case MESSAGE_TYPES.USER:
        return cls([
          ...baseClasses,
          'user',
          'bg-primary',
          'text-primary-foreground',
          'rounded-br-sm',
        ]);

      case MESSAGE_TYPES.ASSISTANT:
        return cls([
          ...baseClasses,
          'assistant',
          'bg-card',
          'text-card-foreground',
          'rounded-bl-sm',
          'border',
          'border-border',
        ]);

      default:
        return cls(baseClasses);
    }
  };

  const renderContent = () => {
    const isStreaming = state === MESSAGE_STATES.STREAMING;
    const rendered = typeof content === 'string'
      ? <div>{formatContentSafe(content)}</div>
      : content;

    if (isStreaming) {
      return (
        <div className="inline">
          {rendered}
          <span className="inline-block w-0.5 h-4 bg-current ml-0.5 align-middle animate-pulse" />
        </div>
      );
    }

    return rendered;
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
    <div className={cls('flex flex-col gap-1', className)}>
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
            <span className={cls('text-xs')}>{type === MESSAGE_TYPES.ASSISTANT ? 'Generation failed' : 'Failed to send'}</span>
          </div>
        )}
      </div>

      {/* External footer with addons and timestamp */}
      {(inlineAddons.length > 0 || (!showTyping && !hideTimestamp)) && (
        <div className={cls(
          'flex items-center justify-between',
          'px-2 py-1 mt-0.5',
          'min-h-[24px]'
        )}>
          {/* Addons */}
          <div className="flex items-center gap-2">
            {inlineAddons.map(renderAddonIcon)}
          </div>

          {/* Timestamp */}
          {!hideTimestamp && (
            <div className={cls(
              'flex items-center gap-1.5',
              'text-[11px] text-muted-foreground',
              'font-medium tracking-wide'
            )}>
              <svg
                className="w-3 h-3 opacity-60"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span>{timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })}</span>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default MessageBubble;
