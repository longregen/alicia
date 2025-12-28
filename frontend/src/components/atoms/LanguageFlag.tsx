import React from 'react';
import { cls } from '../../utils/cls';
import { css } from '../../utils/constants';
import type { BaseComponentProps } from '../../types/components';

// Language flag size mapping - only sizes actually used
const FLAG_SIZE_MAP = {
  sm: 'w-4 h-4 text-xs',
  md: 'w-6 h-6 text-sm',
  lg: 'w-8 h-8 text-base',
} as const;

// Language code to flag mapping
const FLAG_MAP: Record<string, string> = {
  'en': '🇺🇸',
  'es': '🇪🇸',
  'fr': '🇫🇷',
  'de': '🇩🇪',
  'it': '🇮🇹',
  'pt': '🇵🇹',
  'ru': '🇷🇺',
  'zh': '🇨🇳',
  'ja': '🇯🇵',
} as const;

// Language code to name mapping
const LANGUAGE_NAMES: Record<string, string> = {
  'auto': 'Auto-detect',
  'en': 'English',
  'es': 'Spanish',
  'fr': 'French',
  'de': 'German',
  'it': 'Italian',
  'pt': 'Portuguese',
  'ru': 'Russian',
  'zh': 'Chinese',
  'ja': 'Japanese',
} as const;

// Simplified component props interface
export interface LanguageFlagProps extends BaseComponentProps {
  /** Language code (e.g., 'en', 'es', 'fr', 'auto') */
  languageCode?: string;
  /** Flag emoji to display (overrides auto-detection) */
  flag?: string;
  /** Size of the flag */
  size?: keyof typeof FLAG_SIZE_MAP;
}

const LanguageFlag: React.FC<LanguageFlagProps> = ({
  languageCode = 'en',
  flag,
  size = 'md',
  className = ''
}) => {
  const getFlagClasses = (): string => {
    return cls(
      FLAG_SIZE_MAP[size],
      css.roundedFull,
      css.flex,
      css.itemsCenter,
      css.justifyCenter,
      css.selectNone
    );
  };

  // Get the flag to display
  const getDisplayFlag = (): string => {
    // Use provided flag if available
    if (flag) return flag;

    // Auto-detect gets a globe icon
    if (languageCode === 'auto') return '🌐';

    // Look up flag from map, fallback to generic flag
    return FLAG_MAP[languageCode] || '🏳️';
  };

  const ariaLabel = LANGUAGE_NAMES[languageCode] || `Language: ${languageCode}`;
  const displayFlag = getDisplayFlag();

  return (
    <div
      className={cls(getFlagClasses(), className)}
      role="img"
      aria-label={ariaLabel}
      title={ariaLabel}
    >
      <span className="leading-none" style={{ fontSize: 'inherit', position: 'relative', top: '-1px' }}>
        {displayFlag}
      </span>
    </div>
  );
};

export default LanguageFlag;
