package boot

import (
	"context"
	"fmt"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/zoobz-io/grub"
	grubopensearch "github.com/zoobz-io/grub/opensearch"
	osrenderer "github.com/zoobz-io/lucene/opensearch"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/config"
)

// OpenSearch creates a typed OpenSearch search provider from config, through
// the house stack: the opensearch-go client wrapped by grub/opensearch, with
// the lucene renderer for typed queries. The published-document projection is
// written through this provider and the site-facing surface reads through it —
// no raw OpenSearch clients, no hand-written queries.
//
// Callers own lifecycle. Wired into boot.Init once the runtime spine lands.
func OpenSearch(ctx context.Context) (grub.SearchProvider, error) {
	cfg := sum.MustUse[config.OpenSearch](ctx)
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{cfg.Addr},
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("creating opensearch client: %w", err)
	}
	provider := grubopensearch.New(client, grubopensearch.Config{
		Version: osrenderer.V2,
	})
	return provider, nil
}
