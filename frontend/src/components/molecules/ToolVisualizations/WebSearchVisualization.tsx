import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface SearchHit {
  title: string;
  url: string;
  snippet: string;
  content?: string;
}

interface WebSearchResult {
  query: string;
  result_count: number;
  results: SearchHit[];
}

interface WebSearchVisualizationProps {
  result: WebSearchResult;
  className?: string;
}

const WebSearchVisualization: React.FC<WebSearchVisualizationProps> = ({ result, className }) => {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-amber-50 to-orange-50 dark:from-amber-950/30 dark:to-orange-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">🔎</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              Web Search
            </h3>
            <p className="text-xs text-gray-600 dark:text-gray-400">
              "{result.query}" • {result.result_count} results
            </p>
          </div>
        </div>
      </div>

      {/* Results */}
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        {result.results.map((hit, index) => (
          <div
            key={index}
            className="p-4 hover:bg-white/30 dark:hover:bg-black/10 transition-colors"
          >
            <div className="flex items-start gap-3">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-amber-200 dark:bg-amber-800 text-amber-800 dark:text-amber-200 flex items-center justify-center text-xs font-bold">
                {index + 1}
              </span>
              <div className="flex-1 min-w-0">
                <a
                  href={hit.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium text-sm text-blue-600 dark:text-blue-400 hover:underline block truncate"
                >
                  {hit.title || hit.url}
                </a>
                <p className="text-xs text-gray-500 dark:text-gray-500 truncate mt-0.5">
                  {hit.url}
                </p>
                {hit.snippet && (
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                    {hit.snippet}
                  </p>
                )}

                {/* Expanded content */}
                {hit.content && (
                  <>
                    <button
                      onClick={() => setExpandedIndex(expandedIndex === index ? null : index)}
                      className="mt-2 text-xs text-amber-600 dark:text-amber-400 hover:underline focus:outline-none flex items-center gap-1"
                    >
                      <span>{expandedIndex === index ? '▼' : '▶'}</span>
                      {expandedIndex === index ? 'Hide content' : 'Show fetched content'}
                    </button>
                    {expandedIndex === index && (
                      <div className="mt-2 p-3 bg-white/50 dark:bg-black/20 rounded-lg border">
                        <pre className="whitespace-pre-wrap font-sans text-xs text-gray-700 dark:text-gray-300 max-h-60 overflow-auto">
                          {hit.content}
                        </pre>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {result.results.length === 0 && (
        <div className="p-8 text-center text-gray-500 dark:text-gray-400">
          <span className="text-3xl mb-2 block">🔍</span>
          No results found
        </div>
      )}
    </div>
  );
};

export default WebSearchVisualization;
