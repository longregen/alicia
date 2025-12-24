import { useMessageContext } from '../contexts/MessageContext';
import { Severity } from '../types/protocol';
import './ProtocolDisplay.css';

export function ProtocolDisplay() {
  const {
    errors,
    reasoningSteps,
    toolUsages,
    memoryTraces,
    commentaries,
  } = useMessageContext();

  // Get the severity color for errors
  const getSeverityColor = (severity: Severity): string => {
    switch (severity) {
      case Severity.Info:
        return '#2196f3';
      case Severity.Warning:
        return '#ff9800';
      case Severity.Error:
        return '#f44336';
      case Severity.Critical:
        return '#9c27b0';
      default:
        return '#757575';
    }
  };

  const getSeverityLabel = (severity: Severity): string => {
    switch (severity) {
      case Severity.Info:
        return 'INFO';
      case Severity.Warning:
        return 'WARNING';
      case Severity.Error:
        return 'ERROR';
      case Severity.Critical:
        return 'CRITICAL';
      default:
        return 'UNKNOWN';
    }
  };

  const hasProtocolMessages = errors.length > 0 ||
    reasoningSteps.length > 0 ||
    toolUsages.length > 0 ||
    memoryTraces.length > 0 ||
    commentaries.length > 0;

  if (!hasProtocolMessages) {
    return null;
  }

  return (
    <div className="protocol-display">
      {/* Error Messages */}
      {errors.map((error) => (
        <div
          key={error.id}
          className="protocol-item error-message"
          style={{ borderLeftColor: getSeverityColor(error.severity) }}
        >
          <div className="protocol-header">
            <span
              className="severity-badge"
              style={{ backgroundColor: getSeverityColor(error.severity) }}
            >
              {getSeverityLabel(error.severity)}
            </span>
            <span className="protocol-type">Error</span>
          </div>
          <div className="protocol-content">{error.message}</div>
          {error.code && (
            <div className="protocol-meta">Code: {error.code}</div>
          )}
        </div>
      ))}

      {/* Reasoning Steps */}
      {reasoningSteps.length > 0 && (
        <div className="protocol-section">
          <div className="section-title">Reasoning</div>
          {reasoningSteps.map((step) => (
            <div key={step.id} className="protocol-item reasoning-step">
              <div className="protocol-header">
                <span className="step-number">Step {step.sequence}</span>
              </div>
              <div className="protocol-content">{step.content}</div>
            </div>
          ))}
        </div>
      )}

      {/* Tool Usage */}
      {toolUsages.length > 0 && (
        <div className="protocol-section">
          <div className="section-title">Tool Usage</div>
          {toolUsages.map((usage, index) => (
            <div key={usage.request.id || index} className="protocol-item tool-usage">
              <div className="protocol-header">
                <span className="tool-name">{usage.request.toolName}</span>
                <span
                  className={`tool-status ${usage.result ? (usage.result.success ? 'success' : 'failed') : 'pending'}`}
                >
                  {usage.result ? (usage.result.success ? 'Success' : 'Failed') : 'Pending...'}
                </span>
              </div>
              <div className="protocol-content">
                <details>
                  <summary>Parameters</summary>
                  <pre>{JSON.stringify(usage.request.parameters, null, 2)}</pre>
                </details>
                {usage.result && (
                  <details>
                    <summary>Result</summary>
                    {usage.result.success ? (
                      <pre>{JSON.stringify(usage.result.result, null, 2)}</pre>
                    ) : (
                      <div className="error-result">
                        <div>Error: {usage.result.errorMessage}</div>
                        {usage.result.errorCode && <div>Code: {usage.result.errorCode}</div>}
                      </div>
                    )}
                  </details>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Memory Traces */}
      {memoryTraces.length > 0 && (
        <div className="protocol-section">
          <div className="section-title">Retrieved Memories</div>
          {memoryTraces.map((trace) => (
            <div key={trace.id} className="protocol-item memory-trace">
              <div className="protocol-header">
                <span className="memory-id">
                  Memory {typeof trace.memoryId === 'string' ? trace.memoryId.slice(0, 8) : String(trace.memoryId).slice(0, 8)}
                </span>
                <span className="relevance-score">
                  Relevance: {(trace.relevance * 100).toFixed(0)}%
                </span>
              </div>
              <div className="protocol-content">{trace.content}</div>
            </div>
          ))}
        </div>
      )}

      {/* Commentaries */}
      {commentaries.length > 0 && (
        <div className="protocol-section">
          <div className="section-title">System Commentary</div>
          {commentaries.map((commentary) => (
            <div key={commentary.id} className="protocol-item commentary">
              <div className="protocol-content">{commentary.content}</div>
              {commentary.commentaryType && (
                <div className="protocol-meta">Type: {commentary.commentaryType}</div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
