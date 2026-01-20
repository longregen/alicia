import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface WebLink {
  url: string;
  text?: string;
  internal: boolean;
}

interface WebLinksResult {
  url: string;
  total_found: number;
  links: WebLink[];
}

interface WebLinksVisualizationProps {
  result: WebLinksResult;
  className?: string;
}

const WebLinksVisualization: React.FC<WebLinksVisualizationProps> = ({ result, className }) => {
  const [filter, setFilter] = useState<'all' | 'internal' | 'external'>('all');
  const [showAll, setShowAll] = useState(false);

  const filteredLinks = result.links.filter((link) => {
    if (filter === 'internal') return link.internal;
    if (filter === 'external') return !link.internal;
    return true;
  });

  const displayedLinks = showAll ? filteredLinks : filteredLinks.slice(0, 10);
  const internalCount = result.links.filter((l) => l.internal).length;
  const externalCount = result.links.filter((l) => !l.internal).length;

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-violet-50 to-purple-50 dark:from-violet-950/30 dark:to-purple-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2 mb-2">
          <span className="text-2xl">🔗</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              Extracted Links
            </h3>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-violet-600 dark:text-violet-400 hover:underline truncate block"
            >
              {result.url}
            </a>
          </div>
          <span className="px-2 py-1 rounded-full text-xs font-bold bg-violet-200 text-violet-800 dark:bg-violet-800 dark:text-violet-200">
            {result.total_found}
          </span>
        </div>

        {/* Filter tabs */}
        <div className="flex gap-1 mt-2">
          <button
            onClick={() => setFilter('all')}
            className={cls(
              'px-3 py-1 rounded-full text-xs font-medium transition-colors',
              filter === 'all'
                ? 'bg-violet-600 text-white'
                : 'bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/50 dark:text-violet-300'
            )}
          >
            All ({result.total_found})
          </button>
          <button
            onClick={() => setFilter('internal')}
            className={cls(
              'px-3 py-1 rounded-full text-xs font-medium transition-colors',
              filter === 'internal'
                ? 'bg-green-600 text-white'
                : 'bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/50 dark:text-green-300'
            )}
          >
            Internal ({internalCount})
          </button>
          <button
            onClick={() => setFilter('external')}
            className={cls(
              'px-3 py-1 rounded-full text-xs font-medium transition-colors',
              filter === 'external'
                ? 'bg-blue-600 text-white'
                : 'bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/50 dark:text-blue-300'
            )}
          >
            External ({externalCount})
          </button>
        </div>
      </div>

      {/* Links list */}
      <div className="divide-y divide-gray-200 dark:divide-gray-700 max-h-96 overflow-auto">
        {displayedLinks.map((link, index) => (
          <div
            key={index}
            className="px-4 py-2 hover:bg-white/30 dark:hover:bg-black/10 transition-colors flex items-center gap-3"
          >
            <span
              className={cls(
                'flex-shrink-0 w-2 h-2 rounded-full',
                link.internal ? 'bg-green-500' : 'bg-blue-500'
              )}
            />
            <div className="flex-1 min-w-0">
              {link.text && (
                <p className="text-sm text-gray-700 dark:text-gray-300 truncate">
                  {link.text}
                </p>
              )}
              <a
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-blue-600 dark:text-blue-400 hover:underline truncate block"
              >
                {link.url}
              </a>
            </div>
            <span
              className={cls(
                'text-xs px-2 py-0.5 rounded-full',
                link.internal
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300'
                  : 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
              )}
            >
              {link.internal ? 'int' : 'ext'}
            </span>
          </div>
        ))}
      </div>

      {/* Show more/less */}
      {filteredLinks.length > 10 && (
        <div className="px-4 py-2 border-t bg-white/30 dark:bg-black/10">
          <button
            onClick={() => setShowAll(!showAll)}
            className="text-xs text-violet-600 dark:text-violet-400 hover:underline"
          >
            {showAll
              ? '← Show less'
              : `Show all ${filteredLinks.length} links`}
          </button>
        </div>
      )}
    </div>
  );
};

export default WebLinksVisualization;
