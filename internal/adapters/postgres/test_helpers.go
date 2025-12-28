package postgres

import (
	"context"

	"github.com/pashagolub/pgxmock/v4"
)

// setupMockContext creates a context with the mock as a transaction
// This allows the BaseRepository.conn() method to return the mock
func setupMockContext(mock pgxmock.PgxPoolIface) context.Context {
	return context.WithValue(context.Background(), txKey, mock)
}
