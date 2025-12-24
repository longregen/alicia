import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MessageBubble } from './MessageBubble';
import { Message } from '../types/models';

describe('MessageBubble', () => {
  const baseMessage: Message = {
    id: 'msg-1',
    conversation_id: 'conv-123',
    sequence_number: 1,
    role: 'user',
    contents: 'Test message',
    created_at: '2024-01-01T12:00:00Z',
    updated_at: '2024-01-01T12:00:00Z',
  };

  describe('rendering', () => {
    it('should render user message correctly', () => {
      render(<MessageBubble message={baseMessage} />);

      expect(screen.getByText('You')).toBeInTheDocument();
      expect(screen.getByText('Test message')).toBeInTheDocument();
    });

    it('should render assistant message correctly', () => {
      const assistantMessage: Message = {
        ...baseMessage,
        role: 'assistant',
        contents: 'Hello, how can I help you?',
      };

      render(<MessageBubble message={assistantMessage} />);

      expect(screen.getByText('Alicia')).toBeInTheDocument();
      expect(screen.getByText('Hello, how can I help you?')).toBeInTheDocument();
    });

    it('should display formatted timestamp', () => {
      render(<MessageBubble message={baseMessage} />);

      // The timestamp should be formatted as a locale time string
      const timeElement = screen.getByText((content) => {
        // Check if it looks like a time (contains : and numbers)
        return /\d{1,2}:\d{2}/.test(content);
      });

      expect(timeElement).toBeInTheDocument();
    });
  });

  describe('CSS classes', () => {
    it('should apply user role class for user messages', () => {
      const { container } = render(<MessageBubble message={baseMessage} />);

      const bubble = container.querySelector('.message-bubble');
      expect(bubble).toHaveClass('user');
    });

    it('should apply assistant role class for assistant messages', () => {
      const assistantMessage: Message = {
        ...baseMessage,
        role: 'assistant',
      };

      const { container } = render(<MessageBubble message={assistantMessage} />);

      const bubble = container.querySelector('.message-bubble');
      expect(bubble).toHaveClass('assistant');
    });

    it('should have correct structure with role, content, and time elements', () => {
      const { container } = render(<MessageBubble message={baseMessage} />);

      expect(container.querySelector('.message-bubble')).toBeInTheDocument();
      expect(container.querySelector('.message-role')).toBeInTheDocument();
      expect(container.querySelector('.message-content')).toBeInTheDocument();
      expect(container.querySelector('.message-time')).toBeInTheDocument();
    });
  });

  describe('content display', () => {
    it('should handle empty message content', () => {
      const emptyMessage: Message = {
        ...baseMessage,
        contents: '',
      };

      render(<MessageBubble message={emptyMessage} />);

      const contentElement = screen.getByText((content, element) => {
        return element?.className === 'message-content' && content === '';
      });

      expect(contentElement).toBeInTheDocument();
    });

    it('should handle long message content', () => {
      const longMessage: Message = {
        ...baseMessage,
        contents: 'A'.repeat(1000),
      };

      render(<MessageBubble message={longMessage} />);

      expect(screen.getByText('A'.repeat(1000))).toBeInTheDocument();
    });

    it('should handle message with newlines', () => {
      const multilineMessage: Message = {
        ...baseMessage,
        contents: 'Line 1\nLine 2\nLine 3',
      };

      const { container } = render(<MessageBubble message={multilineMessage} />);

      const contentElement = container.querySelector('.message-content');
      expect(contentElement?.textContent).toBe('Line 1\nLine 2\nLine 3');
    });

    it('should handle special characters in content', () => {
      const specialCharMessage: Message = {
        ...baseMessage,
        contents: '<script>alert("xss")</script> & "quotes" \'apostrophes\'',
      };

      render(<MessageBubble message={specialCharMessage} />);

      // React automatically escapes HTML, so the raw text should be displayed
      expect(
        screen.getByText('<script>alert("xss")</script> & "quotes" \'apostrophes\'')
      ).toBeInTheDocument();
    });
  });

  describe('timestamp formatting', () => {
    it('should format timestamp correctly for different times', () => {
      const morningMessage: Message = {
        ...baseMessage,
        created_at: '2024-01-01T09:30:00Z',
      };

      render(<MessageBubble message={morningMessage} />);

      // Just verify a timestamp is displayed - exact format depends on locale
      const timeElement = screen.getByText((content) => /\d{1,2}:\d{2}/.test(content));
      expect(timeElement).toBeInTheDocument();
    });

    it('should handle invalid timestamp gracefully', () => {
      const invalidMessage: Message = {
        ...baseMessage,
        created_at: 'invalid-date',
      };

      // Should not throw an error
      expect(() => render(<MessageBubble message={invalidMessage} />)).not.toThrow();
    });
  });

  describe('message roles', () => {
    it('should display "You" for user role', () => {
      const userMessage: Message = {
        ...baseMessage,
        role: 'user',
      };

      render(<MessageBubble message={userMessage} />);

      expect(screen.getByText('You')).toBeInTheDocument();
    });

    it('should display "Alicia" for assistant role', () => {
      const assistantMessage: Message = {
        ...baseMessage,
        role: 'assistant',
      };

      render(<MessageBubble message={assistantMessage} />);

      expect(screen.getByText('Alicia')).toBeInTheDocument();
    });

    it('should handle system role if present', () => {
      const systemMessage: Message = {
        ...baseMessage,
        role: 'system',
      };

      const { container } = render(<MessageBubble message={systemMessage} />);

      // System messages use the ternary operator which defaults to 'Alicia' for non-user roles
      // The current implementation shows: message.role === 'user' ? 'You' : 'Alicia'
      const roleElement = container.querySelector('.message-role');
      expect(roleElement?.textContent).toBe('Alicia');
    });
  });

  describe('accessibility', () => {
    it('should render message content in accessible structure', () => {
      render(<MessageBubble message={baseMessage} />);

      const bubble = screen.getByText('Test message').closest('.message-bubble');
      expect(bubble).toBeInTheDocument();
    });

    it('should have readable text content', () => {
      const { container } = render(<MessageBubble message={baseMessage} />);

      // All text should be accessible
      expect(container.textContent).toContain('You');
      expect(container.textContent).toContain('Test message');
    });
  });

  describe('edge cases', () => {
    it('should handle messages with all optional fields', () => {
      const minimalMessage: Message = {
        id: 'msg-1',
        conversation_id: 'conv-123',
        sequence_number: 1,
        role: 'user',
        contents: 'Minimal message',
        created_at: '2024-01-01T12:00:00Z',
        updated_at: '2024-01-01T12:00:00Z',
      };

      expect(() => render(<MessageBubble message={minimalMessage} />)).not.toThrow();
    });

    it('should handle messages with additional metadata', () => {
      const messageWithMetadata: Message = {
        ...baseMessage,
        previous_id: 'msg-0',
        local_id: 'local-1',
        server_id: 'server-1',
        sync_status: 'synced',
        synced_at: '2024-01-01T12:01:00Z',
      };

      expect(() => render(<MessageBubble message={messageWithMetadata} />)).not.toThrow();
      expect(screen.getByText('Test message')).toBeInTheDocument();
    });
  });
});
