package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/longregen/alicia/shared/mcp"
)

func (s *Server) getTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "describe_table",
			Description: "Get detailed information about a database table including columns, types, constraints, and basic statistics. Use this to understand a table's structure before writing queries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table": map[string]any{
						"type":        "string",
						"description": "The table name to describe",
					},
				},
				"required": []string{"table"},
			},
		},
		{
			Name:        "execute_sql",
			Description: "Execute a SQL query against the database. Returns results as JSON. On errors, provides hints to fix the query. Only SELECT queries allowed by default.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sql": map[string]any{
						"type":        "string",
						"description": "The SQL query to execute",
					},
					"allow_mutation": map[string]any{
						"type":        "boolean",
						"description": "Set to true to allow INSERT, UPDATE, DELETE queries",
						"default":     false,
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			Name:        "schema_explore",
			Description: "Ask a natural language question about the database schema. Returns information about tables, relationships, and how to query specific data.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{
						"type":        "string",
						"description": "A question about the database, e.g., 'What tables store user data?' or 'How are messages linked to contacts?'",
					},
					"max_tokens": map[string]any{
						"type":        "integer",
						"description": "Maximum tokens for the response (default: 2048)",
						"default":     2048,
					},
				},
				"required": []string{"question"},
			},
		},
	}
}

