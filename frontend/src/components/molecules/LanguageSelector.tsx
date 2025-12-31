import React, { useState, useRef, useEffect } from 'react';
import LanguageFlag from '../atoms/LanguageFlag';
import { languages } from '../../mockData';
import { cls } from '../../utils/cls';
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
      'bg-surface',
      'border-2',
      'border-accent',
      'rounded-lg',
      'flex',
      'items-center',
      'justify-between',
      'gap-3',
      'cursor-pointer',
      'transition-all',
      'duration-200',
      'ease-in-out',
      'focus:outline-none',
      'focus:ring-4',
      'focus:ring-accent',
      'min-w-32',
    ];

    if (disabled) {
      return cls([
        ...baseClasses,
        'bg-sunken',
        'border-muted',
        'text-muted',
        'cursor-not-allowed',
      ]);
    }

    if (isOpen) {
      return cls([
        ...baseClasses,
        'border-accent',
        'ring-4',
        'ring-accent',
        'shadow-lg',
      ]);
    }

    return cls([
      ...baseClasses,
      'hover:border-accent-hover',
      'hover:shadow-md',
      'text-default',
    ]);
  };

  const getDropdownClasses = (): string => {
    return cls([
      'absolute',
      'top-full',
      'left-0',
      'right-0',
      'mt-1',
      'bg-elevated',
      'border-2',
      'border-accent',
      'rounded-lg',
      'shadow-lg',
      'z-50',
      'max-h-64',
      'overflow-hidden',
      isOpen ? 'opacity-100 scale-100' : 'opacity-0 scale-95 pointer-events-none',
      'transition-all',
      'duration-200',
      'ease-in-out',
    ]);
  };

  const getOptionClasses = (language: LanguageData, index: number): string => {
    const baseClasses = [
      'px-3',
      'py-2',
      'flex',
      'items-center',
      'gap-3',
      'cursor-pointer',
      'transition-colors',
      'duration-150',
      'ease-in-out',
    ];

    if (language.code === selectedLanguage) {
      return cls([
        ...baseClasses,
        'bg-accent',
        'text-on-emphasis',
      ]);
    }

    if (index === highlightedIndex) {
      return cls([
        ...baseClasses,
        'bg-accent-subtle',
        'text-default',
      ]);
    }

    return cls([
      ...baseClasses,
      'text-default',
      'hover:bg-accent-subtle',
    ]);
  };

  if (variant === 'modal') {
    console.warn('LanguageSelector: modal variant not yet implemented');
    return null;
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
        <div className="flex items-center gap-3">
          {selectedLang ? (
            <>
              <LanguageFlag
                languageCode={selectedLang.code}
                flag={selectedLang.flag}
                size={size === 'lg' ? 'md' : 'sm'}
              />
              <span className="font-medium">{selectedLang.name}</span>
            </>
          ) : (
            <span className="text-muted">
              {placeholder}
            </span>
          )}
        </div>

        <svg
          className={cls(
            'w-4 h-4 transition-transform duration-200',
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
          <div className="p-3 border-b border-muted">
            <input
              ref={searchInputRef}
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search languages..."
              className="w-full px-3 py-2 text-sm bg-surface border border-accent rounded-md focus:outline-none focus:ring-2 focus:ring-accent focus:border-accent"
            />
          </div>
        )}

        <div className="max-h-48 overflow-y-auto">
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
                <div className="flex-1">
                  <div className="font-medium">{language.name}</div>
                  <div className="text-xs text-muted">
                    {language.code.toUpperCase()}
                  </div>
                </div>
                {language.code === selectedLanguage && (
                  <svg className="w-4 h-4 text-default" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                  </svg>
                )}
              </div>
            ))
          ) : (
            <div className="px-3 py-4 text-center text-muted text-sm">
              No languages found
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LanguageSelector;
