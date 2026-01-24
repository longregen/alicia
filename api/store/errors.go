package store

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/longregen/alicia/api/domain"
)

// WrapError wraps an error with an operation context.
func WrapError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, err)
}

// HandleNotFound converts pgx.ErrNoRows to domain.ErrNotFound.
// Returns the original error if it's not a no-rows error.
func HandleNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

// WrapNotFound wraps an error, converting pgx.ErrNoRows to domain.ErrNotFound.
func WrapNotFound(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return WrapError(operation, err)
}
