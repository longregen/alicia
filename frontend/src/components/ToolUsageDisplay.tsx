import { useState } from 'react';
import { ToolUsage } from '../contexts/MessageContext';
import './ToolUsageDisplay.css';

interface ToolUsageDisplayProps {
  toolUsages: ToolUsage[];
  isLatestMessage?: boolean;
}

export function ToolUsageDisplay({ toolUsages, isLatestMessage = false }: ToolUsageDisplayProps) {
  // Track which tool usage items are expanded
  const [expandedItems, setExpandedItems] = useState<Set<string>>(
    // By default, expand all items if this is the latest message
    isLatestMessage ? new Set(toolUsages.map(u => u.request.id)) : new Set()
  );

  const toggleExpanded = (id: string) => {
    setExpandedItems(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  if (toolUsages.length === 0) {
    return null;
  }

  return (
    <div className="tool-usage-list">
      {toolUsages.map((usage) => {
        const isExpanded = expandedItems.has(usage.request.id);
        const hasResult = usage.result !== null;
        const isSuccess = usage.result !== null && usage.result.success;
        const isPending = !hasResult;

        return (
          <div key={usage.request.id} className="tool-usage-item">
            <div
              className="tool-usage-header"
              onClick={() => toggleExpanded(usage.request.id)}
            >
              <div className="tool-header-left">
                <span className="tool-icon">🔧</span>
                <span className="tool-name">{usage.request.toolName}</span>
              </div>
              <div className="tool-header-right">
                <span
                  className={`tool-status ${isPending ? 'pending' : isSuccess ? 'success' : 'failed'}`}
                >
                  {isPending ? 'Running...' : isSuccess ? 'Success' : 'Failed'}
                </span>
                <span className={`expand-icon ${isExpanded ? 'expanded' : ''}`}>
                  ▼
                </span>
              </div>
            </div>

            {isExpanded && (
              <div className="tool-usage-body">
                {/* Parameters section */}
                <div className="tool-section">
                  <div className="tool-section-title">Parameters</div>
                  <pre className="tool-code">
                    {JSON.stringify(usage.request.parameters, null, 2)}
                  </pre>
                </div>

                {/* Result section */}
                {hasResult && (
                  <div className="tool-section">
                    <div className="tool-section-title">
                      {isSuccess ? 'Result' : 'Error'}
                    </div>
                    {isSuccess ? (
                      <pre className="tool-code">
                        {JSON.stringify(usage.result?.result, null, 2)}
                      </pre>
                    ) : (
                      <div className="tool-error">
                        <div className="error-message">
                          {usage.result?.errorMessage}
                        </div>
                        {usage.result?.errorCode && (
                          <div className="error-code">
                            Code: {usage.result.errorCode}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
