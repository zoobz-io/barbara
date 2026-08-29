package transformers

import (
	"errors"

	"github.com/lib/pq"
	"github.com/zoobz-io/rocco"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrorToResponse converts a domain error into the HTTP error the public API
// returns — the domain -> wire half for the error path. An unrecognized error
// is passed through unchanged (rocco renders it as a 500).
func ErrorToResponse(err error) error {
	// A stale-base_version save conflict carries the current head so the client
	// can refetch and rebase.
	var conflict *stores.VersionConflictError
	if errors.As(err, &conflict) {
		return rocco.ErrConflict.
			WithMessage("base_version is stale").
			WithDetails(rocco.ConflictDetails{Reason: conflict.Error()})
	}
	switch {
	case errors.Is(err, stores.ErrNotFound):
		return rocco.ErrNotFound.WithMessage("resource not found")
	case errors.Is(err, stores.ErrDocumentPublished):
		return rocco.ErrConflict.WithMessage("document is published; unpublish before deleting")
	case errors.Is(err, stores.ErrAppNameTaken):
		return rocco.ErrConflict.WithMessage("an app with that name already exists")
	case errors.Is(err, stores.ErrAppHasReleases):
		return rocco.ErrConflict.WithMessage("app has releases; releases are never deleted")
	case errors.Is(err, stores.ErrCollectionNotEmpty):
		return rocco.ErrConflict.WithMessage("collection is not empty")
	case errors.Is(err, stores.ErrCollectionNameTaken):
		return rocco.ErrConflict.WithMessage("a collection or document with that name already exists in the parent")
	case errors.Is(err, stores.ErrCollectionCycle):
		return rocco.ErrConflict.WithMessage("cannot move a collection into itself or a descendant")
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
