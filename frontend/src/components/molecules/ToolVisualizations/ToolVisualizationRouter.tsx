import React from 'react';
import WebReadVisualization from './WebReadVisualization';
import WebSearchVisualization from './WebSearchVisualization';
import WebFetchVisualization from './WebFetchVisualization';
import WebLinksVisualization from './WebLinksVisualization';
import WebMetadataVisualization from './WebMetadataVisualization';
import WebScreenshotVisualization from './WebScreenshotVisualization';
import GardenTableVisualization from './GardenTableVisualization';
import GardenSQLVisualization from './GardenSQLVisualization';
import GardenSchemaVisualization from './GardenSchemaVisualization';
import { toolIcons, toolDisplayNames } from './index';
import { cls } from '../../../utils/cls';

interface ToolVisualizationRouterProps {
  toolName: string;
  result: unknown;
  className?: string;
}

/**
 * ToolVisualizationRouter automatically selects the appropriate visualization
 * component based on the tool name and renders it with the result data.
 */
const ToolVisualizationRouter: React.FC<ToolVisualizationRouterProps> = ({
  toolName,
  result,
  className,
}) => {
  // Handle string results (JSON strings that need parsing)
  let parsedResult = result;
  if (typeof result === 'string') {
    try {
      parsedResult = JSON.parse(result);
    } catch {
      // Keep as string if not valid JSON
    }
  }

  // Route to appropriate visualization
  switch (toolName) {
    case 'web_read':
      return <WebReadVisualization result={parsedResult as any} className={className} />;

    case 'web_search':
      return <WebSearchVisualization result={parsedResult as any} className={className} />;

    case 'web_fetch_raw':
      return <WebFetchVisualization result={parsedResult as any} type="raw" className={className} />;

    case 'web_fetch_structured':
      return <WebFetchVisualization result={parsedResult as any} type="structured" className={className} />;

    case 'web_extract_links':
      return <WebLinksVisualization result={parsedResult as any} className={className} />;

    case 'web_extract_metadata':
      return <WebMetadataVisualization result={parsedResult as any} className={className} />;

    case 'web_screenshot':
      return <WebScreenshotVisualization result={parsedResult as any} className={className} />;

    case 'garden_describe_table':
      return <GardenTableVisualization result={parsedResult as any} className={className} />;

    case 'garden_execute_sql':
      return <GardenSQLVisualization result={parsedResult as any} className={className} />;

    case 'garden_schema_explore':
      return <GardenSchemaVisualization result={parsedResult as any} className={className} />;

    default:
      // Fallback for unknown tools - render a generic card
      return <GenericToolVisualization toolName={toolName} result={parsedResult} className={className} />;
  }
};

/**
 * Generic visualization for tools without a specific component
 */
interface GenericToolVisualizationProps {
  toolName: string;
  result: unknown;
  className?: string;
}

const GenericToolVisualization: React.FC<GenericToolVisualizationProps> = ({
  toolName,
  result,
  className,
}) => {
  const icon = toolIcons[toolName] || '🔧';
  const displayName = toolDisplayNames[toolName] || toolName;
  const [isExpanded, setIsExpanded] = React.useState(false);

  const resultString = typeof result === 'string' ? result : JSON.stringify(result, null, 2);
  const preview = resultString.slice(0, 300);
  const hasMore = resultString.length > 300;

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-gray-50 to-slate-50 dark:from-gray-950/30 dark:to-slate-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">{icon}</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              {displayName}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 font-mono">
              {toolName}
            </p>
          </div>
        </div>
      </div>

      {/* Result */}
      <div className="p-4">
        <pre className="bg-white/50 dark:bg-black/20 rounded-lg p-3 overflow-auto max-h-80 text-xs font-mono text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
          {isExpanded ? resultString : preview}
          {!isExpanded && hasMore && '...'}
        </pre>
        {hasMore && (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="mt-2 text-xs text-blue-600 dark:text-blue-400 hover:underline"
          >
            {isExpanded ? '← Show less' : 'Show full result'}
          </button>
        )}
      </div>
    </div>
  );
};

export default ToolVisualizationRouter;
