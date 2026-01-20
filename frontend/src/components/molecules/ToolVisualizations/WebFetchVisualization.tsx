import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface WebFetchRawResult {
  url: string;
  status_code: number;
  status: string;
  headers: Record<string, string>;
  body: string;
  body_length: number;
}

interface WebFetchStructuredResult {
  url: string;
  data: Record<string, unknown>;
}

interface WebFetchVisualizationProps {
  result: WebFetchRawResult | WebFetchStructuredResult;
  type: 'raw' | 'structured';
  className?: string;
}

const WebFetchVisualization: React.FC<WebFetchVisualizationProps> = ({ result, type, className }) => {
  const [showHeaders, setShowHeaders] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);

  if (type === 'structured') {
    const structuredResult = result as WebFetchStructuredResult;
    return (
      <div className={cls('rounded-lg border bg-gradient-to-br from-cyan-50 to-teal-50 dark:from-cyan-950/30 dark:to-teal-950/30 overflow-hidden', className)}>
        <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🔍</span>
            <div className="flex-1">
              <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
                Structured Data
              </h3>
              <a
                href={structuredResult.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-cyan-600 dark:text-cyan-400 hover:underline truncate block"
              >
                {structuredResult.url}
              </a>
            </div>
          </div>
        </div>

        <div className="p-4">
          <div className="space-y-3">
            {Object.entries(structuredResult.data).map(([key, value]) => (
              <div key={key} className="bg-white/50 dark:bg-black/20 rounded-lg p-3">
                <span className="text-xs font-medium text-cyan-700 dark:text-cyan-300 uppercase tracking-wide">
                  {key}
                </span>
                <div className="mt-1 text-sm text-gray-700 dark:text-gray-300">
                  {Array.isArray(value) ? (
                    <ul className="list-disc list-inside space-y-1">
                      {value.map((item, i) => (
                        <li key={i} className="truncate">{String(item)}</li>
                      ))}
                    </ul>
                  ) : value === null ? (
                    <span className="text-gray-400 italic">null</span>
                  ) : (
                    <span className="break-words">{String(value)}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  const rawResult = result as WebFetchRawResult;
  const isSuccess = rawResult.status_code >= 200 && rawResult.status_code < 300;
  const isRedirect = rawResult.status_code >= 300 && rawResult.status_code < 400;

  const statusColor = isSuccess
    ? 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-200'
    : isRedirect
    ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-200'
    : 'bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-200';

  const bodyPreview = rawResult.body.slice(0, 500);
  const hasMore = rawResult.body.length > 500;

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-slate-50 to-zinc-50 dark:from-slate-950/30 dark:to-zinc-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">🌐</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              HTTP Response
            </h3>
            <a
              href={rawResult.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-blue-600 dark:text-blue-400 hover:underline truncate block"
            >
              {rawResult.url}
            </a>
          </div>
          <span className={cls('px-2 py-1 rounded-full text-xs font-bold', statusColor)}>
            {rawResult.status_code}
          </span>
        </div>
      </div>

      {/* Headers toggle */}
      <div className="px-4 py-2 border-b bg-white/30 dark:bg-black/10">
        <button
          onClick={() => setShowHeaders(!showHeaders)}
          className="text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 flex items-center gap-1"
        >
          <span>{showHeaders ? '▼' : '▶'}</span>
          Headers ({Object.keys(rawResult.headers).length})
        </button>
        {showHeaders && (
          <div className="mt-2 space-y-1">
            {Object.entries(rawResult.headers).map(([key, value]) => (
              <div key={key} className="flex gap-2 text-xs">
                <span className="font-medium text-gray-700 dark:text-gray-300">{key}:</span>
                <span className="text-gray-600 dark:text-gray-400 truncate">{value}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Body */}
      <div className="p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
            Response Body
          </span>
          <span className="text-xs text-gray-400 dark:text-gray-500">
            {rawResult.body_length.toLocaleString()} bytes
          </span>
        </div>
        <pre className="bg-white/50 dark:bg-black/20 rounded-lg p-3 overflow-auto max-h-80 text-xs font-mono text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
          {isExpanded ? rawResult.body : bodyPreview}
          {!isExpanded && hasMore && '...'}
        </pre>
        {hasMore && (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="mt-2 text-xs text-blue-600 dark:text-blue-400 hover:underline"
          >
            {isExpanded ? '← Show less' : 'Show full response'}
          </button>
        )}
      </div>
    </div>
  );
};

export default WebFetchVisualization;
