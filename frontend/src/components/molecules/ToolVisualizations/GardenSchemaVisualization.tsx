import React, { useState } from 'react';
import { cls } from '../../../utils/cls';

interface SchemaColumn {
  name: string;
  type: string;
  nullable: boolean;
}

interface SchemaTable {
  name: string;
  description?: string;
  column_count: number;
  columns: SchemaColumn[];
}

interface SchemaRelationship {
  source_table: string;
  source_column: string;
  target_table: string;
  target_column: string;
}

interface GardenSchemaResult {
  question?: string;
  tables: SchemaTable[];
  relationships: SchemaRelationship[];
  table_count: number;
}

interface GardenSchemaVisualizationProps {
  result: GardenSchemaResult;
  className?: string;
}

const GardenSchemaVisualization: React.FC<GardenSchemaVisualizationProps> = ({ result, className }) => {
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'list' | 'diagram'>('list');

  const selectedTableData = result.tables.find((t) => t.name === selectedTable);

  // Build a map of relationships for quick lookup
  const relationshipMap = new Map<string, SchemaRelationship[]>();
  result.relationships.forEach((rel) => {
    if (!relationshipMap.has(rel.source_table)) {
      relationshipMap.set(rel.source_table, []);
    }
    relationshipMap.get(rel.source_table)!.push(rel);
  });

  return (
    <div className={cls('rounded-lg border bg-gradient-to-br from-violet-50 to-indigo-50 dark:from-violet-950/30 dark:to-indigo-950/30 overflow-hidden', className)}>
      {/* Header */}
      <div className="px-4 py-3 border-b bg-white/50 dark:bg-black/20">
        <div className="flex items-center gap-2">
          <span className="text-2xl">🗺️</span>
          <div className="flex-1">
            <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">
              Database Schema
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {result.table_count} tables • {result.relationships.length} relationships
            </p>
          </div>
        </div>

        {result.question && (
          <div className="mt-2 p-2 bg-violet-100 dark:bg-violet-900/30 rounded-lg">
            <p className="text-xs text-violet-700 dark:text-violet-300 italic">
              "{result.question}"
            </p>
          </div>
        )}

        {/* View mode toggle */}
        <div className="flex gap-1 mt-2">
          <button
            onClick={() => setViewMode('list')}
            className={cls(
              'px-3 py-1 rounded-full text-xs font-medium transition-colors',
              viewMode === 'list'
                ? 'bg-violet-600 text-white'
                : 'bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/50 dark:text-violet-300'
            )}
          >
            📋 List
          </button>
          <button
            onClick={() => setViewMode('diagram')}
            className={cls(
              'px-3 py-1 rounded-full text-xs font-medium transition-colors',
              viewMode === 'diagram'
                ? 'bg-violet-600 text-white'
                : 'bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/50 dark:text-violet-300'
            )}
          >
            🔗 Relationships
          </button>
        </div>
      </div>

      {viewMode === 'list' ? (
        <div className="flex">
          {/* Tables list */}
          <div className="w-1/3 border-r max-h-96 overflow-auto">
            {result.tables.map((table) => {
              const hasRelations = relationshipMap.has(table.name) ||
                result.relationships.some((r) => r.target_table === table.name);

              return (
                <button
                  key={table.name}
                  onClick={() => setSelectedTable(table.name === selectedTable ? null : table.name)}
                  className={cls(
                    'w-full px-3 py-2 text-left hover:bg-white/30 dark:hover:bg-black/10 border-b transition-colors',
                    selectedTable === table.name && 'bg-violet-100 dark:bg-violet-900/30'
                  )}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm">📊</span>
                    <div className="flex-1 min-w-0">
                      <span className="font-mono text-xs font-medium text-gray-900 dark:text-gray-100 truncate block">
                        {table.name}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {table.column_count} cols
                        {hasRelations && ' • 🔗'}
                      </span>
                    </div>
                  </div>
                </button>
              );
            })}
          </div>

          {/* Table details */}
          <div className="flex-1 p-4 max-h-96 overflow-auto">
            {selectedTableData ? (
              <>
                <h4 className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
                  {selectedTableData.name}
                </h4>
                {selectedTableData.description && (
                  <p className="text-xs text-gray-600 dark:text-gray-400 mb-3">
                    {selectedTableData.description}
                  </p>
                )}
                <div className="space-y-1">
                  {selectedTableData.columns.map((col, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-2 text-xs bg-white/50 dark:bg-black/20 rounded px-2 py-1"
                    >
                      <span className="font-mono font-medium text-gray-700 dark:text-gray-300 flex-1">
                        {col.name}
                      </span>
                      <span className="font-mono text-gray-500 dark:text-gray-400">
                        {col.type}
                      </span>
                      {!col.nullable && (
                        <span className="text-red-500 text-xs" title="NOT NULL">
                          •
                        </span>
                      )}
                    </div>
                  ))}
                </div>

                {/* Show relationships for this table */}
                {relationshipMap.has(selectedTableData.name) && (
                  <div className="mt-4">
                    <h5 className="text-xs font-semibold text-gray-700 dark:text-gray-300 mb-2">
                      References
                    </h5>
                    {relationshipMap.get(selectedTableData.name)!.map((rel, i) => (
                      <div key={i} className="text-xs text-gray-600 dark:text-gray-400">
                        <span className="font-mono">{rel.source_column}</span>
                        <span className="mx-2">→</span>
                        <span className="font-mono text-violet-600 dark:text-violet-400">
                          {rel.target_table}.{rel.target_column}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </>
            ) : (
              <div className="h-full flex items-center justify-center text-gray-400 dark:text-gray-500">
                <p className="text-sm">← Select a table to view details</p>
              </div>
            )}
          </div>
        </div>
      ) : (
        /* Relationships view */
        <div className="p-4 max-h-96 overflow-auto">
          {result.relationships.length > 0 ? (
            <div className="space-y-2">
              {result.relationships.map((rel, i) => (
                <div
                  key={i}
                  className="flex items-center gap-3 bg-white/50 dark:bg-black/20 rounded-lg p-3"
                >
                  <div className="flex-1">
                    <span className="font-mono text-xs font-medium text-gray-900 dark:text-gray-100">
                      {rel.source_table}
                    </span>
                    <span className="font-mono text-xs text-gray-500 dark:text-gray-400">
                      .{rel.source_column}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-violet-500">
                    <span className="w-8 h-px bg-current" />
                    <span>🔗</span>
                    <span className="w-8 h-px bg-current" />
                  </div>
                  <div className="flex-1 text-right">
                    <span className="font-mono text-xs font-medium text-gray-900 dark:text-gray-100">
                      {rel.target_table}
                    </span>
                    <span className="font-mono text-xs text-gray-500 dark:text-gray-400">
                      .{rel.target_column}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-gray-400 dark:text-gray-500 py-8">
              <span className="text-3xl mb-2 block">🔗</span>
              No foreign key relationships found
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default GardenSchemaVisualization;
