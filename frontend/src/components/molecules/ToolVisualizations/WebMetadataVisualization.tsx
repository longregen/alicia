import React from 'react';
import { cls } from '../../../utils/cls';

interface OpenGraphData {
  title?: string;
  description?: string;
  image?: string;
  url?: string;
  type?: string;
  site_name?: string;
}

interface TwitterData {
  card?: string;
  title?: string;
  description?: string;
  image?: string;
  site?: string;
}

interface WebMetadataResult {
  url: string;
  title?: string;
  description?: string;
  author?: string;
  keywords?: string[];
  canonical?: string;
  language?: string;
  favicon?: string;
  published_time?: string;
  modified_time?: string;
  robots?: string;
  open_graph?: OpenGraphData;
  twitter_card?: TwitterData;
  json_ld?: string[];
}

interface WebMetadataVisualizationProps {
  result: WebMetadataResult;
  className?: string;
}

const WebMetadataVisualization: React.FC<WebMetadataVisualizationProps> = ({ result, className }) => {
  const hasOpenGraph = result.open_graph && Object.values(result.open_graph).some(Boolean);
  const hasTwitter = result.twitter_card && Object.values(result.twitter_card).some(Boolean);

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-emerald-50 to-teal-50 dark:from-emerald-950/30 dark:to-teal-950/30 overflow-hidden', className)}>
      {/* Header with preview card */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-start gap-3">
          <span className="text-2xl">📋</span>
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              {result.title || 'Page Metadata'}
            </h3>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-emerald-600 dark:text-emerald-400 hover:underline truncate block"
            >
              {result.url}
            </a>
            {result.description && (
              <p className="text-xs text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                {result.description}
              </p>
            )}
          </div>
          {result.favicon && (
            <img
              src={result.favicon}
              alt="Favicon"
              className="w-8 h-8 rounded"
              onError={(e) => (e.currentTarget.style.display = 'none')}
            />
          )}
        </div>
      </div>

      {/* Basic metadata */}
      <div className="p-4 space-y-3">
        {/* Key info badges */}
        <div className="flex flex-wrap gap-2">
          {result.language && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200">
              🌐 {result.language}
            </span>
          )}
          {result.author && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-200">
              ✍️ {result.author}
            </span>
          )}
          {result.published_time && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-200">
              📅 {new Date(result.published_time).toLocaleDateString()}
            </span>
          )}
          {result.robots && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200">
              🤖 {result.robots}
            </span>
          )}
        </div>

        {/* Keywords */}
        {result.keywords && result.keywords.length > 0 && (
          <div>
            <span className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">
              Keywords
            </span>
            <div className="flex flex-wrap gap-1 mt-1">
              {result.keywords.slice(0, 10).map((keyword, i) => (
                <span
                  key={i}
                  className="px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                >
                  {keyword}
                </span>
              ))}
              {result.keywords.length > 10 && (
                <span className="text-xs text-gray-500">
                  +{result.keywords.length - 10} more
                </span>
              )}
            </div>
          </div>
        )}

        {/* Open Graph section */}
        {hasOpenGraph && (
          <div className="bg-white/50 dark:bg-black/20 rounded-lg p-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm">📱</span>
              <span className="text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide">
                Open Graph
              </span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              {result.open_graph?.type && (
                <div>
                  <span className="text-gray-500">Type:</span>{' '}
                  <span className="text-gray-700 dark:text-gray-300">{result.open_graph.type}</span>
                </div>
              )}
              {result.open_graph?.site_name && (
                <div>
                  <span className="text-gray-500">Site:</span>{' '}
                  <span className="text-gray-700 dark:text-gray-300">{result.open_graph.site_name}</span>
                </div>
              )}
            </div>
            {result.open_graph?.image && (
              <img
                src={result.open_graph.image}
                alt="OG Image"
                className="mt-2 rounded max-h-32 object-cover"
                onError={(e) => (e.currentTarget.style.display = 'none')}
              />
            )}
          </div>
        )}

        {/* Twitter Card section */}
        {hasTwitter && (
          <div className="bg-white/50 dark:bg-black/20 rounded-lg p-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm">🐦</span>
              <span className="text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide">
                Twitter Card
              </span>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              {result.twitter_card?.card && (
                <div>
                  <span className="text-gray-500">Card:</span>{' '}
                  <span className="text-gray-700 dark:text-gray-300">{result.twitter_card.card}</span>
                </div>
              )}
              {result.twitter_card?.site && (
                <div>
                  <span className="text-gray-500">Site:</span>{' '}
                  <span className="text-gray-700 dark:text-gray-300">{result.twitter_card.site}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Canonical */}
        {result.canonical && result.canonical !== result.url && (
          <div className="text-xs">
            <span className="text-gray-500">Canonical:</span>{' '}
            <a
              href={result.canonical}
              target="_blank"
              rel="noopener noreferrer"
              className="text-emerald-600 dark:text-emerald-400 hover:underline"
            >
              {result.canonical}
            </a>
          </div>
        )}

        {/* JSON-LD count */}
        {result.json_ld && result.json_ld.length > 0 && (
          <div className="text-xs text-gray-500">
            📦 {result.json_ld.length} JSON-LD structured data block(s) found
          </div>
        )}
      </div>
    </div>
  );
};

export default WebMetadataVisualization;
