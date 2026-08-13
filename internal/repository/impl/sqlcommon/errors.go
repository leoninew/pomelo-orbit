package sqlcommon

import (
	"database/sql"
	"errors"

	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// TranslateError maps database-specific not-found errors to the repository
// contract while preserving the original error text and cause.
func TranslateError(err error) error {
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return notFoundError{err: err}
}

type notFoundError struct {
	err error
}

func (e notFoundError) Error() string {
	return e.err.Error()
}

func (e notFoundError) Unwrap() []error {
	return []error{repository.ErrNotFound, e.err}
}
