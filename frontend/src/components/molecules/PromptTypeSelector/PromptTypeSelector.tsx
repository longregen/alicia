import React from 'react';
import { cls } from '../../../utils/cls';

/**
 * Supported prompt types for optimization.
 */
export type PromptType =
  | 'conversation'
  | 'tool_selection'
  | 'memory_extraction'
  | 'tool_description'
  | 'tool_result_format';

/**
 * Configuration for each prompt type with display properties.
 */
const PROMPT_TYPE_CONFIG: Array<{
  type: PromptType;
  label: string;
  icon: string;
  description: string;
}> = [
  {
    type: 'conversation',
    label: 'Conversation',
    icon: '💬',
    description: 'General conversation and response generation',
  },
  {
    type: 'tool_selection',
    label: 'Tool Selection',
    icon: '🔧',
    description: 'Selecting which tools to use for a task',
  },
  {
    type: 'memory_extraction',
    label: 'Memory Extraction',
    icon: '🧠',
    description: 'Extracting memories from conversations',
  },
  {
    type: 'tool_description',
    label: 'Tool Description',
    icon: '📝',
    description: 'Describing tool capabilities and usage',
  },
  {
    type: 'tool_result_format',
    label: 'Result Format',
    icon: '📊',
    description: 'Formatting tool execution results',
  },
];

export interface PromptTypeSelectorProps {
  /** Currently selected prompt type */
  value: PromptType;
  /** Callback when prompt type changes */
  onChange: (type: PromptType) => void;
  /** Whether the selector is disabled */
  disabled?: boolean;
}

/**
 * Visual button group selector for prompt types.
 * Allows selection of different prompt optimization targets.
 */
export const PromptTypeSelector: React.FC<PromptTypeSelectorProps> = ({
  value,
  onChange,
  disabled = false,
}) => {
  return (
    <div
      className={cls(
        'bg-surface border border-default rounded-lg p-4',
        disabled && 'opacity-60 pointer-events-none'
      )}
    >
      <div className="layout-center-gap mb-3">
        <span className="text-base">🎯</span>
        <span className="font-semibold text-sm text-foreground">Prompt Type</span>
      </div>

      <div className="flex flex-wrap gap-2">
        {PROMPT_TYPE_CONFIG.map((config) => (
          <button
            key={config.type}
            type="button"
            className={cls(
              'flex items-center gap-1 px-3 py-2 rounded-md cursor-pointer text-sm transition-all',
              value === config.type
                ? 'bg-accent/20 text-accent font-medium'
                : 'bg-secondary text-muted-foreground hover:bg-accent/10 hover:text-accent disabled:opacity-50 disabled:cursor-not-allowed'
            )}
            onClick={() => onChange(config.type)}
            disabled={disabled}
            title={config.description}
            aria-pressed={value === config.type}
          >
            <span className="text-sm">{config.icon}</span>
            <span className="text-xs">{config.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
};

export default PromptTypeSelector;
