package transformers

import (
	"errors"

	"github.com/zoobz-io/rocco"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrorToResponse converts a domain error into the HTTP error the site-facing
// surface returns — the domain -> wire half for the error path. An unrecognized
// error is passed through unchanged (rocco renders it as a 500).
func ErrorToResponse(err error) error {
	switch {
	case errors.Is(err, stores.ErrNotFound):
		return rocco.ErrNotFound.WithMessage("document not found")
	case errors.Is(err, auth.ErrNoTenant):
		return rocco.ErrUnauthorized.WithMessage("request has no tenant")
	default:
		return err
	}
}
