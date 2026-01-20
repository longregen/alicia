import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface ColumnInfo {
  name: string;
  type: string;
  nullable: boolean;
  default?: string;
}

interface ForeignKey {
  column: string;
  references: string;
}

interface Index {
  name: string;
  definition: string;
}

interface GardenTableResult {
  table: string;
  columns: ColumnInfo[];
  row_count: number;
  primary_key?: string[];
  foreign_keys?: ForeignKey[];
  indexes?: Index[];
}

interface GardenTableVisualizationProps {
  result: GardenTableResult;
  className?: string;
}

const GardenTableVisualization: React.FC<GardenTableVisualizationProps> = ({ result, className }) => {
  const [showIndexes, setShowIndexes] = useState(false);

  const getTypeColor = (type: string) => {
    const t = type.toLowerCase();
    if (t.includes('int') || t.includes('serial') || t.includes('numeric') || t.includes('decimal')) {
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-200';
    }
    if (t.includes('text') || t.includes('varchar') || t.includes('char')) {
      return 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-200';
    }
    if (t.includes('bool')) {
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900/50 dark:text-purple-200';
    }
    if (t.includes('timestamp') || t.includes('date') || t.includes('time')) {
      return 'bg-orange-100 text-orange-800 dark:bg-orange-900/50 dark:text-orange-200';
    }
    if (t.includes('json') || t.includes('array')) {
      return 'bg-pink-100 text-pink-800 dark:bg-pink-900/50 dark:text-pink-200';
    }
    if (t.includes('uuid')) {
      return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/50 dark:text-cyan-200';
    }
    return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
  };

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-indigo-50 to-blue-50 dark:from-indigo-950/30 dark:to-blue-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">📊</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100 font-mono">
              {result.table}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {result.columns.length} columns • {result.row_count.toLocaleString()} rows
            </p>
          </div>
        </div>

        {/* Stats badges */}
        <div className="flex flex-wrap gap-2 mt-2">
          {result.primary_key && result.primary_key.length > 0 && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-200">
              🔑 PK: {result.primary_key.join(', ')}
            </span>
          )}
          {result.foreign_keys && result.foreign_keys.length > 0 && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-200">
              🔗 {result.foreign_keys.length} FK
            </span>
          )}
          {result.indexes && result.indexes.length > 0 && (
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-teal-100 text-teal-800 dark:bg-teal-900/50 dark:text-teal-200">
              📇 {result.indexes.length} indexes
            </span>
          )}
        </div>
      </div>

      {/* Columns table */}
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead className="bg-white/50 dark:bg-black/20 border-b">
            <tr>
              <th className="px-4 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Column</th>
              <th className="px-4 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Type</th>
              <th className="px-4 py-2 text-center font-semibold text-gray-700 dark:text-gray-300">Nullable</th>
              <th className="px-4 py-2 text-left font-semibold text-gray-700 dark:text-gray-300">Default</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {result.columns.map((col, i) => {
              const isPK = result.primary_key?.includes(col.name);
              const fk = result.foreign_keys?.find((f) => f.column === col.name);

              return (
                <tr key={i} className="hover:bg-white/30 dark:hover:bg-black/10">
                  <td className="px-4 py-2">
                    <div className="flex items-center gap-2">
                      {isPK && <span title="Primary Key">🔑</span>}
                      {fk && <span title={`Foreign Key → ${fk.references}`}>🔗</span>}
                      <span className="font-mono font-medium text-gray-900 dark:text-gray-100">
                        {col.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-2">
                    <span className={cls('px-2 py-0.5 rounded text-xs font-mono', getTypeColor(col.type))}>
                      {col.type}
                    </span>
                  </td>
                  <td className="px-4 py-2 text-center">
                    {col.nullable ? (
                      <span className="text-gray-400">○</span>
                    ) : (
                      <span className="text-red-500" title="NOT NULL">●</span>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    {col.default ? (
                      <code className="text-xs text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-1 rounded">
                        {col.default}
                      </code>
                    ) : (
                      <span className="text-gray-400">-</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Foreign Keys */}
      {result.foreign_keys && result.foreign_keys.length > 0 && (
        <div className="px-4 py-3 border-t bg-white/30 dark:bg-black/10">
          <h4 className="text-xs font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Foreign Keys
          </h4>
          <div className="space-y-1">
            {result.foreign_keys.map((fk, i) => (
              <div key={i} className="text-xs text-gray-600 dark:text-gray-400">
                <span className="font-mono">{fk.column}</span>
                <span className="mx-2">→</span>
                <span className="font-mono text-indigo-600 dark:text-indigo-400">{fk.references}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Indexes (collapsible) */}
      {result.indexes && result.indexes.length > 0 && (
        <div className="px-4 py-2 border-t bg-white/30 dark:bg-black/10">
          <button
            onClick={() => setShowIndexes(!showIndexes)}
            className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline flex items-center gap-1"
          >
            <span>{showIndexes ? '▼' : '▶'}</span>
            {showIndexes ? 'Hide indexes' : `Show ${result.indexes.length} indexes`}
          </button>
          {showIndexes && (
            <div className="mt-2 space-y-2">
              {result.indexes.map((idx, i) => (
                <div key={i} className="bg-white/50 dark:bg-black/20 rounded p-2">
                  <div className="font-mono text-xs font-medium text-gray-700 dark:text-gray-300">
                    {idx.name}
                  </div>
                  <div className="font-mono text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
                    {idx.definition}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default GardenTableVisualization;
