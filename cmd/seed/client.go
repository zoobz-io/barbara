package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// client is the thin HTTP client the seeder needs: the public-API calls it
// makes, typed against the slices of the wire shapes it reads. It carries no
// auth — the dev stub resolves every request to the dev tenant.
type client struct {
	http *http.Client
	base string
}

func newClient(base string) *client {
	return &client{
		base: strings.TrimRight(base, "/"),
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// app is the slice of the API's app response the seeder reads.
type app struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// collection is the slice of the API's collection response the seeder reads.
type collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// document is the slice of the API's document response the seeder reads,
// plus the per-run state the site script keeps beside it: how many writes the
// script has made to it so far, and its head version number once known.
type document struct {
	ID     string   `json:"id"`
	Key    string   `json:"key"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`

	writes    int
	head      int
	headKnown bool
}

// contents is a folder listing: the app root's or a collection's direct
// subcollections and documents.
type contents struct {
	Subcollections []collection `json:"subcollections"`
	Documents      []document   `json:"documents"`
}

// document finds a listed document by key, or nil.
func (c *contents) document(key string) *document {
	for i := range c.Documents {
		if c.Documents[i].Key == key {
			return &c.Documents[i]
		}
	}
	return nil
}

// release is the slice of the API's release response the seeder reads.
type release struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
}

// storedAsset is the metadata the upload endpoint returns.
type storedAsset struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// apiError is the API's error envelope, surfaced when a call fails.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *client) listApps(ctx context.Context) ([]app, error) {
	var out struct {
		Apps []app `json:"apps"`
	}
	if err := c.do(ctx, http.MethodGet, "/apps", "", nil, &out); err != nil {
		return nil, err
	}
	return out.Apps, nil
}

func (c *client) createApp(ctx context.Context, name string) (app, error) {
	body, err := json.Marshal(struct {
		Name string `json:"name"`
	}{Name: name})
	if err != nil {
		return app{}, fmt.Errorf("encoding app: %w", err)
	}
	var out app
	if err := c.do(ctx, http.MethodPost, "/apps", "application/json", body, &out); err != nil {
		return app{}, fmt.Errorf("creating app %q: %w", name, err)
	}
	return out, nil
}

func (c *client) listContents(ctx context.Context, appID string, collectionID *string) (contents, error) {
	path := "/apps/" + url.PathEscape(appID) + "/contents"
	if collectionID != nil {
		path = "/apps/" + url.PathEscape(appID) + "/collections/" + url.PathEscape(*collectionID) + "/contents"
	}
	var out contents
	if err := c.do(ctx, http.MethodGet, path, "", nil, &out); err != nil {
		return contents{}, err
	}
	return out, nil
}

func (c *client) createCollection(ctx context.Context, appID string, parentID *string, name string) (collection, error) {
	body, err := json.Marshal(struct {
		ParentID *string `json:"parent_id,omitempty"`
		Name     string  `json:"name"`
	}{ParentID: parentID, Name: name})
	if err != nil {
		return collection{}, fmt.Errorf("encoding collection: %w", err)
	}
	var out collection
	if err := c.do(ctx, http.MethodPost, "/apps/"+url.PathEscape(appID)+"/collections", "application/json", body, &out); err != nil {
		return collection{}, err
	}
	return out, nil
}

func (c *client) createDocument(ctx context.Context, appID string, collectionID *string, name string) (document, error) {
	body, err := json.Marshal(struct {
		CollectionID *string `json:"collection_id,omitempty"`
		Name         string  `json:"name"`
	}{CollectionID: collectionID, Name: name})
	if err != nil {
		return document{}, fmt.Errorf("encoding document: %w", err)
	}
	var out document
	if err := c.do(ctx, http.MethodPost, "/apps/"+url.PathEscape(appID)+"/documents", "application/json", body, &out); err != nil {
		return document{}, err
	}
	return out, nil
}

func (c *client) addTag(ctx context.Context, documentID, tag string) (document, error) {
	body, err := json.Marshal(struct {
		Tag string `json:"tag"`
	}{Tag: tag})
	if err != nil {
		return document{}, fmt.Errorf("encoding tag: %w", err)
	}
	var out document
	if err := c.do(ctx, http.MethodPost, "/documents/"+url.PathEscape(documentID)+"/tags", "application/json", body, &out); err != nil {
		return document{}, err
	}
	return out, nil
}

// headVersion returns the document's head version number, 0 when it has no
// versions yet.
func (c *client) headVersion(ctx context.Context, documentID string) (int, error) {
	var out struct {
		Content *struct {
			VersionNumber int `json:"version_number"`
		} `json:"content"`
	}
	if err := c.do(ctx, http.MethodGet, "/documents/"+url.PathEscape(documentID)+"/content", "", nil, &out); err != nil {
		return 0, err
	}
	if out.Content == nil {
		return 0, nil
	}
	return out.Content.VersionNumber, nil
}

// saveVersion appends a version on top of baseVersion, the head the write is
// based on (0 for the first).
func (c *client) saveVersion(ctx context.Context, documentID, content string, baseVersion int) (int, error) {
	body, err := json.Marshal(struct {
		Content     string `json:"content"`
		BaseVersion int    `json:"base_version"`
	}{Content: content, BaseVersion: baseVersion})
	if err != nil {
		return 0, fmt.Errorf("encoding version: %w", err)
	}
	var out struct {
		VersionNumber int `json:"version_number"`
	}
	if err := c.do(ctx, http.MethodPost, "/documents/"+url.PathEscape(documentID)+"/versions", "application/json", body, &out); err != nil {
		return 0, err
	}
	return out.VersionNumber, nil
}

func (c *client) cutRelease(ctx context.Context, appID string) (release, error) {
	var out release
	if err := c.do(ctx, http.MethodPost, "/apps/"+url.PathEscape(appID)+"/releases", "", nil, &out); err != nil {
		return release{}, err
	}
	return out, nil
}

func (c *client) putAsset(ctx context.Context, appID string, a asset) (storedAsset, error) {
	path := "/apps/" + url.PathEscape(appID) + "/assets/object?key=" + url.QueryEscape(a.Key)
	var out storedAsset
	if err := c.do(ctx, http.MethodPut, path, a.ContentType, a.Data, &out); err != nil {
		return storedAsset{}, err
	}
	return out, nil
}

// do issues one request and decodes a JSON success body into out. A non-2xx
// status is an error carrying the API's envelope when it sent one.
func (c *client) do(ctx context.Context, method, path, contentType string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var envelope apiError
		if json.Unmarshal(data, &envelope) == nil && envelope.Code != "" {
			return fmt.Errorf("%s %s: %s (%s)", method, path, envelope.Message, envelope.Code)
		}
		return fmt.Errorf("%s %s: unexpected status %d", method, path, resp.StatusCode)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
