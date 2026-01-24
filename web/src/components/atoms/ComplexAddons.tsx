import React, { useState, useEffect } from 'react';
import AudioAddon from './AudioAddon';
import FeedbackControls from './FeedbackControls';
import BranchNavigator from './BranchNavigator';
import { HoverPopover } from './HoverPopover';
import { cls } from '../../utils/cls';
import { useFeedback } from '../../hooks/useFeedback';
import type { BaseComponentProps, MessageAddon, AudioState, MemoryAddonData } from '../../types/components';

const AUDIO_STATES = { IDLE: 'idle', PLAYING: 'playing', PAUSED: 'paused' } as const;

const TOOL_EMOJIS: Record<string, string> = {
  calculate: '🧮',
  describe_table: '📊',
  execute_sql: '🗃️',
  schema_explore: '🔍',
  read: '📖',
  fetch_raw: '📥',
  fetch_structured: '📋',
  search: '🔎',
  extract_links: '🔗',
  extract_metadata: '🏷️',
  screenshot: '📸',
  memory_search: '🧠',
  memory_query: '🧠',
  memory: '🧠',
  web_search: '🔎',
};

const DISPLAY_LIMITS = {
  RESULTS: 6,
  SQL_ROWS: 10,
  LINKS: 15,
  CONTENT_CHARS: 80,
} as const;

const getHostname = (url: string): string | null => {
  try { return new URL(url).hostname; } catch { return null; }
};

const truncateText = (text: string | undefined, maxLen = DISPLAY_LIMITS.CONTENT_CHARS): string =>
  text ? (text.length > maxLen ? text.substring(0, maxLen) + '...' : text) : '';

const parseToolArgs = (args: string): Record<string, unknown> => {
  try {
    return JSON.parse(args.replace(/^Arguments:\s*/i, ''));
  } catch { return {}; }
};

export function getToolEmoji(toolName: string): string {
  const lowerName = toolName.toLowerCase();
  if (TOOL_EMOJIS[lowerName]) return TOOL_EMOJIS[lowerName];
  for (const [key, emoji] of Object.entries(TOOL_EMOJIS)) {
    if (lowerName.includes(key)) return emoji;
  }
  return '⚡';
}

export interface ToolDetail {
  id: string;
  name: string;
  description: string;
  result?: string;
  status?: 'pending' | 'running' | 'completed' | 'error';
}

