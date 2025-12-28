import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import LanguageFlag from './LanguageFlag';

describe('LanguageFlag', () => {
  describe('Basic Rendering', () => {
    it('renders with default props', () => {
      render(<LanguageFlag />);

      const flag = screen.getByRole('img');
      expect(flag).toBeInTheDocument();
    });

    it('renders English flag by default', () => {
      render(<LanguageFlag />);

      expect(screen.getByText('🇺🇸')).toBeInTheDocument();
    });

    it('applies custom className', () => {
      render(<LanguageFlag className="custom-class" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('custom-class');
    });
  });

  describe('Language Code Mapping', () => {
    const languageTests = [
      { code: 'en', flag: '🇺🇸', name: 'English' },
      { code: 'es', flag: '🇪🇸', name: 'Spanish' },
      { code: 'fr', flag: '🇫🇷', name: 'French' },
      { code: 'de', flag: '🇩🇪', name: 'German' },
      { code: 'it', flag: '🇮🇹', name: 'Italian' },
      { code: 'pt', flag: '🇵🇹', name: 'Portuguese' },
      { code: 'ru', flag: '🇷🇺', name: 'Russian' },
      { code: 'zh', flag: '🇨🇳', name: 'Chinese' },
      { code: 'ja', flag: '🇯🇵', name: 'Japanese' },
    ];

    languageTests.forEach(({ code, flag, name }) => {
      it(`renders ${flag} for language code "${code}"`, () => {
        render(<LanguageFlag languageCode={code} />);

        expect(screen.getByText(flag)).toBeInTheDocument();
      });

      it(`has correct aria-label for "${code}"`, () => {
        render(<LanguageFlag languageCode={code} />);

        const flagElement = screen.getByRole('img');
        expect(flagElement).toHaveAttribute('aria-label', name);
      });
    });

    it('renders globe icon for auto-detect', () => {
      render(<LanguageFlag languageCode="auto" />);

      expect(screen.getByText('🌐')).toBeInTheDocument();
    });

    it('has correct aria-label for auto-detect', () => {
      render(<LanguageFlag languageCode="auto" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveAttribute('aria-label', 'Auto-detect');
    });

    it('renders fallback flag for unknown language code', () => {
      render(<LanguageFlag languageCode="unknown" />);

      expect(screen.getByText('🏳️')).toBeInTheDocument();
    });

    it('has fallback aria-label for unknown language code', () => {
      render(<LanguageFlag languageCode="xyz" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveAttribute('aria-label', 'Language: xyz');
    });
  });

  describe('Custom Flag Override', () => {
    it('uses provided flag over language code mapping', () => {
      render(<LanguageFlag languageCode="en" flag="🏴󠁧󠁢󠁥󠁮󠁧󠁿" />);

      expect(screen.getByText('🏴󠁧󠁢󠁥󠁮󠁧󠁿')).toBeInTheDocument();
      expect(screen.queryByText('🇺🇸')).not.toBeInTheDocument();
    });

    it('uses provided flag for auto-detect', () => {
      render(<LanguageFlag languageCode="auto" flag="🎯" />);

      expect(screen.getByText('🎯')).toBeInTheDocument();
      expect(screen.queryByText('🌐')).not.toBeInTheDocument();
    });
  });

  describe('Size Variants', () => {
    it('renders small size', () => {
      render(<LanguageFlag size="sm" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('w-4');
      expect(flag).toHaveClass('h-4');
      expect(flag).toHaveClass('text-xs');
    });

    it('renders medium size by default', () => {
      render(<LanguageFlag />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('w-6');
      expect(flag).toHaveClass('h-6');
      expect(flag).toHaveClass('text-sm');
    });

    it('renders large size', () => {
      render(<LanguageFlag size="lg" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('w-8');
      expect(flag).toHaveClass('h-8');
      expect(flag).toHaveClass('text-base');
    });
  });

  describe('Accessibility', () => {
    it('has role="img"', () => {
      render(<LanguageFlag />);

      expect(screen.getByRole('img')).toBeInTheDocument();
    });

    it('has aria-label attribute', () => {
      render(<LanguageFlag languageCode="fr" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveAttribute('aria-label', 'French');
    });

    it('has title attribute matching aria-label', () => {
      render(<LanguageFlag languageCode="de" />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveAttribute('title', 'German');
    });
  });

  describe('Styling', () => {
    it('has rounded-full class', () => {
      render(<LanguageFlag />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('rounded-full');
    });

    it('has flex centering classes', () => {
      render(<LanguageFlag />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('flex');
      expect(flag).toHaveClass('items-center');
      expect(flag).toHaveClass('justify-center');
    });

    it('has select-none to prevent text selection', () => {
      render(<LanguageFlag />);

      const flag = screen.getByRole('img');
      expect(flag).toHaveClass('select-none');
    });
  });
});
