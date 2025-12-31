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

func nullStringPtr(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func ptrIntToInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func getInt(ni sql.NullInt32) int {
	if ni.Valid {
		return int(ni.Int32)
	}
	return 0
}

func getTimePtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func getStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func checkNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func unmarshalJSONField[T any](data []byte, target *T) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

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

func marshalJSONField[T any](value *T) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

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

type RowScanner[T any] func(row pgx.Row) (*T, error)
