import React, { useState, useEffect } from 'react';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type { BaseComponentProps, Variant } from '../../types/components';

// Toggle size mapping
const TOGGLE_SIZE_MAP = {
  sm: {
    track: 'w-8 h-4',
    thumb: 'w-3 h-3',
    translate: 'translate-x-4',
  },
  md: {
    track: 'w-11 h-6',
    thumb: 'w-5 h-5',
    translate: 'translate-x-5',
  },
  lg: {
    track: 'w-14 h-7',
    thumb: 'w-6 h-6',
    translate: 'translate-x-7',
  },
} as const;

// Component props interface
export interface ToggleSwitchProps extends BaseComponentProps {
  /** Whether the toggle is checked */
  checked?: boolean;
  /** Callback fired when the toggle state changes */
  onChange?: (checked: boolean) => void;
  /** Whether the toggle is disabled */
  disabled?: boolean;
  /** Size of the toggle switch */
  size?: keyof typeof TOGGLE_SIZE_MAP;
  /** Visual variant when checked */
  variant?: Variant;
  /** Optional label text */
  label?: string;
}

const ToggleSwitch: React.FC<ToggleSwitchProps> = ({
  checked,
  onChange,
  disabled = false,
  size = 'md',
  variant = 'default',
  label,
  className = ''
}) => {
  const [internalChecked, setInternalChecked] = useState<boolean>(checked ?? false);

  // Determine if component is controlled
  const isControlled = checked !== undefined;
  const isChecked = isControlled ? checked : internalChecked;

  // Sync internal state when checked prop changes in controlled mode
  useEffect(() => {
    if (isControlled && checked !== undefined) {
      setInternalChecked(checked);
    }
  }, [checked, isControlled]);

  const handleToggle = (newValue: boolean): void => {
    if (!isControlled) {
      setInternalChecked(newValue);
    }
    onChange?.(newValue);
  };

  const getTrackClasses = (): string => {
    const baseClasses = [
      TOGGLE_SIZE_MAP[size].track,
      'relative',
      CSS.roundedFull,
      CSS.transitionAll,
      CSS.duration200,
      'ease-in-out',
      CSS.cursorPointer,
    ];

    if (disabled) {
      return cls(baseClasses, [
        CSS.bgInactiveDisabled,
        CSS.cursorNotAllowed,
        'opacity-60',
      ]);
    }

    if (isChecked) {
      switch (variant) {
        case 'success':
          return cls(baseClasses, [
            CSS.bgActiveSpeaking,
            'hover:bg-active-speaking',
          ]);
        case 'warning':
          return cls(baseClasses, [
            CSS.bgToolResult,
            'hover:bg-tool-result',
          ]);
        case 'error':
          return cls(baseClasses, [
            CSS.bgError,
            'hover:bg-error',
          ]);
        default:
          return cls(baseClasses, [
            CSS.bgPrimaryBlue,
            CSS.hoverBgPrimaryBlue,
          ]);
      }
    } else {
      return cls(baseClasses, [
        CSS.bgInactiveDisabled,
        'hover:bg-primary-blue-glow',
      ]);
    }
  };

  const getThumbClasses = (): string => {
    const baseClasses = [
      TOGGLE_SIZE_MAP[size].thumb,
      CSS.bgWhite,
      CSS.roundedFull,
      'shadow-lg',
      'transform',
      CSS.transitionAll,
      CSS.duration200,
      'ease-in-out',
      'absolute',
      'top-0.5',
      'left-0.5',
    ];

    if (isChecked) {
      return cls(baseClasses, TOGGLE_SIZE_MAP[size].translate);
    }

    return cls(baseClasses);
  };

  const handleClick = (): void => {
    if (!disabled) {
      handleToggle(!isChecked);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent): void => {
    if ((e.key === 'Enter' || e.key === ' ') && !disabled) {
      e.preventDefault();
      handleToggle(!isChecked);
    }
  };

  const getLabelClasses = (): string => {
    const baseClasses = [
      CSS.textSm,
      CSS.fontMedium,
      CSS.selectNone,
      CSS.cursorPointer,
    ];

    if (disabled) {
      return cls(baseClasses, [
        CSS.textMuted,
        CSS.cursorNotAllowed,
      ]);
    }

    return cls(baseClasses, [
      CSS.textPrimary,
    ]);
  };

  const ariaLabel = label || `Toggle switch ${isChecked ? 'on' : 'off'}`;

  return (
    <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap3, className)}>
      {label && (
        <label
          className={getLabelClasses()}
          onClick={handleClick}
        >
          {label}
        </label>
      )}

      <div
        className={getTrackClasses()}
        onClick={handleClick}
        role="button"
        tabIndex={disabled ? -1 : 0}
        onKeyDown={handleKeyDown}
        aria-pressed={isChecked}
        aria-label={ariaLabel}
        aria-disabled={disabled}
      >
        <input
          type="checkbox"
          checked={isChecked}
          onChange={() => {}} // Controlled by parent div
          disabled={disabled}
          className="sr-only"
        />
        <div className={getThumbClasses()} />
      </div>
    </div>
  );
};

export default ToggleSwitch;
