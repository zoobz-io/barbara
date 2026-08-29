package transformers

import (
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/zoobz-io/rocco"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/auth"
)

func TestErrorToResponse(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want rocco.ErrorDefinition
	}{
		{"not found", stores.ErrNotFound, rocco.ErrNotFound},
		{"published", stores.ErrDocumentPublished, rocco.ErrConflict},
		{"version mismatch", stores.ErrVersionMismatch, rocco.ErrBadRequest},
		{"no tenant", auth.ErrNoTenant, rocco.ErrUnauthorized},
		{"no user", auth.ErrNoUser, rocco.ErrUnauthorized},
		{"duplicate key", &pq.Error{Code: "23505"}, rocco.ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ErrorToResponse(tc.in)
			var def rocco.ErrorDefinition
			if !errors.As(got, &def) || def.Code() != tc.want.Code() {
				t.Fatalf("ErrorToResponse(%v) = %v, want code %s", tc.in, got, tc.want.Code())
			}
		})
	}
}

func TestErrorToResponse_PassesThroughUnknown(t *testing.T) {
	sentinel := errors.New("boom")
	if got := ErrorToResponse(sentinel); !errors.Is(got, sentinel) {
		t.Errorf("unknown error not passed through: got %v", got)
	}
}

func TestErrorToResponse_NonUniquePqError(t *testing.T) {
	// A Postgres error that is not a unique violation falls through to default.
	other := &pq.Error{Code: "23503"} // foreign_key_violation
	if got := ErrorToResponse(other); !errors.Is(got, other) {
		t.Errorf("non-unique pq error not passed through: got %v", got)
	}
}
