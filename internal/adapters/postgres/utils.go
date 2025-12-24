package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const DefaultQueryTimeout = 30 * time.Second

// withTimeout wraps a context with a default query timeout if not already set
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// Check if context already has a deadline
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, DefaultQueryTimeout)
}

// Nullable field converters - from Go to SQL
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt32 {
	if i == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(i), Valid: true}
}

func nullFloat32(f float32) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: float64(f), Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func ptrIntToInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Nullable field extractors - from SQL to Go
// These reduce boilerplate when scanning nullable fields

// getString extracts a string from sql.NullString, returning empty string if null
func getString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// getInt extracts an int from sql.NullInt32, returning 0 if null
func getInt(ni sql.NullInt32) int {
	if ni.Valid {
		return int(ni.Int32)
	}
	return 0
}

// getTimePtr extracts a *time.Time from sql.NullTime, returning nil if null
func getTimePtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// Error handling helpers

// checkNoRows returns true if the error is pgx.ErrNoRows (indicating no result found)
func checkNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// JSON helpers

// unmarshalJSONField unmarshals a JSON byte slice into the target pointer
// Returns nil if data is empty (no error for empty data)
func unmarshalJSONField[T any](data []byte, target *T) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

// unmarshalJSONPointer unmarshals JSON data into a new pointer of type T
// Returns nil pointer if data is empty
func unmarshalJSONPointer[T any](data []byte) (*T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// marshalJSONField marshals a value to JSON, handling nil pointers
// Returns nil byte slice for nil pointers
func marshalJSONField[T any](value *T) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

// unmarshalJSONSlice unmarshals JSON data into a slice of type T
// Returns nil slice if data is empty
func unmarshalJSONSlice[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var result []T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Row scanner - reduces duplicate scanning logic

// ScanRow is a generic helper that handles common error patterns for single-row queries
// It executes the provided scanner function and standardizes error handling
type RowScanner[T any] func(row pgx.Row) (*T, error)
