package transformers

import (
	"errors"

	"github.com/lib/pq"
	"github.com/zoobz-io/rocco"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrorToResponse converts a domain error into the HTTP error the admin surface
// returns — the domain -> wire half for the error path. An unrecognized error
// is passed through unchanged (rocco renders it as a 500).
func ErrorToResponse(err error) error {
	switch {
	case errors.Is(err, stores.ErrNotFound):
		return rocco.ErrNotFound.WithMessage("resource not found")
	case errors.Is(err, stores.ErrDocumentPublished):
		return rocco.ErrConflict.WithMessage("document is published; unpublish before deleting")
	case errors.Is(err, stores.ErrVersionMismatch):
		return rocco.ErrBadRequest.WithMessage("version does not belong to the document")
	case errors.Is(err, auth.ErrNoTenant):
		return rocco.ErrUnauthorized.WithMessage("request has no tenant")
	case errors.Is(err, auth.ErrNoUser):
		return rocco.ErrUnauthorized.WithMessage("request has no acting user")
	case isUniqueViolation(err):
		return rocco.ErrConflict.WithMessage("a document with that key already exists")
	default:
		return err
	}
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (a duplicate key).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