const formatWebSearchCompact = (toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const query = (parsedArgs.query as string) || '';
  const limit = (parsedArgs.limit as number) || 5;
  let results: Array<{ title?: string; url?: string }> = [];

  try {
    const parsedResult = JSON.parse(result);
    results = parsedResult.results || [];
  } catch { /* ignore */ }

  return (
    <div className="space-y-2">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">{toolName}:</span> <span className="text-accent">"{query}"</span> <span className="text-muted-foreground/60">(limit {limit})</span>
      </div>
      <div className="text-xs">
        <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({results.length})</span>
      </div>
      {results.length > 0 && (
        <div className="space-y-1.5 pl-1">
          {results.slice(0, DISPLAY_LIMITS.RESULTS).map((item, index) => (
            <div key={index}>
              <div className="text-xs font-medium text-default truncate">
                • {item.title || 'Untitled'}
              </div>
              {item.url && (
                <a
                  href={item.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block text-[10px] text-accent/70 truncate pl-2.5 hover:text-accent hover:underline"
                  onClick={(e) => {
                    e.stopPropagation();
                  }}
                >
                  {item.url}
                </a>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const formatWebSearchResult = (parsed: Record<string, unknown>): React.ReactNode => {
  const results = parsed.results as Array<{ title?: string; url?: string; snippet?: string }> | undefined;
  const resultCount = parsed.result_count as number | undefined;

  return (
    <div className="space-y-1.5">
      {results && results.length > 0 && (
        <>
          <div className="text-xs text-success flex items-center gap-1.5 mb-2">
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            Results ({resultCount || results.length}):
          </div>
          <div className="space-y-1.5 pl-1">
            {results.slice(0, 6).map((item, index) => (
              <div key={index}>
                <div className="text-xs font-medium text-default truncate">
                  • {item.title || 'Untitled'}
                </div>
                {item.url && (
                  <div className="text-[10px] text-accent/70 truncate pl-2.5">
                    {item.url}
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

const formatMemoryCompact = (toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const query = (parsedArgs.query as string) || (parsedArgs.text as string) || '';
  const limit = (parsedArgs.limit as number) || (parsedArgs.k as number) || 5;

  let memories: Array<{ id?: string; content?: string; similarity?: number; relevance?: number }> = [];
  try {
    const parsedResult = JSON.parse(result);
    memories = parsedResult.memories || parsedResult || [];
  } catch { /* ignore */ }

  return (
    <div className="space-y-2">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">{toolName}:</span> <span className="text-accent">"{query}"</span> <span className="text-muted-foreground/60">(limit {limit})</span>
      </div>
      <div className="text-xs">
        <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({memories.length})</span>
      </div>
      {memories.length > 0 && (
        <div className="space-y-1.5 pl-1">
          {memories.slice(0, DISPLAY_LIMITS.RESULTS).map((item, index) => (
            <div key={item.id || index}>
              {item.id ? (
                <a
                  href={`/memory/${item.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block text-xs text-default hover:text-accent hover:underline"
                  onClick={(e) => e.stopPropagation()}
                >
                  • {truncateText(item.content) || 'No content'}
                </a>
              ) : (
                <div className="text-xs text-default">
                  • {truncateText(item.content) || 'No content'}
                </div>
              )}
              {(item.similarity !== undefined || item.relevance !== undefined) && (
                <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                  {Math.round((item.similarity ?? item.relevance ?? 0) * 100)}% match
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const formatMemoryQueryResult = (parsed: Record<string, unknown>): React.ReactNode => {
  const memories = parsed.memories as Array<{ id?: string; content?: string; similarity?: number }> | undefined;
  const count = parsed.count as number | undefined;

  return (
    <div className="space-y-1.5">
      {memories && memories.length > 0 && (
        <>
          <div className="text-xs">
            <span className="text-muted-foreground">Results:</span> <span className="text-muted-foreground/60">({count || memories.length})</span>
          </div>
          <div className="space-y-1.5 pl-1">
            {memories.slice(0, 6).map((item, index) => (
              <div key={item.id || index}>
                {item.id ? (
                  <a
                    href={`/memory/${item.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block text-xs text-default hover:text-accent hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    • {truncateText(item.content) || 'No content'}
                  </a>
                ) : (
                  <div className="text-xs text-default">
                    • {truncateText(item.content) || 'No content'}
                  </div>
                )}
                {item.similarity !== undefined && (
                  <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                    {Math.round(item.similarity * 100)}% match
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
};

const formatCalculateCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const code = (parsedArgs.code as string) || '';

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">🧮 calculate</span>
      </div>
      {code && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Code:</div>
          <pre className="text-xs bg-black/30 rounded p-2 overflow-x-auto max-h-32 font-mono text-accent">
            {code.length > 300 ? code.substring(0, 300) + '...' : code}
          </pre>
        </div>
      )}
      {result && (
        <div>
          <div className="text-xs text-success mb-1">Result:</div>
          <pre className="text-xs bg-success/10 rounded p-2 overflow-x-auto max-h-24 font-mono text-default">
            {result.length > 500 ? result.substring(0, 500) + '...' : result}
          </pre>
        </div>
      )}
    </div>
  );
};

interface TableInfo {
  columns?: Array<{ name: string; type: string; nullable: boolean }>;
  row_count?: number;
  primary_key?: string[];
  foreign_keys?: Array<{ column: string; references: string }>;
}

const formatDescribeTableCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const tableName = (parsedArgs.table as string) || '';

  let tableInfo: TableInfo = {};
  try { tableInfo = JSON.parse(result); } catch { /* ignore */ }

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">📊 describe_table:</span>{' '}
        <span className="text-accent font-semibold">{tableName}</span>
        {tableInfo.row_count !== undefined && (
          <span className="text-muted-foreground/60 ml-2">({tableInfo.row_count.toLocaleString()} rows)</span>
        )}
      </div>

      {tableInfo.columns && tableInfo.columns.length > 0 && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Columns ({tableInfo.columns.length}):</div>
          <div className="space-y-0.5 max-h-40 overflow-y-auto">
            {tableInfo.columns.map((col, i) => (
              <div key={i} className="text-xs flex items-center gap-2">
                <span className="text-default font-mono">{col.name}</span>
                <span className="text-muted-foreground/60">{col.type}</span>
                {!col.nullable && <span className="text-warning text-[10px]">NOT NULL</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {tableInfo.primary_key && tableInfo.primary_key.length > 0 && (
        <div className="text-xs">
          <span className="text-muted-foreground">Primary Key:</span>{' '}
          <span className="text-accent font-mono">{tableInfo.primary_key.join(', ')}</span>
        </div>
      )}

      {tableInfo.foreign_keys && tableInfo.foreign_keys.length > 0 && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Foreign Keys:</div>
          <div className="space-y-0.5">
            {tableInfo.foreign_keys.map((fk, i) => (
              <div key={i} className="text-xs">
                <span className="font-mono text-default">{fk.column}</span>
                <span className="text-muted-foreground/60"> → </span>
                <span className="font-mono text-accent">{fk.references}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

interface SQLResult {
  success?: boolean;
  columns?: string[];
  rows?: Record<string, unknown>[];
  row_count?: number;
  error?: string;
  hint?: string;
  truncated?: boolean;
}

const formatExecuteSQLCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const sql = (parsedArgs.sql as string) || '';

  let sqlResult: SQLResult = {};
  try { sqlResult = JSON.parse(result); } catch { /* ignore */ }

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">🗃️ execute_sql</span>
      </div>

      {sql && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Query:</div>
          <pre className="text-xs bg-black/30 rounded p-2 overflow-x-auto max-h-20 font-mono text-accent">
            {sql.length > 200 ? sql.substring(0, 200) + '...' : sql}
          </pre>
        </div>
      )}

      {sqlResult.success === false && sqlResult.error && (
        <div>
          <div className="text-xs text-error mb-1">Error:</div>
          <div className="text-xs text-error/80 bg-error/10 rounded p-2">{sqlResult.error}</div>
          {sqlResult.hint && (
            <div className="text-xs text-warning mt-2 bg-warning/10 rounded p-2">
              💡 {sqlResult.hint}
            </div>
          )}
        </div>
      )}

      {sqlResult.success && sqlResult.columns && (
        <div>
          <div className="text-xs text-success mb-1 flex items-center gap-2">
            <span>✓ {sqlResult.row_count} row{sqlResult.row_count !== 1 ? 's' : ''}</span>
            {sqlResult.truncated && <span className="text-warning">(truncated)</span>}
          </div>

          {sqlResult.rows && sqlResult.rows.length > 0 && (
            <div className="overflow-x-auto max-h-48">
              <table className="text-xs w-full">
                <thead>
                  <tr className="border-b border-border">
                    {sqlResult.columns.map((col, i) => (
                      <th key={i} className="text-left p-1 text-muted-foreground font-mono">{col}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {sqlResult.rows.slice(0, 10).map((row, rowIdx) => (
                    <tr key={rowIdx} className="border-b border-border/50">
                      {sqlResult.columns!.map((col, colIdx) => (
                        <td key={colIdx} className="p-1 text-default font-mono truncate max-w-[150px]">
                          {row[col] === null ? <span className="text-muted-foreground/50 italic">null</span> : String(row[col])}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
              {sqlResult.rows.length > 10 && (
                <div className="text-xs text-muted-foreground/60 mt-1">...and {sqlResult.rows.length - 10} more rows</div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const formatSchemaExploreCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const question = (parsedArgs.question as string) || '';

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">🔍 schema_explore</span>
      </div>

      {question && (
        <div className="text-xs">
          <span className="text-muted-foreground">Question:</span>{' '}
          <span className="text-accent italic">"{question}"</span>
        </div>
      )}

      {result && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Answer:</div>
          <div className="text-xs text-default bg-black/20 rounded p-2 max-h-48 overflow-y-auto whitespace-pre-wrap">
            {result.length > 1000 ? result.substring(0, 1000) + '...' : result}
          </div>
        </div>
      )}
    </div>
  );
};

interface ReadResult {
  url?: string;
  title?: string;
  content?: string;
  word_count?: number;
  estimated_tokens?: number;
  excerpt?: string;
  author?: string;
  site_name?: string;
  js_rendered?: boolean;
}

const formatReadCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const url = (parsedArgs.url as string) || '';

  let readResult: ReadResult = {};
  try { readResult = JSON.parse(result); } catch { /* ignore */ }

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">📖 read</span>
      </div>

      {(url || readResult.url) && (
        <a
          href={readResult.url || url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-accent hover:underline block truncate"
          onClick={(e) => e.stopPropagation()}
        >
          {readResult.url || url}
        </a>
      )}

      {readResult.title && (
        <div className="text-sm font-medium text-default">{readResult.title}</div>
      )}

      <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground/60">
        {readResult.site_name && <span>🌐 {readResult.site_name}</span>}
        {readResult.author && <span>✍️ {readResult.author}</span>}
        {readResult.word_count && <span>📝 {readResult.word_count.toLocaleString()} words</span>}
        {readResult.estimated_tokens && <span>🔢 ~{readResult.estimated_tokens.toLocaleString()} tokens</span>}
        {readResult.js_rendered && <span>⚡ JS rendered</span>}
      </div>

      {readResult.excerpt && (
        <div className="text-xs text-muted-foreground italic border-l-2 border-accent/30 pl-2">
          {readResult.excerpt.length > 200 ? readResult.excerpt.substring(0, 200) + '...' : readResult.excerpt}
        </div>
      )}

      {readResult.content && (
        <details className="text-xs">
          <summary className="text-muted-foreground cursor-pointer hover:text-accent">Show content preview</summary>
          <div className="mt-2 text-default bg-black/20 rounded p-2 max-h-32 overflow-y-auto whitespace-pre-wrap">
            {readResult.content.length > 500 ? readResult.content.substring(0, 500) + '...' : readResult.content}
          </div>
        </details>
      )}
    </div>
  );
};

const formatExtractLinksCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const url = (parsedArgs.url as string) || '';
  const hostname = getHostname(url);

  let links: Array<{ url?: string; text?: string; href?: string }> = [];
  try {
    const parsed = JSON.parse(result);
    links = parsed.links || parsed || [];
  } catch { /* ignore */ }

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">🔗 extract_links</span>
        {hostname && <span className="text-muted-foreground/60 ml-2">from {hostname}</span>}
      </div>

      <div className="text-xs text-muted-foreground">
        Found {links.length} link{links.length !== 1 ? 's' : ''}
      </div>

      {links.length > 0 && (
        <div className="space-y-1 max-h-48 overflow-y-auto">
          {links.slice(0, DISPLAY_LIMITS.LINKS).map((link, i) => (
            <div key={i} className="text-xs">
              <a
                href={link.url || link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent hover:underline"
                onClick={(e) => e.stopPropagation()}
              >
                {link.text || link.url || link.href || 'Link'}
              </a>
            </div>
          ))}
          {links.length > DISPLAY_LIMITS.LINKS && (
            <div className="text-xs text-muted-foreground/60">...and {links.length - DISPLAY_LIMITS.LINKS} more links</div>
          )}
        </div>
      )}
    </div>
  );
};

const formatExtractMetadataCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const url = (parsedArgs.url as string) || '';
  const hostname = getHostname(url);

  let metadata: Record<string, unknown> = {};
  try { metadata = JSON.parse(result); } catch { /* ignore */ }

  const renderValue = (value: unknown): string => {
    if (typeof value === 'string') return value;
    if (typeof value === 'number' || typeof value === 'boolean') return String(value);
    if (Array.isArray(value)) return value.join(', ');
    return JSON.stringify(value);
  };

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">🏷️ extract_metadata</span>
        {hostname && <span className="text-muted-foreground/60 ml-2">from {hostname}</span>}
      </div>

      {Object.keys(metadata).length > 0 && (
        <div className="space-y-1 max-h-48 overflow-y-auto">
          {Object.entries(metadata).slice(0, 20).map(([key, value]) => (
            <div key={key} className="text-xs flex gap-2">
              <span className="text-muted-foreground font-mono min-w-[80px]">{key}:</span>
              <span className="text-default truncate">{renderValue(value)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

interface ScreenshotResult {
  url?: string;
  width?: number;
  height?: number;
  format?: string;
  size?: number;
  base64?: string;
}

const formatScreenshotCompact = (_toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const url = (parsedArgs.url as string) || '';

  let screenshotResult: ScreenshotResult = {};
  try { screenshotResult = JSON.parse(result); } catch { /* ignore */ }

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">📸 screenshot</span>
      </div>

      {(url || screenshotResult.url) && (
        <a
          href={screenshotResult.url || url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-accent hover:underline block truncate"
          onClick={(e) => e.stopPropagation()}
        >
          {screenshotResult.url || url}
        </a>
      )}

      <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
        {screenshotResult.width && screenshotResult.height && (
          <span>📐 {screenshotResult.width}×{screenshotResult.height}</span>
        )}
        {screenshotResult.format && <span>🖼️ {screenshotResult.format.toUpperCase()}</span>}
        {screenshotResult.size && <span>📦 {(screenshotResult.size / 1024).toFixed(1)}KB</span>}
      </div>

      {screenshotResult.base64 && (
        <div className="mt-2">
          <img
            src={`data:image/${screenshotResult.format || 'png'};base64,${screenshotResult.base64}`}
            alt="Screenshot"
            className="max-w-full max-h-48 rounded border border-border"
          />
        </div>
      )}
    </div>
  );
};

const formatFetchCompact = (toolName: string, args: string, result: string): React.ReactNode => {
  const parsedArgs = parseToolArgs(args);
  const url = (parsedArgs.url as string) || '';
  const isStructured = toolName.toLowerCase().includes('structured');
  const emoji = isStructured ? '📋' : '📥';

  return (
    <div className="space-y-3">
      <div className="text-xs">
        <span className="text-muted-foreground font-mono">{emoji} {toolName}</span>
      </div>

      {url && (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-accent hover:underline block truncate"
          onClick={(e) => e.stopPropagation()}
        >
          {url}
        </a>
      )}

      {result && (
        <div>
          <div className="text-xs text-muted-foreground mb-1">Response ({result.length.toLocaleString()} chars):</div>
          <pre className="text-xs bg-black/20 rounded p-2 max-h-32 overflow-y-auto font-mono text-default whitespace-pre-wrap">
            {result.length > 500 ? result.substring(0, 500) + '...' : result}
          </pre>
        </div>
      )}
    </div>
  );
};

const formatToolArguments = (description: string): React.ReactNode => {
  try {
    const parsed = JSON.parse(description);

    if (typeof parsed === 'object' && parsed !== null) {
      const entries = Object.entries(parsed);
      if (entries.length === 0) {
        return <span className="text-muted text-xs italic">No arguments</span>;
      }

      return (
        <div className="space-y-1">
          {entries.map(([key, value]) => (
            <div key={key} className="flex items-start gap-2 text-xs">
              <span className="text-muted font-medium min-w-[60px]">{key}:</span>
              <span className="text-default">
                {typeof value === 'string' ? `"${value}"` : JSON.stringify(value)}
              </span>
            </div>
          ))}
        </div>
      );
    }

    return <span className="text-default text-xs">{description}</span>;
  } catch {
    // Not JSON, return as plain text
    return <span className="text-default text-xs">{description}</span>;
  }
};

const formatToolResult = (result: string, toolName: string): React.ReactNode => {
  try {
    const parsed = JSON.parse(result);
    const lowerToolName = toolName.toLowerCase();

    // Handle web search results
    if (lowerToolName.includes('web_search') || lowerToolName.includes('search')) {
      if (parsed.results && Array.isArray(parsed.results)) {
        return formatWebSearchResult(parsed);
      }
    }

    // Handle memory query results
    if (lowerToolName.includes('memory')) {
      if (parsed.memories && Array.isArray(parsed.memories)) {
        return formatMemoryQueryResult(parsed);
      }
      // Handle array format for memory results
      if (Array.isArray(parsed)) {
        return (
          <div className="space-y-1.5 pl-1">
            {parsed.slice(0, 6).map((item: { id?: string; content?: string; relevance?: number; similarity?: number }, index: number) => (
              <div key={item.id || index}>
                {item.id ? (
                  <a
                    href={`/memory/${item.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block text-xs text-default hover:text-accent hover:underline"
                    onClick={(e) => e.stopPropagation()}
                  >
                    • {truncateText(item.content) || 'No content'}
                  </a>
                ) : (
                  <div className="text-xs text-default">
                    • {truncateText(item.content) || 'No content'}
                  </div>
                )}
                {(item.relevance !== undefined || item.similarity !== undefined) && (
                  <div className="text-[10px] text-muted-foreground/60 pl-2.5">
                    {Math.round((item.relevance ?? item.similarity ?? 0) * 100)}% match
                  </div>
                )}
              </div>
            ))}
          </div>
        );
      }
    }

    // Handle simple success/error results
    if (typeof parsed === 'object' && parsed !== null) {
      if ('success' in parsed || 'error' in parsed || 'message' in parsed) {
        return (
          <div className="space-y-1">
            {parsed.success !== undefined && (
              <div className={cls(
                'flex items-center gap-1.5 text-xs',
                parsed.success ? 'text-success' : 'text-error'
              )}>
                <span>{parsed.success ? '✓' : '✗'}</span>
                <span>{parsed.success ? 'Success' : 'Failed'}</span>
              </div>
            )}
            {parsed.message && (
              <div className="text-xs text-default">{String(parsed.message)}</div>
            )}
            {parsed.error && (
              <div className="text-xs text-error">{String(parsed.error)}</div>
            )}
          </div>
        );
      }
    }

    // Generic JSON formatting with truncation for readability
    const jsonStr = JSON.stringify(parsed, null, 2);
    const truncated = jsonStr.length > 500;
    return (
      <pre className="text-xs text-default whitespace-pre-wrap font-mono bg-surface-bg p-2 rounded max-h-48 overflow-y-auto">
        {truncated ? jsonStr.substring(0, 500) + '\n...' : jsonStr}
      </pre>
    );
  } catch {
    // Not JSON, return as plain text
    return <span className="text-default text-xs">{result}</span>;
  }
};

// Popover content component for tool details
interface ToolPopoverContentProps {
  toolDetail: ToolDetail;
  showFeedback?: boolean;
}

const getToolFormatter = (toolName: string): ((name: string, args: string, result: string) => React.ReactNode) | null => {
  // Strip MCP server prefix (e.g. "garden:execute_sql" -> "execute_sql")
  const baseName = toolName.includes(':') ? toolName.split(':').pop()! : toolName;
  const lowerName = baseName.toLowerCase();

  if (lowerName === 'calculate') return formatCalculateCompact;
  if (lowerName === 'describe_table') return formatDescribeTableCompact;
  if (lowerName === 'execute_sql') return formatExecuteSQLCompact;
  if (lowerName === 'schema_explore') return formatSchemaExploreCompact;
  if (lowerName === 'read') return formatReadCompact;
  if (lowerName === 'search' || lowerName.includes('web_search')) return formatWebSearchCompact;
  if (lowerName === 'screenshot') return formatScreenshotCompact;
  if (lowerName === 'extract_links') return formatExtractLinksCompact;
  if (lowerName === 'extract_metadata') return formatExtractMetadataCompact;
  if (lowerName === 'fetch_raw' || lowerName === 'fetch_structured') return formatFetchCompact;
  if (lowerName.includes('memory')) return formatMemoryCompact;

  return null;
};

const ToolPopoverContent: React.FC<ToolPopoverContentProps> = ({ toolDetail, showFeedback = false }) => {
  const {
    currentVote,
    vote,
    counts,
    isLoading: feedbackLoading,
  } = useFeedback('tool_use', toolDetail.id);

  const isCompleted = toolDetail.status === 'completed';
  const hasResult = toolDetail.result && toolDetail.description;

  const formatter = getToolFormatter(toolDetail.name);

  if (formatter && isCompleted && hasResult) {
    return (
      <div className="space-y-3">
        <div className="max-h-64 overflow-y-auto pr-1 custom-scrollbar">
          {formatter(toolDetail.name, toolDetail.description, toolDetail.result!)}
        </div>
        {showFeedback && (
          <div className="pt-3 border-t border-border-muted">
            <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
            <FeedbackControls
              currentVote={currentVote as 'up' | 'down' | null}
              onVote={vote}
              upvotes={counts.up}
              downvotes={counts.down}
              isLoading={feedbackLoading}
              compact
            />
          </div>
        )}
      </div>
    );
  }

  // Generic tool content
  return (
    <div className="space-y-3">
      {/* Header with tool name and emoji */}
      <div>
        <span className="text-sm">{getToolEmoji(toolDetail.name)}</span>{' '}
        <span className="text-sm font-medium text-muted-foreground font-mono">{toolDetail.name}</span>
      </div>

      {/* Running indicator */}
      {toolDetail.status === 'running' && (
        <div className="text-xs text-accent flex items-center gap-1.5">
          <div className="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin" />
          Processing...
        </div>
      )}

      {/* Arguments section */}
      {toolDetail.description && (
        <div>
          <div className="flex items-center gap-1.5 text-xs text-muted font-medium mb-2">
            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
            </svg>
            Arguments
          </div>
          <div className="rounded-md px-3 py-2 bg-black/20 text-muted-foreground">
            {formatToolArguments(toolDetail.description)}
          </div>
        </div>
      )}

      {/* Results section */}
      {toolDetail.result && (toolDetail.status === 'completed' || toolDetail.status === 'error') && (
        <div>
          <div className={cls(
            'flex items-center gap-1.5 text-xs font-medium mb-2',
            toolDetail.status === 'error' ? 'text-error' : 'text-success'
          )}>
            {toolDetail.status === 'error' ? (
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            ) : (
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            )}
            {toolDetail.status === 'error' ? 'Error' : 'Result'}
          </div>
          <div className="max-h-48 overflow-y-auto pr-1 custom-scrollbar">
            {formatToolResult(toolDetail.result, toolDetail.name)}
          </div>
        </div>
      )}

      {/* Pending state */}
      {toolDetail.status === 'pending' && (
        <div className="text-xs text-warning flex items-center gap-1.5">
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          Waiting to execute...
        </div>
      )}

      {/* Feedback controls for completed tools */}
      {showFeedback && toolDetail.status === 'completed' && (
        <div className="pt-3 border-t border-border-muted">
          <div className="text-xs text-muted mb-2">Was this tool use helpful?</div>
          <FeedbackControls
            currentVote={currentVote as 'up' | 'down' | null}
            onVote={vote}
            upvotes={counts.up}
            downvotes={counts.down}
            isLoading={feedbackLoading}
            compact
          />
        </div>
      )}
    </div>
  );
};

// Popover content component for memory
interface MemoryPopoverContentProps {
  memory: MemoryAddonData;
  percentage: number;
  relevanceBgColor: string;
  relevanceColor: string;
}

const MemoryPopoverContent: React.FC<MemoryPopoverContentProps> = ({
  memory,
  percentage,
  relevanceBgColor,
  relevanceColor,
}) => {
  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">🧠</span>
          <span className="text-sm font-semibold text-accent">Memory Trace</span>
        </div>
        <div
          className={cls(
            'text-xs px-2 py-1 rounded-full font-semibold',
            relevanceBgColor,
            relevanceColor
          )}
        >
          {percentage}% relevant
        </div>
      </div>

      {/* Content */}
      <div className="text-sm text-default leading-relaxed max-h-48 overflow-y-auto pr-2 custom-scrollbar">
        {memory.content}
      </div>

      {/* Footer */}
      <div className="text-[11px] text-muted pt-3 border-t border-border-muted">
        Memory ID: {memory.id}
      </div>
    </div>
  );
};

// Branch navigation data interface
export interface BranchData {
  currentIndex: number;
  totalBranches: number;
  onNavigate: (direction: 'prev' | 'next') => void;
}

export interface ComplexAddonsProps extends BaseComponentProps {
  addons: MessageAddon[];
  toolDetails?: ToolDetail[];
  timestamp: Date;
  showFeedback?: boolean;
  branchData?: BranchData;
}

const ComplexAddons: React.FC<ComplexAddonsProps> = ({
  addons,
  toolDetails = [],
  timestamp,
  className = '',
  showFeedback = false,
  branchData,
}) => {
  const [audioState, setAudioState] = useState<AudioState>(AUDIO_STATES.IDLE);
  const [audioCurrentTime, setAudioCurrentTime] = useState(0);

  // Mock audio duration - in a real app this would come from the audio file
  const audioDuration = 45; // seconds

  // Simple audio simulation for demo
  useEffect(() => {
    if (audioState === AUDIO_STATES.PLAYING) {
      const interval = setInterval(() => {
        setAudioCurrentTime(prev => {
          if (prev >= audioDuration) {
            setAudioState(AUDIO_STATES.IDLE);
            return 0;
          }
          return prev + 0.1;
        });
      }, 100);
      return () => clearInterval(interval);
    }
  }, [audioState, audioDuration]);

  const getToolDetail = (addonId: string): ToolDetail | undefined => {
    return toolDetails.find(tool => tool.id === addonId);
  };

  const getAddonAnimation = (_addon: MessageAddon, toolDetail?: ToolDetail) => {
    if (toolDetail?.status === 'running') {
      return 'animate-pulse scale-110';
    }
    if (toolDetail?.status === 'pending') {
      return 'animate-pulse opacity-70';
    }
    if (toolDetail?.status === 'error') {
      return 'text-error';
    }
    return '';
  };

  // Helper functions for memory badges
  const getRelevancePercentage = (relevance: number): number => Math.round(relevance * 100);

  const getRelevanceColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'text-success';
    if (relevance >= 0.4) return 'text-accent';
    return 'text-muted';
  };

  const getRelevanceBgColor = (relevance: number): string => {
    if (relevance >= 0.7) return 'bg-success/10';
    if (relevance >= 0.4) return 'bg-accent-subtle';
    return 'bg-surface';
  };

  const renderMemoryBadge = (memory: MemoryAddonData) => {
    const percentage = getRelevancePercentage(memory.relevance);
    const relevanceColor = getRelevanceColor(memory.relevance);
    const relevanceBgColor = getRelevanceBgColor(memory.relevance);

    return (
      <HoverPopover
        key={memory.id}
        content={
          <MemoryPopoverContent
            memory={memory}
            percentage={percentage}
            relevanceBgColor={relevanceBgColor}
            relevanceColor={relevanceColor}
          />
        }
        side="top"
        align="start"
        sideOffset={8}
        alignOffset={8}
        openDelay={150}
        closeDelay={300}
      >
        <button
          className={cls(
            'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
            'transition-all duration-200 cursor-pointer',
            'hover:scale-105 hover:ring-2 hover:ring-accent/50',
            relevanceBgColor,
            relevanceColor
          )}
          title={`Memory trace (${percentage}% relevant)`}
        >
          <span className="text-sm">🧠</span>
          <span className="font-semibold">{percentage}%</span>
        </button>
      </HoverPopover>
    );
  };

  const renderAddon = (addon: MessageAddon) => {
    // Feedback addon - render FeedbackControls inline
    if (addon.type === 'feedback' && addon.feedbackData) {
      return (
        <FeedbackControls
          key={addon.id}
          currentVote={addon.feedbackData.currentVote}
          onVote={addon.feedbackData.onVote}
          upvotes={addon.feedbackData.upvotes}
          downvotes={addon.feedbackData.downvotes}
          isLoading={addon.feedbackData.isLoading}
          compact
        />
      );
    }

    // Memory addon - render memory badges with hover popovers
    if (addon.type === 'memory' && addon.memoryData) {
      const sortedMemories = [...addon.memoryData].sort((a, b) => b.relevance - a.relevance);
      return (
        <div key={addon.id} className="flex items-center gap-1.5">
          {sortedMemories.map(renderMemoryBadge)}
        </div>
      );
    }

    // Audio addon
    if (addon.type === 'audio') {
      return (
        <AudioAddon
          key={addon.id}
          mode="compact"
          state={audioState}
          onPlay={() => setAudioState(AUDIO_STATES.PLAYING)}
          onPause={() => setAudioState(AUDIO_STATES.PAUSED)}
          onStop={() => {
            setAudioState(AUDIO_STATES.IDLE);
            setAudioCurrentTime(0);
          }}
          duration={audioDuration}
          currentTime={audioCurrentTime}
        />
      );
    }

    // Default rendering for tool/icon addon types - with hover popover
    const toolDetail = getToolDetail(addon.id);

    // Render tool badge with hover popover
    return (
      <HoverPopover
        key={addon.id}
        content={
          toolDetail ? (
            <ToolPopoverContent toolDetail={toolDetail} showFeedback={showFeedback} />
          ) : (
            <div className="text-sm text-muted">No details available</div>
          )
        }
        side="top"
        align="start"
        sideOffset={8}
        alignOffset={8}
        openDelay={150}
        closeDelay={300}
        width="w-96"
      >
        <button
          className={cls(
            'inline-flex items-center justify-center w-7 h-7 rounded-md',
            'text-sm cursor-pointer transition-all duration-200',
            'hover:bg-surface-hover hover:ring-2 hover:ring-accent/50 border border-transparent',
            toolDetail?.status === 'running' ? 'bg-accent/10 border-accent/30' : 'bg-surface',
            toolDetail?.status === 'completed' ? 'hover:border-success/30' : '',
            toolDetail?.status === 'error' ? 'bg-error/10 border-error/30' : '',
            getAddonAnimation(addon, toolDetail)
          )}
          title={`${toolDetail?.name || addon.tooltip} - Hover for details`}
        >
          {toolDetail?.name ? getToolEmoji(toolDetail.name) : addon.emoji}
        </button>
      </HoverPopover>
    );
  };

  return (
    <div className={cls('space-y-2 w-full', className)}>
      {/* Main addon row */}
      <div className="flex items-center justify-between w-full gap-3">
        {/* Left: All addons inline */}
        <div className={cls('flex items-center gap-2', 'flex-wrap')}>
          {addons.map(renderAddon)}
        </div>

        {/* Right: Branch navigation + Timestamp */}
        <div className="flex items-center gap-3 flex-shrink-0">
          {/* Branch Navigator */}
          {branchData && branchData.totalBranches > 1 && (
            <BranchNavigator
              currentIndex={branchData.currentIndex}
              totalBranches={branchData.totalBranches}
              onNavigate={branchData.onNavigate}
            />
          )}

          {/* Timestamp with clock icon */}
          <div className={cls(
            'flex items-center gap-1.5',
            'text-[11px] text-muted-foreground',
            'font-medium tracking-wide'
          )}>
            <svg
              className="w-3 h-3 opacity-60"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <span>{timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ComplexAddons;
