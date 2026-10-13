package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("record not found")

// ErrParentNotFound is returned when a nested resource is created under a
// parent that does not exist.
var ErrParentNotFound = errors.New("parent resource not found")

// ErrHasChildren is returned when a resource cannot be deleted because
// dependent records exist and cascade was not requested.
var ErrHasChildren = errors.New("resource has dependent records")

// IsNotFound reports whether err means the requested resource does not exist,
// whether it came from a repository method or straight from gorm.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}

// IsUniqueViolation reports whether err is a PostgreSQL unique constraint
// violation (SQLSTATE 23505), such as a duplicate unique identifier.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isFKViolation reports whether err is a PostgreSQL foreign key violation (SQLSTATE 23503).
func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// ErrSchemaInUse protects the interpretation of existing observations or commands.
var ErrSchemaInUse = errors.New("schema cannot change while dependent records exist")

// ValidationError denotes a rejected resource payload, rather than a storage failure.
type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }
