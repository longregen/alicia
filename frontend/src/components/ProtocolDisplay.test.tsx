import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useEffect } from 'react';
import userEvent from '@testing-library/user-event';
import { ProtocolDisplay } from './ProtocolDisplay';
import { MessageProvider, useMessageContext } from '../contexts/MessageContext';
import { Severity } from '../types/protocol';

// Test wrapper component to populate context
function TestWrapper({ children, contextValue }: any) {
  const context = useMessageContext();

  // Populate context with test data
  useEffect(() => {
    if (contextValue.errors) {
      contextValue.errors.forEach((error: any) => context.addError(error));
    }
    if (contextValue.reasoningSteps) {
      contextValue.reasoningSteps.forEach((step: any) => context.addReasoningStep(step));
    }
    if (contextValue.toolUsages) {
      contextValue.toolUsages.forEach((usage: any) => context.addToolUsage(usage));
    }
    if (contextValue.memoryTraces) {
      contextValue.memoryTraces.forEach((trace: any) => context.addMemoryTrace(trace));
    }
    if (contextValue.commentaries) {
      contextValue.commentaries.forEach((commentary: any) => context.addCommentary(commentary));
    }
  }, []);

  return <>{children}</>;
}

describe('ProtocolDisplay', () => {
  describe('rendering', () => {
    it('should render nothing when no protocol messages exist', () => {
      const { container } = render(
        <MessageProvider>
          <ProtocolDisplay />
        </MessageProvider>
      );

      expect(container.querySelector('.protocol-display')).not.toBeInTheDocument();
    });
  });

  describe('error display', () => {
    it('should display error messages', () => {
      const mockError = {
        id: 'error-1',
        severity: Severity.Error,
        message: 'Something went wrong',
        code: 'ERR_001',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ errors: [mockError] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
      expect(screen.getByText('ERROR')).toBeInTheDocument();
      expect(screen.getByText('Code: ERR_001')).toBeInTheDocument();
    });

    it('should display severity badge with correct color for error', () => {
      const mockError = {
        id: 'error-1',
        severity: Severity.Error,
        message: 'Error message',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ errors: [mockError] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const badge = screen.getByText('ERROR');
      expect(badge).toHaveStyle({ backgroundColor: '#f44336' });
    });

    it('should display severity badge with correct color for warning', () => {
      const mockError = {
        id: 'warn-1',
        severity: Severity.Warning,
        message: 'Warning message',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ errors: [mockError] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const badge = screen.getByText('WARNING');
      expect(badge).toHaveStyle({ backgroundColor: '#ff9800' });
    });

    it('should display severity badge with correct color for info', () => {
      const mockError = {
        id: 'info-1',
        severity: Severity.Info,
        message: 'Info message',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ errors: [mockError] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const badge = screen.getByText('INFO');
      expect(badge).toHaveStyle({ backgroundColor: '#2196f3' });
    });

    it('should display severity badge with correct color for critical', () => {
      const mockError = {
        id: 'crit-1',
        severity: Severity.Critical,
        message: 'Critical error',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ errors: [mockError] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const badge = screen.getByText('CRITICAL');
      expect(badge).toHaveStyle({ backgroundColor: '#9c27b0' });
    });
  });

  describe('reasoning step rendering', () => {
    it('should display reasoning steps', () => {
      const mockSteps = [
        { id: 'step-1', sequence: 1, content: 'First thought' },
        { id: 'step-2', sequence: 2, content: 'Second thought' },
      ];

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ reasoningSteps: mockSteps }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Reasoning')).toBeInTheDocument();
      expect(screen.getByText('First thought')).toBeInTheDocument();
      expect(screen.getByText('Second thought')).toBeInTheDocument();
      expect(screen.getByText('Step 1')).toBeInTheDocument();
      expect(screen.getByText('Step 2')).toBeInTheDocument();
    });

    it('should display reasoning steps in correct order', () => {
      const mockSteps = [
        { id: 'step-2', sequence: 2, content: 'Second thought' },
        { id: 'step-1', sequence: 1, content: 'First thought' },
      ];

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ reasoningSteps: mockSteps }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const steps = screen.getAllByText(/Step \d/);
      expect(steps[0]).toHaveTextContent('Step 1');
      expect(steps[1]).toHaveTextContent('Step 2');
    });
  });

  describe('tool usage expansion', () => {
    it('should display tool usage with expandable details', async () => {
      const user = userEvent.setup();
      const mockToolUsage = {
        request: {
          id: 'tool-1',
          toolName: 'calculator',
          parameters: { operation: 'add', values: [1, 2] },
        },
        result: {
          requestId: 'tool-1',
          success: true,
          result: { answer: 3 },
        },
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ toolUsages: [mockToolUsage] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Tool Usage')).toBeInTheDocument();
      expect(screen.getByText('calculator')).toBeInTheDocument();
      expect(screen.getByText('Success')).toBeInTheDocument();

      // Expand parameters
      const parametersSummary = screen.getByText('Parameters');
      await user.click(parametersSummary);

      expect(screen.getByText(/"operation": "add"/)).toBeInTheDocument();
    });

    it('should display pending status for incomplete tool usage', () => {
      const mockToolUsage = {
        request: {
          id: 'tool-1',
          toolName: 'search',
          parameters: { query: 'test' },
        },
        result: null,
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ toolUsages: [mockToolUsage] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('search')).toBeInTheDocument();
      expect(screen.getByText('Pending...')).toBeInTheDocument();
    });

    it('should display failed status for failed tool usage', () => {
      const mockToolUsage = {
        request: {
          id: 'tool-1',
          toolName: 'api_call',
          parameters: { endpoint: '/test' },
        },
        result: {
          requestId: 'tool-1',
          success: false,
          errorMessage: 'Network error',
          errorCode: 'NET_001',
        },
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ toolUsages: [mockToolUsage] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('api_call')).toBeInTheDocument();
      expect(screen.getByText('Failed')).toBeInTheDocument();
    });

    it('should expand to show error details for failed tools', async () => {
      const user = userEvent.setup();
      const mockToolUsage = {
        request: {
          id: 'tool-1',
          toolName: 'api_call',
          parameters: { endpoint: '/test' },
        },
        result: {
          requestId: 'tool-1',
          success: false,
          errorMessage: 'Network error',
          errorCode: 'NET_001',
        },
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ toolUsages: [mockToolUsage] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      const resultSummary = screen.getByText('Result');
      await user.click(resultSummary);

      expect(screen.getByText('Error: Network error')).toBeInTheDocument();
      expect(screen.getByText('Code: NET_001')).toBeInTheDocument();
    });
  });

  describe('memory trace display', () => {
    it('should display memory traces', () => {
      const mockTrace = {
        id: 'trace-1',
        memoryId: 'mem-12345678',
        content: 'User prefers dark mode',
        relevance: 0.95,
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ memoryTraces: [mockTrace] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Retrieved Memories')).toBeInTheDocument();
      expect(screen.getByText('User prefers dark mode')).toBeInTheDocument();
      expect(screen.getByText('Memory mem-1234')).toBeInTheDocument();
      expect(screen.getByText('Relevance: 95%')).toBeInTheDocument();
    });

    it('should display relevance score correctly', () => {
      const mockTrace = {
        id: 'trace-1',
        memoryId: 'mem-123',
        content: 'Test memory',
        relevance: 0.7342,
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ memoryTraces: [mockTrace] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Relevance: 73%')).toBeInTheDocument();
    });
  });

  describe('commentary display', () => {
    it('should display commentaries', () => {
      const mockCommentary = {
        id: 'comment-1',
        content: 'Processing user request',
        commentaryType: 'status',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ commentaries: [mockCommentary] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('System Commentary')).toBeInTheDocument();
      expect(screen.getByText('Processing user request')).toBeInTheDocument();
      expect(screen.getByText('Type: status')).toBeInTheDocument();
    });

    it('should handle commentary without type', () => {
      const mockCommentary = {
        id: 'comment-1',
        content: 'General comment',
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={{ commentaries: [mockCommentary] }}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('General comment')).toBeInTheDocument();
      expect(screen.queryByText(/Type:/)).not.toBeInTheDocument();
    });
  });

  describe('multiple message types', () => {
    it('should display all protocol message types together', () => {
      const contextValue = {
        errors: [{ id: 'err-1', severity: Severity.Error, message: 'Error occurred' }],
        reasoningSteps: [{ id: 'step-1', sequence: 1, content: 'Thinking' }],
        toolUsages: [{
          request: { id: 'tool-1', toolName: 'calc', parameters: {} },
          result: null,
        }],
        memoryTraces: [{
          id: 'mem-1',
          memoryId: 'memory-123',
          content: 'Memory content',
          relevance: 0.8,
        }],
        commentaries: [{ id: 'com-1', content: 'Commentary' }],
      };

      render(
        <MessageProvider>
          <TestWrapper contextValue={contextValue}>
            <ProtocolDisplay />
          </TestWrapper>
        </MessageProvider>
      );

      expect(screen.getByText('Error occurred')).toBeInTheDocument();
      expect(screen.getByText('Reasoning')).toBeInTheDocument();
      expect(screen.getByText('Tool Usage')).toBeInTheDocument();
      expect(screen.getByText('Retrieved Memories')).toBeInTheDocument();
      expect(screen.getByText('System Commentary')).toBeInTheDocument();
    });
  });
});
