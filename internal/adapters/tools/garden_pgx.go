package tools

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxGardenDB implements GardenDB using pgx
type PgxGardenDB struct {
	pool *pgxpool.Pool
}

// NewPgxGardenDB creates a new PgxGardenDB
func NewPgxGardenDB(pool *pgxpool.Pool) *PgxGardenDB {
	return &PgxGardenDB{pool: pool}
}

func (db *PgxGardenDB) Query(ctx context.Context, sql string, args ...any) (GardenRows, error) {
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{rows: rows}, nil
}

func (db *PgxGardenDB) QueryRow(ctx context.Context, sql string, args ...any) GardenRow {
	return db.pool.QueryRow(ctx, sql, args...)
}

// pgxRows wraps pgx.Rows to implement GardenRows
type pgxRows struct {
	rows pgx.Rows
}

func (r *pgxRows) Next() bool {
	return r.rows.Next()
}

func (r *pgxRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *pgxRows) Close() {
	r.rows.Close()
}

func (r *pgxRows) Columns() []string {
	fields := r.rows.FieldDescriptions()
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = string(f.Name)
	}
	return cols
}

func (r *pgxRows) Values() ([]any, error) {
	return r.rows.Values()
}
