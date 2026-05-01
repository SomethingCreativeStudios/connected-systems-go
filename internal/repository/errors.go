package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("record not found")

// ErrHasChildren is returned when a resource cannot be deleted because
// dependent records exist and cascade was not requested.
var ErrHasChildren = errors.New("resource has dependent records")

// isFKViolation reports whether err is a PostgreSQL foreign key violation (SQLSTATE 23503).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
