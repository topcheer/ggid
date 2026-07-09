package repository

import (
	"github.com/ggid/ggid/pkg/errors"
	"github.com/jackc/pgx/v5"
)

// mapErr translates pgx errors into GGID errors.
func mapErr(err error, resource, id string) error {
	if err == pgx.ErrNoRows {
		return errors.NotFound(resource, id)
	}
	return errors.Wrap(errors.ErrInternal, "database error", err)
}

func notFound(resource, id string) error {
	return errors.NotFound(resource, id)
}
