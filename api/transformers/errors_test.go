package transformers

import (
	"errors"
	"testing"

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
		{"no tenant", auth.ErrNoTenant, rocco.ErrUnauthorized},
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
