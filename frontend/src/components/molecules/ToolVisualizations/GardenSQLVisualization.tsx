import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface GardenSQLResult {
  success: boolean;
  columns?: string[];
  rows?: Record<string, unknown>[];
  row_count: number;
  truncated?: boolean;
  error?: string;
}

interface GardenSQLVisualizationProps {
  result: GardenSQLResult;
  className?: string;
}

const GardenSQLVisualization: React.FC<GardenSQLVisualizationProps> = ({ result, className }) => {
  const [currentPage, setCurrentPage] = useState(0);
  const rowsPerPage = 10;

  if (!result.success) {
    return (
      <div className={cls('rounded-lg border bg-gradient-to-br from-red-50 to-rose-50 dark:from-red-950/30 dark:to-rose-950/30 overflow-hidden', className)}>
        <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
          <div className="flex items-center gap-2">
            <span className="text-2xl">❌</span>
            <div className="flex-1">
              <h3 className="font-semibold text-sm text-red-700 dark:text-red-300">
                Query Failed
              </h3>
            </div>
          </div>
        </div>
        <div className="p-4">
          <pre className="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/50 rounded-lg p-3 whitespace-pre-wrap font-mono">
            {result.error}
          </pre>
        </div>
      </div>
    );
  }

  const rows = result.rows || [];
  const columns = result.columns || [];
  const totalPages = Math.ceil(rows.length / rowsPerPage);
  const displayedRows = rows.slice(currentPage * rowsPerPage, (currentPage + 1) * rowsPerPage);

  const formatValue = (value: unknown): string => {
    if (value === null) return 'NULL';
    if (value === undefined) return '';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  const getValueClass = (value: unknown): string => {
    if (value === null) return 'text-gray-400 italic';
    if (typeof value === 'number') return 'text-blue-600 dark:text-blue-400';
    if (typeof value === 'boolean') return value ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400';
    return 'text-gray-700 dark:text-gray-300';
  };

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-emerald-50 to-green-50 dark:from-emerald-950/30 dark:to-green-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">⚡</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              Query Results
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {result.row_count} row{result.row_count !== 1 ? 's' : ''} returned
              {result.truncated && (
                <span className="ml-2 text-amber-600 dark:text-amber-400">
                  (truncated)
                </span>
              )}
            </p>
          </div>
          <span className="px-2 py-1 rounded-full text-xs font-bold bg-green-200 text-green-800 dark:bg-green-800 dark:text-green-200">
            ✓ Success
          </span>
        </div>
      </div>

      {/* Results table */}
      {rows.length > 0 ? (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="bg-white/50 dark:bg-black/20 border-b sticky top-0">
                <tr>
                  <th className="px-3 py-2 text-center font-semibold text-gray-500 dark:text-gray-400 w-8">
                    #
                  </th>
                  {columns.map((col, i) => (
                    <th
                      key={i}
                      className="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-300 font-mono"
                    >
                      {col}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {displayedRows.map((row, rowIndex) => (
                  <tr
                    key={rowIndex}
                    className="hover:bg-white/30 dark:hover:bg-black/10"
                  >
                    <td className="px-3 py-2 text-center text-gray-400 text-xs">
                      {currentPage * rowsPerPage + rowIndex + 1}
                    </td>
                    {columns.map((col, colIndex) => (
                      <td
                        key={colIndex}
                        className={cls(
                          'px-3 py-2 font-mono max-w-xs truncate',
                          getValueClass(row[col])
                        )}
                        title={formatValue(row[col])}
                      >
                        {formatValue(row[col])}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="px-4 py-2 border-t bg-white/30 dark:bg-black/10 flex items-center justify-between">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                Page {currentPage + 1} of {totalPages}
              </span>
              <div className="flex gap-1">
                <button
                  onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                  disabled={currentPage === 0}
                  className={cls(
                    'px-2 py-1 rounded text-xs font-medium',
                    currentPage === 0
                      ? 'bg-gray-100 text-gray-400 cursor-not-allowed dark:bg-gray-800'
                      : 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/50 dark:text-emerald-300'
                  )}
                >
                  ← Prev
                </button>
                <button
                  onClick={() => setCurrentPage(Math.min(totalPages - 1, currentPage + 1))}
                  disabled={currentPage === totalPages - 1}
                  className={cls(
                    'px-2 py-1 rounded text-xs font-medium',
                    currentPage === totalPages - 1
                      ? 'bg-gray-100 text-gray-400 cursor-not-allowed dark:bg-gray-800'
                      : 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/50 dark:text-emerald-300'
                  )}
                >
                  Next →
                </button>
              </div>
            </div>
          )}
        </>
      ) : (
        <div className="p-8 text-center text-gray-500 dark:text-gray-400">
          <span className="text-3xl mb-2 block">📭</span>
          Query returned no rows
        </div>
      )}
    </div>
  );
};

export default GardenSQLVisualization;
