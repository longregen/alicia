import React from 'react';
import * as SwitchPrimitive from '@radix-ui/react-switch';
import { cn } from '../../lib/utils';
import type { BaseComponentProps, Variant } from '../../types/components';

// Base Switch component following shadcn/ui pattern
const Switch = React.forwardRef<
  React.ComponentRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitive.Root
    data-slot="switch"
    className={cn(
      'peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input',
      className
    )}
    {...props}
    ref={ref}
  >
    <SwitchPrimitive.Thumb
      data-slot="switch-thumb"
      className={cn(
        'pointer-events-none block size-4 rounded-full bg-background shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0'
      )}
    />
  </SwitchPrimitive.Root>
));

Switch.displayName = 'Switch';

// Component props interface for backwards compatibility wrapper
export interface ToggleSwitchProps extends BaseComponentProps {
  /** Whether the toggle is checked */
  checked?: boolean;
  /** Callback fired when the toggle state changes */
  onChange?: (checked: boolean) => void;
  /** Whether the toggle is disabled */
  disabled?: boolean;
  /** Size of the toggle switch (maintained for API compatibility, maps to className) */
  size?: 'sm' | 'md' | 'lg';
  /** Visual variant when checked (maintained for API compatibility, limited support) */
  variant?: Variant;
  /** Optional label text */
  label?: string;
}

// Backwards compatibility wrapper
const ToggleSwitch: React.FC<ToggleSwitchProps> = ({
  checked,
  onChange,
  disabled = false,
  size = 'md',
  variant: _variant = 'default',
  label,
  className = ''
}) => {
  const handleCheckedChange = (newChecked: boolean): void => {
    onChange?.(newChecked);
  };

  const getSizeClasses = (): string => {
    switch (size) {
      case 'sm':
        return 'h-4 w-8 [&_[data-slot=switch-thumb]]:size-3 [&_[data-slot=switch-thumb]]:data-[state=checked]:translate-x-4';
      case 'lg':
        return 'h-6 w-11 [&_[data-slot=switch-thumb]]:size-5 [&_[data-slot=switch-thumb]]:data-[state=checked]:translate-x-5';
      default:
        return '';
    }
  };

  const getLabelClasses = (): string => {
    return cn(
      'text-sm font-medium select-none cursor-pointer',
      disabled ? 'text-muted cursor-not-allowed' : 'text-default'
    );
  };

  return (
    <div className={cn('flex items-center gap-3', className)}>
      {label && (
        <label
          className={getLabelClasses()}
          onClick={() => !disabled && onChange?.(!checked)}
        >
          {label}
        </label>
      )}

      <Switch
        checked={checked}
        onCheckedChange={handleCheckedChange}
        disabled={disabled}
        className={getSizeClasses()}
      />
    </div>
  );
};

export default ToggleSwitch;
export { Switch, ToggleSwitch };
