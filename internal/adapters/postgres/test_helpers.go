package postgres

import (
	"context"

	"github.com/pashagolub/pgxmock/v4"
)

func setupMockContext(mock pgxmock.PgxPoolIface) context.Context {
	return context.WithValue(context.Background(), txKey, mock)
}
