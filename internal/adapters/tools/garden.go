package tools

import (
	"context"
	"fmt"
	"strings"
)

// GardenDB defines the interface for database operations required by garden tools
type GardenDB interface {
	// Query executes a query and returns rows
	Query(ctx context.Context, sql string, args ...any) (GardenRows, error)
	// QueryRow executes a query that returns a single row
	QueryRow(ctx context.Context, sql string, args ...any) GardenRow
}

// GardenRows represents query results
type GardenRows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Columns() []string
	Values() ([]any, error)
}

// GardenRow represents a single row result
type GardenRow interface {
	Scan(dest ...any) error
}

// GardenDescribeTableTool describes a database table
type GardenDescribeTableTool struct {
	db GardenDB
}

func NewGardenDescribeTableTool(db GardenDB) *GardenDescribeTableTool {
	return &GardenDescribeTableTool{db: db}
}

func (t *GardenDescribeTableTool) Name() string {
	return "garden_describe_table"
}

func (t *GardenDescribeTableTool) Description() string {
	return "Get detailed information about a database table including columns, types, constraints, and basic statistics. Use this to understand a table's structure before writing queries."
}

func (t *GardenDescribeTableTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"table": map[string]any{
				"type":        "string",
				"description": "The table name to describe",
			},
		},
		"required": []string{"table"},
	}
}

func (t *GardenDescribeTableTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	table, ok := args["table"].(string)
	if !ok || table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	if !isValidIdentifier(table) {
		return nil, fmt.Errorf("invalid table name")
	}

	// Get columns
	colQuery := `
		SELECT column_name, data_type, is_nullable = 'YES' as nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	colRows, err := t.db.Query(ctx, colQuery, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer colRows.Close()

	var columns []GardenColumnInfo
	for colRows.Next() {
		var c GardenColumnInfo
		var defaultVal *string
		if err := colRows.Scan(&c.Name, &c.Type, &c.Nullable, &defaultVal); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		if defaultVal != nil {
			c.Default = *defaultVal
		}
		columns = append(columns, c)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("table '%s' not found", table)
	}

	// Get primary key
	pkQuery := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass AND i.indisprimary
	`
	var primaryKeys []string
	pkRows, err := t.db.Query(ctx, pkQuery, table)
	if err == nil {
		for pkRows.Next() {
			var pk string
			pkRows.Scan(&pk)
			primaryKeys = append(primaryKeys, pk)
		}
		pkRows.Close()
	}

	// Get foreign keys
	fkQuery := `
		SELECT kcu.column_name, ccu.table_name, ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = $1
	`
	var foreignKeys []GardenForeignKey
	fkRows, err := t.db.Query(ctx, fkQuery, table)
	if err == nil {
		for fkRows.Next() {
			var col, refTable, refCol string
			fkRows.Scan(&col, &refTable, &refCol)
			foreignKeys = append(foreignKeys, GardenForeignKey{
				Column:     col,
				References: fmt.Sprintf("%s.%s", refTable, refCol),
			})
		}
		fkRows.Close()
	}

	// Get row count
	var rowCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	t.db.QueryRow(ctx, countQuery).Scan(&rowCount)

	// Get indexes
	idxQuery := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = $1 AND schemaname = 'public'
	`
	var indexes []GardenIndex
	idxRows, err := t.db.Query(ctx, idxQuery, table)
	if err == nil {
		for idxRows.Next() {
			var idx GardenIndex
			idxRows.Scan(&idx.Name, &idx.Definition)
			indexes = append(indexes, idx)
		}
		idxRows.Close()
	}

	result := GardenTableDescription{
		Table:       table,
		Columns:     columns,
		RowCount:    rowCount,
		PrimaryKey:  primaryKeys,
		ForeignKeys: foreignKeys,
		Indexes:     indexes,
	}

	return result, nil
}

// GardenColumnInfo describes a table column
type GardenColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
}

// GardenForeignKey describes a foreign key relationship
type GardenForeignKey struct {
	Column     string `json:"column"`
	References string `json:"references"`
}

// GardenIndex describes a table index
type GardenIndex struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// GardenTableDescription is the full table description result
type GardenTableDescription struct {
	Table       string             `json:"table"`
	Columns     []GardenColumnInfo `json:"columns"`
	RowCount    int64              `json:"row_count"`
	PrimaryKey  []string           `json:"primary_key,omitempty"`
	ForeignKeys []GardenForeignKey `json:"foreign_keys,omitempty"`
	Indexes     []GardenIndex      `json:"indexes,omitempty"`
}

// GardenExecuteSQLTool executes SQL queries
type GardenExecuteSQLTool struct {
	db              GardenDB
	maxResponseSize int
}

func NewGardenExecuteSQLTool(db GardenDB) *GardenExecuteSQLTool {
	return &GardenExecuteSQLTool{
		db:              db,
		maxResponseSize: 50000, // 50KB default
	}
}

func (t *GardenExecuteSQLTool) Name() string {
	return "garden_execute_sql"
}

func (t *GardenExecuteSQLTool) Description() string {
	return "Execute a SQL query against the database. Returns results as structured data. Only SELECT queries allowed by default; set allow_mutation=true to enable INSERT/UPDATE/DELETE."
}

func (t *GardenExecuteSQLTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sql": map[string]any{
				"type":        "string",
				"description": "The SQL query to execute",
			},
			"allow_mutation": map[string]any{
				"type":        "boolean",
				"description": "Set to true to allow INSERT, UPDATE, DELETE queries (default: false)",
				"default":     false,
			},
		},
		"required": []string{"sql"},
	}
}

func (t *GardenExecuteSQLTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	sql, ok := args["sql"].(string)
	if !ok || sql == "" {
		return nil, fmt.Errorf("sql parameter is required")
	}

	allowMutation := false
	if v, ok := args["allow_mutation"].(bool); ok {
		allowMutation = v
	}

	if !allowMutation && isMutationQuery(sql) {
		return nil, fmt.Errorf("mutation queries not allowed. Set allow_mutation=true to enable INSERT/UPDATE/DELETE")
	}

	rows, err := t.db.Query(ctx, sql)
	if err != nil {
		return GardenSQLResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer rows.Close()

	columns := rows.Columns()

	var results []map[string]any
	const maxRows = 500
	for rows.Next() && len(results) < maxRows {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("error reading row: %w", err)
		}
		row := make(map[string]any)
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	result := GardenSQLResult{
		Success:   true,
		Columns:   columns,
		Rows:      results,
		RowCount:  len(results),
		Truncated: len(results) >= maxRows,
	}

	return result, nil
}

// GardenSQLResult is the result of a SQL query
type GardenSQLResult struct {
	Success   bool             `json:"success"`
	Columns   []string         `json:"columns,omitempty"`
	Rows      []map[string]any `json:"rows,omitempty"`
	RowCount  int              `json:"row_count"`
	Truncated bool             `json:"truncated,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// GardenSchemaExploreTool answers questions about the database schema
type GardenSchemaExploreTool struct {
	db GardenDB
}

func NewGardenSchemaExploreTool(db GardenDB) *GardenSchemaExploreTool {
	return &GardenSchemaExploreTool{db: db}
}

func (t *GardenSchemaExploreTool) Name() string {
	return "garden_schema_explore"
}

func (t *GardenSchemaExploreTool) Description() string {
	return "Explore the database schema. Returns information about all tables, their columns, relationships, and can help with understanding how to query specific data."
}

func (t *GardenSchemaExploreTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "Optional: A specific question about the schema (e.g., 'What tables store user data?'). If not provided, returns full schema overview.",
			},
		},
		"required": []string{},
	}
}

