import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface WebScreenshotResult {
  url: string;
  width: number;
  height: number;
  format: string;
  data: string;
  size_bytes: number;
}

interface WebScreenshotVisualizationProps {
  result: WebScreenshotResult;
  className?: string;
}

const WebScreenshotVisualization: React.FC<WebScreenshotVisualizationProps> = ({ result, className }) => {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const imageSrc = `data:image/${result.format};base64,${result.data}`;

  const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <>
      <div className={cls('rounded-lg border bg-gradient-to-br from-pink-50 to-rose-50 dark:from-pink-950/30 dark:to-rose-950/30 overflow-hidden', className)}>
        {/* Header */}
        <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
          <div className="flex items-center gap-2">
            <span className="text-2xl">📸</span>
            <div className="flex-1">
              <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
                Screenshot
              </h3>
              <a
                href={result.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-pink-600 dark:text-pink-400 hover:underline truncate block"
              >
                {result.url}
              </a>
            </div>
          </div>

          {/* Metadata badges */}
          <div className="flex flex-wrap gap-2 mt-2">
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-pink-100 text-pink-800 dark:bg-pink-900/50 dark:text-pink-200">
              {result.width} × {result.height}
            </span>
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200">
              {result.format.toUpperCase()}
            </span>
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-200">
              {formatBytes(result.size_bytes)}
            </span>
          </div>
        </div>

        {/* Screenshot preview */}
        <div className="p-4">
          <div
            className="relative rounded-lg overflow-hidden border border-gray-200 dark:border-gray-700 shadow-lg cursor-pointer group"
            onClick={() => setIsFullscreen(true)}
          >
            <img
              src={imageSrc}
              alt={`Screenshot of ${result.url}`}
              className="w-full h-auto max-h-96 object-contain bg-gray-100 dark:bg-gray-900"
            />
            {/* Hover overlay */}
            <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center">
              <span className="opacity-0 group-hover:opacity-100 transition-opacity text-white text-sm font-medium bg-black/50 px-3 py-1.5 rounded-full">
                🔍 Click to enlarge
              </span>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="px-4 py-2 border-t bg-white/30 dark:bg-black/10 flex gap-2">
          <a
            href={imageSrc}
            download={`screenshot-${Date.now()}.${result.format}`}
            className="text-xs text-pink-600 dark:text-pink-400 hover:underline flex items-center gap-1"
          >
            ⬇️ Download
          </a>
          <button
            onClick={() => navigator.clipboard.writeText(result.data)}
            className="text-xs text-pink-600 dark:text-pink-400 hover:underline flex items-center gap-1"
          >
            📋 Copy base64
          </button>
        </div>
      </div>

      {/* Fullscreen modal */}
      {isFullscreen && (
        <div
          className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center p-4"
          onClick={() => setIsFullscreen(false)}
        >
          <button
            onClick={() => setIsFullscreen(false)}
            className="absolute top-4 right-4 text-white hover:text-gray-300 text-2xl"
          >
            ✕
          </button>
          <img
            src={imageSrc}
            alt={`Screenshot of ${result.url}`}
            className="max-w-full max-h-full object-contain"
          />
        </div>
      )}
    </>
  );
};

export default WebScreenshotVisualization;
