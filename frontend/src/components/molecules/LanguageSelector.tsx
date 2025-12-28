import React, { useState, useRef, useEffect } from 'react';
import LanguageFlag from '../atoms/LanguageFlag';
import { languages } from '../../mockData';
import { cls } from '../../utils/cls';
import { CSS } from '../../utils/constants';
import type { BaseComponentProps, LanguageData } from '../../types/components';

// Language selector size mapping
const SELECTOR_SIZE_MAP = {
  sm: 'text-sm px-3 py-1.5',
  md: 'text-sm px-4 py-2',
  lg: 'text-base px-4 py-3',
} as const;

// Variant types
export type LanguageSelectorVariant = 'dropdown' | 'modal';

// Component props interface
export interface LanguageSelectorProps extends BaseComponentProps {
  /** Currently selected language code */
  selectedLanguage?: string;
  /** Callback when language selection changes */
  onLanguageChange?: (languageCode: string) => void;
  /** Whether the selector is disabled */
  disabled?: boolean;
  /** Placeholder text when no language is selected */
  placeholder?: string;
  /** Selector variant */
  variant?: LanguageSelectorVariant;
  /** Whether to show search functionality */
  showSearch?: boolean;
  /** Size of the selector */
  size?: keyof typeof SELECTOR_SIZE_MAP;
}

