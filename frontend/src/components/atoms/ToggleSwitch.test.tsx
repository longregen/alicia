import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ToggleSwitch from './ToggleSwitch';

describe('ToggleSwitch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders with default props', () => {
      render(<ToggleSwitch />);

      const toggle = screen.getByRole('button');
      expect(toggle).toBeInTheDocument();
      expect(toggle).toHaveAttribute('aria-pressed', 'false');
    });

    it('renders with label', () => {
      render(<ToggleSwitch label="Test Label" />);

      expect(screen.getByText('Test Label')).toBeInTheDocument();
    });

    it('renders checked state', () => {
      render(<ToggleSwitch checked={true} />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-pressed', 'true');
    });

    it('renders unchecked state', () => {
      render(<ToggleSwitch checked={false} />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-pressed', 'false');
    });

    it('applies custom className', () => {
      const { container } = render(<ToggleSwitch className="custom-class" />);

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('Controlled Mode', () => {
    it('uses controlled value when checked prop is provided', () => {
      const { rerender } = render(<ToggleSwitch checked={false} />);

      let toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-pressed', 'false');

      rerender(<ToggleSwitch checked={true} />);

      toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-pressed', 'true');
    });

    it('calls onChange when toggled', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch checked={false} onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.click(toggle);

      expect(onChange).toHaveBeenCalledWith(true);
    });

    it('does not change internal state in controlled mode', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch checked={false} onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.click(toggle);

      // Still shows false because controlled
      expect(toggle).toHaveAttribute('aria-pressed', 'false');
    });
  });

  describe('Uncontrolled Mode', () => {
    it('manages its own state when checked prop is not provided', () => {
      render(<ToggleSwitch />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-pressed', 'false');

      fireEvent.click(toggle);

      expect(toggle).toHaveAttribute('aria-pressed', 'true');
    });

    it('toggles back to false on second click', () => {
      render(<ToggleSwitch />);

      const toggle = screen.getByRole('button');

      fireEvent.click(toggle);
      expect(toggle).toHaveAttribute('aria-pressed', 'true');

      fireEvent.click(toggle);
      expect(toggle).toHaveAttribute('aria-pressed', 'false');
    });

    it('calls onChange in uncontrolled mode', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.click(toggle);

      expect(onChange).toHaveBeenCalledWith(true);
    });
  });

  describe('Disabled State', () => {
    it('does not toggle when disabled', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch disabled onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.click(toggle);

      expect(onChange).not.toHaveBeenCalled();
    });

    it('has aria-disabled attribute when disabled', () => {
      render(<ToggleSwitch disabled />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-disabled', 'true');
    });

    it('has negative tabIndex when disabled', () => {
      render(<ToggleSwitch disabled />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('tabIndex', '-1');
    });

    it('applies disabled styling', () => {
      const { container } = render(<ToggleSwitch disabled />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('cursor-not-allowed');
    });
  });

  describe('Keyboard Interaction', () => {
    it('toggles on Enter key', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.keyDown(toggle, { key: 'Enter' });

      expect(onChange).toHaveBeenCalledWith(true);
    });

    it('toggles on Space key', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.keyDown(toggle, { key: ' ' });

      expect(onChange).toHaveBeenCalledWith(true);
    });

    it('does not toggle on other keys', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.keyDown(toggle, { key: 'Tab' });

      expect(onChange).not.toHaveBeenCalled();
    });

    it('does not toggle on keyboard when disabled', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch disabled onChange={onChange} />);

      const toggle = screen.getByRole('button');
      fireEvent.keyDown(toggle, { key: 'Enter' });

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('Label Click', () => {
    it('toggles when label is clicked', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch label="Click me" onChange={onChange} />);

      const label = screen.getByText('Click me');
      fireEvent.click(label);

      expect(onChange).toHaveBeenCalledWith(true);
    });

    it('does not toggle label click when disabled', () => {
      const onChange = vi.fn();
      render(<ToggleSwitch label="Click me" disabled onChange={onChange} />);

      const label = screen.getByText('Click me');
      fireEvent.click(label);

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('Size Variants', () => {
    it('renders small size', () => {
      const { container } = render(<ToggleSwitch size="sm" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('w-8');
      expect(track).toHaveClass('h-4');
    });

    it('renders medium size by default', () => {
      const { container } = render(<ToggleSwitch />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('w-11');
      expect(track).toHaveClass('h-6');
    });

    it('renders large size', () => {
      const { container } = render(<ToggleSwitch size="lg" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('w-14');
      expect(track).toHaveClass('h-7');
    });
  });

  describe('Visual Variants', () => {
    it('applies default variant styling when checked', () => {
      const { container } = render(<ToggleSwitch checked={true} variant="default" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('bg-primary-blue');
    });

    it('applies success variant styling when checked', () => {
      const { container } = render(<ToggleSwitch checked={true} variant="success" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('bg-active-speaking');
    });

    it('applies warning variant styling when checked', () => {
      const { container } = render(<ToggleSwitch checked={true} variant="warning" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('bg-tool-result');
    });

    it('applies error variant styling when checked', () => {
      const { container } = render(<ToggleSwitch checked={true} variant="error" />);

      const track = container.querySelector('[role="button"]');
      expect(track).toHaveClass('bg-error');
    });
  });

  describe('Accessibility', () => {
    it('has correct aria-label with label prop', () => {
      render(<ToggleSwitch label="Test Toggle" />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-label', 'Test Toggle');
    });

    it('has default aria-label without label prop', () => {
      render(<ToggleSwitch />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-label', 'Toggle switch off');
    });

    it('updates aria-label based on checked state', () => {
      render(<ToggleSwitch checked={true} />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('aria-label', 'Toggle switch on');
    });

    it('has focusable toggle', () => {
      render(<ToggleSwitch />);

      const toggle = screen.getByRole('button');
      expect(toggle).toHaveAttribute('tabIndex', '0');
    });
  });
});
