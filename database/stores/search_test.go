//go:build testing

package stores

import (
	"context"
	"testing"

	"github.com/zoobz-io/sum"
)

// The search store decodes the job's JSON payload into a DocumentIndex before
// writing. An invalid payload is rejected before any OpenSearch call, so a
// malformed projection never reaches the cluster.
func TestSearch_Index_RejectsInvalidPayload(t *testing.T) {
	sum.Reset()
	sum.New()
	s := NewSearch(nil) // provider is never reached on the decode-error path

	err := s.Index(context.Background(), "doc-1", []byte("not json"))
	if err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
}