const LanguageSelector: React.FC<LanguageSelectorProps> = ({
  selectedLanguage = 'en',
  onLanguageChange,
  disabled = false,
  placeholder = 'Select language',
  variant = 'dropdown',
  showSearch = true,
  size = 'md',
  className = ''
}) => {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [highlightedIndex, setHighlightedIndex] = useState<number>(-1);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const selectedLang = languages.find(lang => lang.code === selectedLanguage);

  const filteredLanguages = languages.filter(lang =>
    lang.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    lang.code.toLowerCase().includes(searchTerm.toLowerCase())
  );

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent): void => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearchTerm('');
        setHighlightedIndex(-1);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    if (isOpen && searchInputRef.current && showSearch) {
      searchInputRef.current.focus();
    }
  }, [isOpen, showSearch]);

  const handleKeyDown = (e: React.KeyboardEvent): void => {
    if (!isOpen) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        setIsOpen(true);
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex(prev =>
          prev < filteredLanguages.length - 1 ? prev + 1 : 0
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex(prev =>
          prev > 0 ? prev - 1 : filteredLanguages.length - 1
        );
        break;
      case 'Enter':
        e.preventDefault();
        if (highlightedIndex >= 0) {
          handleLanguageSelect(filteredLanguages[highlightedIndex]);
        }
        break;
      case 'Escape':
        setIsOpen(false);
        setSearchTerm('');
        setHighlightedIndex(-1);
        break;
    }
  };

  const handleLanguageSelect = (language: LanguageData): void => {
    if (onLanguageChange) {
      onLanguageChange(language.code);
    }
    setIsOpen(false);
    setSearchTerm('');
    setHighlightedIndex(-1);
  };

  const getTriggerClasses = (): string => {
    const baseClasses = [
      SELECTOR_SIZE_MAP[size],
      CSS.bgSurfaceBg,
      'border-2',
      'border-primary-blue-glow',
      CSS.roundedLg,
      CSS.flex,
      CSS.itemsCenter,
      CSS.justifyBetween,
      CSS.gap3,
      'cursor-pointer',
      CSS.transitionAll,
      CSS.duration200,
      'ease-in-out',
      'focus:outline-none',
      'focus:ring-4',
      'focus:ring-primary-blue-glow',
      'min-w-32',
    ];

    if (disabled) {
      return cls([
        ...baseClasses,
        CSS.bgInactiveDisabled,
        'border-inactive-disabled',
        CSS.textMuted,
        'cursor-not-allowed',
      ]);
    }

    if (isOpen) {
      return cls([
        ...baseClasses,
        'border-primary-blue',
        'ring-4',
        'ring-primary-blue-glow',
        'shadow-lg',
        'shadow-primary-blue-glow',
      ]);
    }

    return cls([
      ...baseClasses,
      'hover:border-primary-blue',
      'hover:shadow-md',
      'hover:shadow-primary-blue-glow',
      CSS.textPrimary,
    ]);
  };

  const getDropdownClasses = (): string => {
    return cls([
      'absolute',
      'top-full',
      'left-0',
      'right-0',
      'mt-1',
      CSS.bgContainerBg,
      'border-2',
      'border-primary-blue-glow',
      CSS.roundedLg,
      'shadow-lg',
      'shadow-primary-blue-glow',
      'z-50',
      'max-h-64',
      'overflow-hidden',
      isOpen ? 'opacity-100 scale-100' : 'opacity-0 scale-95 pointer-events-none',
      CSS.transitionAll,
      CSS.duration200,
      'ease-in-out',
    ]);
  };

  const getOptionClasses = (language: LanguageData, index: number): string => {
    const baseClasses = [
      CSS.px3,
      CSS.py2,
      CSS.flex,
      CSS.itemsCenter,
      CSS.gap3,
      'cursor-pointer',
      CSS.transitionColors,
      'duration-150',
      'ease-in-out',
    ];

    if (language.code === selectedLanguage) {
      return cls([
        ...baseClasses,
        CSS.bgPrimaryBlueGlow,
        CSS.textWhite,
      ]);
    }

    if (index === highlightedIndex) {
      return cls([
        ...baseClasses,
        CSS.bgPrimaryBlue,
        'bg-opacity-50',
        CSS.textPrimary,
      ]);
    }

    return cls([
      ...baseClasses,
      CSS.textPrimary,
      'hover:bg-primary-blue-glow',
      'hover:bg-opacity-30',
    ]);
  };

  if (variant === 'modal') {
    // Modal implementation would go here
    return null; // Placeholder for modal variant
  }

  return (
    <div className={cls('relative', className)} ref={dropdownRef}>
      <div
        className={getTriggerClasses()}
        onClick={() => !disabled && setIsOpen(!isOpen)}
        onKeyDown={handleKeyDown}
        tabIndex={disabled ? -1 : 0}
        role="combobox"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        aria-label="Language selector"
      >
        <div className={cls(CSS.flex, CSS.itemsCenter, CSS.gap3)}>
          {selectedLang ? (
            <>
              <LanguageFlag
                languageCode={selectedLang.code}
                flag={selectedLang.flag}
                size={size === 'lg' ? 'md' : 'sm'}
              />
              <span className={cls(CSS.fontMedium)}>{selectedLang.name}</span>
            </>
          ) : (
            <span className={cls(CSS.textMuted)}>
              {placeholder}
            </span>
          )}
        </div>

        <svg
          className={cls(
            'w-4',
            'h-4',
            CSS.transitionTransform,
            CSS.duration200,
            isOpen ? 'rotate-180' : ''
          )}
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
        </svg>
      </div>

      <div className={getDropdownClasses()}>
        {showSearch && (
          <div className={cls(CSS.p3, 'border-b', 'border-surface-200', 'dark:border-surface-700')}>
            <input
              ref={searchInputRef}
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search languages..."
              className={cls(
                CSS.wFull,
                CSS.px3,
                CSS.py2,
                CSS.textSm,
                CSS.bgSurfaceBg,
                'border',
                'border-primary-blue-glow',
                'rounded-md',
                'focus:outline-none',
                'focus:ring-2',
                'focus:ring-primary-blue',
                'focus:border-primary-blue'
              )}
            />
          </div>
        )}

        <div className={cls('max-h-48', 'overflow-y-auto')}>
          {filteredLanguages.length > 0 ? (
            filteredLanguages.map((language, index) => (
              <div
                key={language.code}
                className={getOptionClasses(language, index)}
                onClick={() => handleLanguageSelect(language)}
                onMouseEnter={() => setHighlightedIndex(index)}
                role="option"
                aria-selected={language.code === selectedLanguage}
              >
                <LanguageFlag
                  languageCode={language.code}
                  flag={language.flag}
                  size="sm"
                />
                <div className={cls('flex-1')}>
                  <div className={cls(CSS.fontMedium)}>{language.name}</div>
                  <div className={cls(CSS.textXs, CSS.textMuted)}>
                    {language.code.toUpperCase()}
                  </div>
                </div>
                {language.code === selectedLanguage && (
                  <svg className={cls('w-4', 'h-4', CSS.textPrimary)} fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                )}
              </div>
            ))
          ) : (
            <div className={cls(CSS.px3, 'py-4', CSS.textCenter, CSS.textMuted, CSS.textSm)}>
              No languages found
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LanguageSelector;