func (t *GardenSchemaExploreTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	question, _ := args["question"].(string)

	// Get all tables with column counts
	tableQuery := `
		SELECT
			t.table_name,
			(SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_name = t.table_name AND c.table_schema = 'public') as column_count,
			obj_description((quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))::regclass::oid, 'pg_class') as description
		FROM information_schema.tables t
		WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name
	`

	rows, err := t.db.Query(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []GardenSchemaTable
	for rows.Next() {
		var table GardenSchemaTable
		var desc *string
		if err := rows.Scan(&table.Name, &table.ColumnCount, &desc); err != nil {
			continue
		}
		if desc != nil {
			table.Description = *desc
		}

		// Get columns for each table
		colQuery := `
			SELECT column_name, data_type, is_nullable = 'YES'
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position
		`
		colRows, err := t.db.Query(ctx, colQuery, table.Name)
		if err == nil {
			for colRows.Next() {
				var col GardenSchemaColumn
				colRows.Scan(&col.Name, &col.Type, &col.Nullable)
				table.Columns = append(table.Columns, col)
			}
			colRows.Close()
		}

		tables = append(tables, table)
	}

	// Get foreign key relationships
	fkQuery := `
		SELECT
			tc.table_name as source_table,
			kcu.column_name as source_column,
			ccu.table_name as target_table,
			ccu.column_name as target_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = 'public'
	`

	var relationships []GardenSchemaRelationship
	fkRows, err := t.db.Query(ctx, fkQuery)
	if err == nil {
		for fkRows.Next() {
			var rel GardenSchemaRelationship
			fkRows.Scan(&rel.SourceTable, &rel.SourceColumn, &rel.TargetTable, &rel.TargetColumn)
			relationships = append(relationships, rel)
		}
		fkRows.Close()
	}

	result := GardenSchemaOverview{
		Tables:        tables,
		Relationships: relationships,
		TableCount:    len(tables),
	}

	if question != "" {
		result.Question = question
	}

	return result, nil
}

// GardenSchemaTable describes a table in the schema
type GardenSchemaTable struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	ColumnCount int                  `json:"column_count"`
	Columns     []GardenSchemaColumn `json:"columns"`
}

// GardenSchemaColumn describes a column
type GardenSchemaColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// GardenSchemaRelationship describes a foreign key relationship
type GardenSchemaRelationship struct {
	SourceTable  string `json:"source_table"`
	SourceColumn string `json:"source_column"`
	TargetTable  string `json:"target_table"`
	TargetColumn string `json:"target_column"`
}

// GardenSchemaOverview is the full schema exploration result
type GardenSchemaOverview struct {
	Question      string                     `json:"question,omitempty"`
	Tables        []GardenSchemaTable        `json:"tables"`
	Relationships []GardenSchemaRelationship `json:"relationships"`
	TableCount    int                        `json:"table_count"`
}

// Helper functions

func isValidIdentifier(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for i, c := range s {
		if i == 0 && c >= '0' && c <= '9' {
			return false
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func isMutationQuery(sql string) bool {
	keywords := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "GRANT", "REVOKE"}
	upper := strings.ToUpper(strings.TrimSpace(sql))
	for _, kw := range keywords {
		if strings.HasPrefix(upper, kw) {
			return true
		}
	}
	return false
}
