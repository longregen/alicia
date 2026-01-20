import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface WebReadResult {
  url: string;
  title?: string;
  content: string;
  excerpt?: string;
  author?: string;
  site_name?: string;
  word_count: number;
  estimated_tokens: number;
}

interface WebReadVisualizationProps {
  result: WebReadResult;
  className?: string;
}

const WebReadVisualization: React.FC<WebReadVisualizationProps> = ({ result, className }) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const contentPreview = result.content.slice(0, 500);
  const hasMore = result.content.length > 500;

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/30 dark:to-indigo-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-2xl">📖</span>
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100 truncate">
              {result.title || 'Web Page'}
            </h3>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-blue-600 dark:text-blue-400 hover:underline truncate block"
            >
              {result.url}
            </a>
          </div>
        </div>

        {/* Metadata badges */}
        <div className="flex flex-wrap gap-2 mt-2">
          {result.site_name && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-200">
              {result.site_name}
            </span>
          )}
          {result.author && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-200">
              ✍️ {result.author}
            </span>
          )}
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200">
            {result.word_count.toLocaleString()} words
          </span>
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-200">
            ~{result.estimated_tokens.toLocaleString()} tokens
          </span>
        </div>
      </div>

      {/* Excerpt */}
      {result.excerpt && (
        <div className="px-4 py-2 bg-white/30 dark:bg-black/10 border-b">
          <p className="text-xs text-gray-600 dark:text-gray-400 italic">
            "{result.excerpt}"
          </p>
        </div>
      )}

      {/* Content */}
      <div className="p-4">
        <div className="prose prose-sm dark:prose-invert max-w-none">
          <pre className="whitespace-pre-wrap font-sans text-sm text-gray-700 dark:text-gray-300 bg-white/50 dark:bg-black/20 rounded-lg p-3 overflow-auto max-h-96">
            {isExpanded ? result.content : contentPreview}
            {!isExpanded && hasMore && '...'}
          </pre>
        </div>

        {hasMore && (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="mt-2 text-sm text-blue-600 dark:text-blue-400 hover:underline focus:outline-none"
          >
            {isExpanded ? '← Show less' : `Show full content (${result.content.length.toLocaleString()} chars)`}
          </button>
        )}
      </div>
    </div>
  );
};

export default WebReadVisualization;
