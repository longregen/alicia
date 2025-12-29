import React, { useState, useRef, useEffect } from 'react';
import { cls } from '../../utils/cls';
import { css } from '../../utils/constants';
import type { BaseComponentProps } from '../../types/components';

// Component props interface
export interface ResizableBarTextInputProps extends BaseComponentProps {
  /** Current value of the input (for controlled usage) */
  value?: string;
  /** Callback fired when the input value changes */
  onChange?: (value: string) => void;
  /** Callback fired when Enter is pressed without Shift */
  onSubmit?: (value: string) => void;
  /** Placeholder text when input is empty */
  placeholder?: string;
  /** Whether the input is disabled */
  disabled?: boolean;
  /** Whether to show an indicator that it's a multiline input */
  showMultiline?: boolean;
  /** Maximum number of rows before scrolling */
  maxRows?: number;
  /** Minimum number of rows */
  minRows?: number;
  /** Whether to auto-focus the input on mount */
  autoFocus?: boolean;
}

const ResizableBarTextInput: React.FC<ResizableBarTextInputProps> = ({
  value = '',
  onChange,
  onSubmit,
  placeholder = 'Type a message...',
  disabled = false,
  minRows = 1,
  maxRows = 4,
  showMultiline = false,
  autoFocus = false,
  className = ''
}) => {
  const [internalValue, setInternalValue] = useState<string>(value);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [isFocused, setIsFocused] = useState<boolean>(false);

  // Handle controlled vs uncontrolled
  const currentValue = onChange ? value : internalValue;
  const handleValueChange = onChange || setInternalValue;

  // Auto focus
  useEffect(() => {
    if (autoFocus && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [autoFocus]);

  // Auto-resize functionality
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const adjustHeight = () => {
      // Reset height to get accurate scrollHeight
      textarea.style.height = 'auto';

      // Calculate line height
      const style = window.getComputedStyle(textarea);
      const lineHeight = parseInt(style.lineHeight) || parseInt(style.fontSize) * 1.2;

      // Calculate number of lines
      const lines = Math.floor(textarea.scrollHeight / lineHeight);

      // Apply min/max constraints
      const finalLines = Math.max(minRows, Math.min(lines, maxRows || 10));

      // Set the height
      textarea.style.height = `${finalLines * lineHeight}px`;
    };

    adjustHeight();
  }, [currentValue, minRows, maxRows]);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>): void => {
    handleValueChange(e.target.value);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>): void => {
    // Submit on Enter (without Shift)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (onSubmit && currentValue.trim()) {
        onSubmit(currentValue.trim());
      }
    }
  };

  const handleFocus = (): void => {
    setIsFocused(true);
  };

  const handleBlur = (): void => {
    setIsFocused(false);
  };

  const getContainerClasses = (): string => {
    const baseClasses = [
      'relative',
      css.wFull,
      'pl-3',
      'pr-0',
      'py-2',
      'rounded-xl',
      'border',
    ];

    if (disabled) {
      return cls(baseClasses, [
        'opacity-60',
        css.cursorNotAllowed,
        'border-inactive-disabled',
        css.bgInactiveDisabled,
      ]);
    }

    if (isFocused) {
      return cls(baseClasses, [
        'border-primary-blue',
      ]);
    }

    return cls(baseClasses, [
      'border-primary-blue-glow',
      'hover:border-primary-blue',
    ]);
  };

  const getTextareaClasses = (): string => {
    const baseClasses = [
      css.wFull,
      'pr-4',
      css.textSm,
      css.textPrimary,
      'placeholder-muted-text',
      'rounded-xl',
      'resize-none',
      'overflow-auto',
      'outline-none',
      'focus:outline-none',
      'border-0'
    ];

    if (disabled) {
      return cls(baseClasses, [
        'bg-transparent', // Transparent since container has the background
        css.cursorNotAllowed,
      ]);
    }

    return cls(baseClasses, [
      'bg-surface-900', // Solid background
    ]);
  };

  const lineCount = currentValue.split('\n').length;
  const isMultiline = lineCount > 1 || currentValue.length > 50;

  return (
    <div className={cls(getContainerClasses(), className)}>
      <textarea
        ref={textareaRef}
        value={currentValue}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        onFocus={handleFocus}
        onBlur={handleBlur}
        placeholder={placeholder}
        disabled={disabled}
        className={getTextareaClasses()}
        aria-label="Message input"
      />

      {/* Visual indicator for multiline */}
      {isMultiline && showMultiline && (
        <div className={cls('absolute', 'top-2', 'right-3', css.textXs, css.textSurface400, 'dark:text-surface-500', 'pointer-events-none')}>
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
          </svg>
        </div>
      )}

    </div>
  );
};

export default ResizableBarTextInput;
