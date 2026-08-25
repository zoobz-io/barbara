package boot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"

	osmappings "github.com/zoobz-io/barbara/database/migrations/opensearch"
)

// indexNameRe extracts the index name from a mapping filename like
// "001_documents.json" — the numeric prefix orders application, the capture
// group is the index name.
var indexNameRe = regexp.MustCompile(`^\d+_(.+)\.json$`)

// EnsureIndices creates OpenSearch indices with explicit mappings if they do
// not already exist, reading the embedded mapping files from
// database/migrations/opensearch/. Idempotent: existing indices are left
// untouched. Ready to be invoked by boot once the OpenSearch address is known.
func EnsureIndices(ctx context.Context, addr string) error {
	entries, err := osmappings.Mappings.ReadDir(".")
	if err != nil {
		return fmt.Errorf("reading embedded mappings: %w", err)
	}

	// Sort by filename so indices are created in migration order.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := indexNameRe.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		indexName := matches[1]

		mapping, err := osmappings.Mappings.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("reading mapping %s: %w", entry.Name(), err)
		}

		exists, err := indexExists(ctx, addr, indexName)
		if err != nil {
			return fmt.Errorf("checking index %s: %w", indexName, err)
		}
		if exists {
			log.Printf("index %s already exists, skipping", indexName)
			continue
		}

		if err := createIndex(ctx, addr, indexName, mapping); err != nil {
			return fmt.Errorf("creating index %s: %w", indexName, err)
		}
		log.Printf("index %s created", indexName)
	}

	return nil
}

// indexExists checks whether an OpenSearch index exists (HEAD /{index}).
func indexExists(ctx context.Context, addr, index string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, fmt.Sprintf("%s/%s", addr, index), nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}

// createIndex creates an OpenSearch index with the given mapping (PUT /{index}).
func createIndex(ctx context.Context, addr, index string, mapping []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/%s", addr, index), bytes.NewReader(mapping))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
