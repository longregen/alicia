import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ToolUsageDisplay } from './ToolUsageDisplay';
import { ToolUsage } from '../contexts/MessageContext';

describe('ToolUsageDisplay', () => {
  const mockToolUsage: ToolUsage = {
    request: {
      id: 'tool-1',
      messageId: 'msg-1',
      conversationId: 'conv-123',
      toolName: 'testTool',
      parameters: { param1: 'value1', param2: 123 },
      execution: 'server' as const,
    },
    result: {
      id: 'result-1',
      requestId: 'tool-1',
      conversationId: 'conv-123',
      success: true,
      result: { output: 'success data' },
    },
  };

  it('should render nothing when toolUsages is empty', () => {
    const { container } = render(<ToolUsageDisplay toolUsages={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('should render tool usage with success status', () => {
    render(<ToolUsageDisplay toolUsages={[mockToolUsage]} />);

    expect(screen.getByText('testTool')).toBeInTheDocument();
    expect(screen.getByText('Success')).toBeInTheDocument();
  });

  it('should render tool usage with pending status', () => {
    const pendingUsage: ToolUsage = {
      request: mockToolUsage.request,
      result: null,
    };

    render(<ToolUsageDisplay toolUsages={[pendingUsage]} />);

    expect(screen.getByText('testTool')).toBeInTheDocument();
    expect(screen.getByText('Running...')).toBeInTheDocument();
  });

  it('should render tool usage with failed status', () => {
    const failedUsage: ToolUsage = {
      request: mockToolUsage.request,
      result: {
        id: 'result-1',
        requestId: 'tool-1',
        conversationId: 'conv-123',
        success: false,
        errorMessage: 'Something went wrong',
        errorCode: 'ERR_TEST',
      },
    };

    render(<ToolUsageDisplay toolUsages={[failedUsage]} />);

    expect(screen.getByText('testTool')).toBeInTheDocument();
    expect(screen.getByText('Failed')).toBeInTheDocument();
  });
});