func (s *Server) describeTable(ctx context.Context, args map[string]any) (string, bool) {
	table, ok := args["table"].(string)
	if !ok || table == "" {
		return "Error: 'table' parameter is required", true
	}

	if !isValidIdentifier(table) {
		return "Error: invalid table name", true
	}

	// Get columns
	colQuery := `
		SELECT column_name, data_type, is_nullable = 'YES' as nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	colRows, err := s.pool.Query(ctx, colQuery, table)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), true
	}
	defer colRows.Close()

	type columnInfo struct {
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Nullable bool    `json:"nullable"`
		Default  *string `json:"default,omitempty"`
	}

	var columns []columnInfo
	for colRows.Next() {
		var c columnInfo
		if err := colRows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Default); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}
		columns = append(columns, c)
	}

	if len(columns) == 0 {
		return fmt.Sprintf("Table '%s' not found", table), true
	}

	// Get primary key
	pkQuery := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass AND i.indisprimary
	`
	var primaryKeys []string
	pkRows, _ := s.pool.Query(ctx, pkQuery, table)
	if pkRows != nil {
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
	type fkInfo struct {
		Column     string `json:"column"`
		References string `json:"references"`
	}
	var foreignKeys []fkInfo
	fkRows, _ := s.pool.Query(ctx, fkQuery, table)
	if fkRows != nil {
		for fkRows.Next() {
			var col, refTable, refCol string
			fkRows.Scan(&col, &refTable, &refCol)
			foreignKeys = append(foreignKeys, fkInfo{
				Column:     col,
				References: fmt.Sprintf("%s.%s", refTable, refCol),
			})
		}
		fkRows.Close()
	}

	// Get reverse foreign keys (tables referencing this table)
	rfkQuery := `
		SELECT tc.table_name, kcu.column_name, ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND ccu.table_name = $1
	`
	type rfkInfo struct {
		Table  string `json:"table"`
		Column string `json:"column"`
		On     string `json:"on"`
	}
	var referencedBy []rfkInfo
	rfkRows, _ := s.pool.Query(ctx, rfkQuery, table)
	if rfkRows != nil {
		for rfkRows.Next() {
			var refTable, refCol, localCol string
			rfkRows.Scan(&refTable, &refCol, &localCol)
			referencedBy = append(referencedBy, rfkInfo{
				Table:  refTable,
				Column: refCol,
				On:     localCol,
			})
		}
		rfkRows.Close()
	}

	// Get row count
	var rowCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	s.pool.QueryRow(ctx, countQuery).Scan(&rowCount)

	// Build result
	result := map[string]any{
		"table":     table,
		"columns":   columns,
		"row_count": rowCount,
	}
	if len(primaryKeys) > 0 {
		result["primary_key"] = primaryKeys
	}
	if len(foreignKeys) > 0 {
		result["foreign_keys"] = foreignKeys
	}
	if len(referencedBy) > 0 {
		result["referenced_by"] = referencedBy
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return string(jsonResult), false
}

func (s *Server) executeSQL(ctx context.Context, args map[string]any) (string, bool) {
	sql, ok := args["sql"].(string)
	if !ok || sql == "" {
		return "Error: 'sql' parameter is required", true
	}

	allowMutation, _ := args["allow_mutation"].(bool)

	// Fast-path hint: warn the user before even starting a transaction
	if !allowMutation && isMutationQuery(sql) {
		return "Error: This appears to be a mutation query. Set allow_mutation=true to enable.", true
	}

	// Use a read-only transaction to enforce no mutations at the database level,
	// preventing bypasses via CTEs, comments, semicolons, etc.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Sprintf("Error starting transaction: %v", err), true
	}
	defer tx.Rollback(ctx)

	if !allowMutation {
		if _, err := tx.Exec(ctx, "SET TRANSACTION READ ONLY"); err != nil {
			return fmt.Sprintf("Error setting read-only transaction: %v", err), true
		}
	}

	rows, err := tx.Query(ctx, sql)
	if err != nil {
		hint := s.llm.GenerateSQLHint(ctx, sql, err.Error(), s.config.SchemaDoc)
		result := map[string]any{
			"success": false,
			"error":   err.Error(),
			"hint":    hint,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return string(jsonResult), true
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = string(f.Name)
	}

	// UUID shortening: map full UUIDs to short aliases
	uuidMap := make(map[string]string) // full UUID -> short alias
	uuidCounter := 1

	var results []map[string]any
	const maxRows = 500
	for rows.Next() && len(results) < maxRows {
		values, err := rows.Values()
		if err != nil {
			return fmt.Sprintf("Error reading row: %v", err), true
		}
		row := make(map[string]any)
		for i, col := range columns {
			row[col] = formatValue(values[i], uuidMap, &uuidCounter)
		}
		results = append(results, row)
	}

	output := map[string]any{
		"success":   true,
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
	}
	if len(results) >= maxRows {
		output["truncated"] = true
	}

	// Include UUID mapping legend if we shortened any UUIDs
	if len(uuidMap) > 0 {
		// Invert map for legend: short alias -> full UUID
		legend := make(map[string]string)
		for full, short := range uuidMap {
			legend[short] = full
		}
		output["uuid_legend"] = legend
	}

	jsonResult, _ := json.MarshalIndent(output, "", "  ")

	// Check response size limit
	if len(jsonResult) > s.config.MaxResponseSize {
		return fmt.Sprintf("Error: Response too large (%d characters, limit %d). Use LIMIT in your query to reduce results.", len(jsonResult), s.config.MaxResponseSize), true
	}

	return string(jsonResult), false
}

// schema_explore implementation
func (s *Server) schemaExplore(ctx context.Context, args map[string]any) (string, bool) {
	question, ok := args["question"].(string)
	if !ok || question == "" {
		return "Error: 'question' parameter is required", true
	}

	maxTokens := 2048
	if v, ok := args["max_tokens"].(float64); ok && v > 0 {
		maxTokens = int(v)
	}

	// Build schema context from database if no doc provided
	schemaContext := s.config.SchemaDoc
	if schemaContext == "" {
		schemaContext = s.buildSchemaContext(ctx)
	}

	// If LLM configured, use it
	if s.llm.IsConfigured() {
		answer, err := s.llm.AnswerSchemaQuestion(ctx, question, schemaContext, maxTokens)
		if err != nil {
			return fmt.Sprintf("LLM error: %v\n\nSchema:\n%s", err, schemaContext), false
		}
		return answer, false
	}

	// No LLM, return raw schema
	return fmt.Sprintf("Question: %s\n\n%s", question, schemaContext), false
}

// buildSchemaContext generates schema info from the database
func (s *Server) buildSchemaContext(ctx context.Context) string {
	query := `
		SELECT table_name,
		       (SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_name = t.table_name) as col_count
		FROM information_schema.tables t
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return "Unable to query schema"
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("Tables in database:\n")
	for rows.Next() {
		var name string
		var colCount int
		rows.Scan(&name, &colCount)
		sb.WriteString(fmt.Sprintf("- %s (%d columns)\n", name, colCount))
	}
	return sb.String()
}

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

// isMutationQuery is a fast-path heuristic to detect mutation queries.
// The actual enforcement is done via SET TRANSACTION READ ONLY in executeSQL.
func isMutationQuery(sql string) bool {
	keywords := []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "GRANT", "REVOKE"}
	upper := strings.ToUpper(strings.TrimSpace(sql))
	for _, kw := range keywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// formatValue converts database values to LLM-friendly formats.
// UUIDs are shortened to $1, $2, etc. and tracked in uuidMap for the legend.
func formatValue(v any, uuidMap map[string]string, counter *int) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case [16]byte:
		// UUID as fixed-size byte array
		uuid := formatUUID(val[:])
		return shortenUUID(uuid, uuidMap, counter)

	case []byte:
		// Could be UUID (16 bytes) or other binary data
		if len(val) == 16 {
			uuid := formatUUID(val)
			return shortenUUID(uuid, uuidMap, counter)
		}
		// Other binary data: return as hex
		return "0x" + hex.EncodeToString(val)

	case pgtype.UUID:
		if !val.Valid {
			return nil
		}
		uuid := formatUUID(val.Bytes[:])
		return shortenUUID(uuid, uuidMap, counter)

	case time.Time:
		return val.Format(time.RFC3339)

	case pgtype.Timestamp:
		if !val.Valid {
			return nil
		}
		return val.Time.Format(time.RFC3339)

	case pgtype.Timestamptz:
		if !val.Valid {
			return nil
		}
		return val.Time.Format(time.RFC3339)

	case pgtype.Text:
		if !val.Valid {
			return nil
		}
		return val.String

	case pgtype.Int4:
		if !val.Valid {
			return nil
		}
		return val.Int32

	case pgtype.Int8:
		if !val.Valid {
			return nil
		}
		return val.Int64

	case pgtype.Bool:
		if !val.Valid {
			return nil
		}
		return val.Bool

	default:
		return val
	}
}

// formatUUID converts 16 bytes to standard UUID string format
func formatUUID(b []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// shortenUUID returns a short alias for a UUID, reusing existing alias if seen before
func shortenUUID(uuid string, uuidMap map[string]string, counter *int) string {
	if short, exists := uuidMap[uuid]; exists {
		return short
	}
	short := fmt.Sprintf("$%d", *counter)
	uuidMap[uuid] = short
	*counter++
	return short
}
