//go:build testing

package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	// Register the decoders image.Decode needs for the catalog's formats.
	_ "image/jpeg"
	_ "image/png"
)

// fakeAPI is an in-memory stand-in for the slice of the public API the seeder
// talks to: apps, the collection/document tree, versions with head checking,
// releases that snapshot the tree, and asset uploads. It records what the
// seeder sent and derives document status the way the real store does, so
// the tests can assert the shape the studio will see.
type fakeAPI struct {
	mu          sync.Mutex
	apps        []app
	collections map[string]*fakeCollection
	documents   map[string]*fakeDocument
	releases    map[string][]fakeRelease // by app id
	uploads     map[string]upload        // by app id + key
	created     []string
	nextID      int
	rejectAt    string // asset key whose upload fails, when set
	rejectSave  string // document key whose next version save fails, when set
}

type fakeCollection struct {
	id, appID, name string
	parentID        *string
}

type fakeDocument struct {
	id, appID, name, key string
	collectionID         *string
	tags                 []string
	versions             []string
}

type fakeRelease struct {
	id      string
	number  int
	entries map[string]int // key -> version number
}

type upload struct {
	contentType string
	size        int
}

// The seeder logs every step; keep the test output to the failures.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func newFakeAPI(apps ...app) *fakeAPI {
	return &fakeAPI{
		apps:        apps,
		collections: map[string]*fakeCollection{},
		documents:   map[string]*fakeDocument{},
		releases:    map[string][]fakeRelease{},
		uploads:     map[string]upload{},
	}
}

func (f *fakeAPI) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%d", prefix, f.nextID)
}

func (f *fakeAPI) fail(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Code: code, Message: msg})
}

func (f *fakeAPI) ok(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// pathOf is a collection's slash path from the app root.
func (f *fakeAPI) pathOf(id *string) string {
	if id == nil {
		return ""
	}
	col := f.collections[*id]
	parent := f.pathOf(col.parentID)
	if parent == "" {
		return col.name
	}
	return parent + "/" + col.name
}

// status derives a document's lifecycle status from the app's current release.
func (f *fakeAPI) status(d *fakeDocument) string {
	rels := f.releases[d.appID]
	if len(rels) == 0 {
		return "draft"
	}
	live, ok := rels[len(rels)-1].entries[d.key]
	switch {
	case !ok:
		return "draft"
	case live < len(d.versions):
		return "published-with-newer-draft"
	default:
		return "published"
	}
}

func (f *fakeAPI) docResponse(d *fakeDocument) map[string]any {
	tags := d.tags
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{"id": d.id, "key": d.key, "status": f.status(d), "tags": tags}
}

func (f *fakeAPI) contentsOf(appID string, collectionID *string) map[string]any {
	subs := []map[string]any{}
	for _, c := range f.collections {
		if c.appID == appID && sameID(c.parentID, collectionID) {
			subs = append(subs, map[string]any{"id": c.id, "name": c.name, "parent_id": c.parentID})
		}
	}
	docs := []map[string]any{}
	for _, d := range f.documents {
		if d.appID == appID && sameID(d.collectionID, collectionID) {
			docs = append(docs, f.docResponse(d))
		}
	}
	return map[string]any{"subcollections": subs, "documents": docs}
}

func sameID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	seg := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	at := func(i int) string {
		if i < len(seg) {
			return seg[i]
		}
		return ""
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/apps":
		f.ok(w, 200, map[string]any{"apps": f.apps})
	case r.Method == http.MethodPost && r.URL.Path == "/apps":
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		a := app{ID: "created-" + body.Name, Name: body.Name}
		f.apps = append(f.apps, a)
		f.created = append(f.created, body.Name)
		f.ok(w, 201, a)

	case r.Method == http.MethodGet && at(0) == "apps" && at(2) == "contents":
		f.ok(w, 200, f.contentsOf(at(1), nil))
	case r.Method == http.MethodGet && at(0) == "apps" && at(2) == "collections" && at(4) == "contents":
		id := at(3)
		if _, ok := f.collections[id]; !ok {
			f.fail(w, 404, "NOT_FOUND", "no collection")
			return
		}
		f.ok(w, 200, f.contentsOf(at(1), &id))
	case r.Method == http.MethodPost && at(0) == "apps" && at(2) == "collections" && len(seg) == 3:
		var body struct {
			ParentID *string `json:"parent_id"`
			Name     string  `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, c := range f.collections {
			if c.appID == at(1) && sameID(c.parentID, body.ParentID) && c.name == body.Name {
				f.fail(w, 409, "CONFLICT", "name taken")
				return
			}
		}
		c := &fakeCollection{id: f.id("col"), appID: at(1), name: body.Name, parentID: body.ParentID}
		f.collections[c.id] = c
		f.ok(w, 201, map[string]any{"id": c.id, "name": c.name, "parent_id": c.parentID})
	case r.Method == http.MethodPost && at(0) == "apps" && at(2) == "documents" && len(seg) == 3:
		var body struct {
			CollectionID *string `json:"collection_id"`
			Name         string  `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		key := body.Name
		if p := f.pathOf(body.CollectionID); p != "" {
			key = p + "/" + body.Name
		}
		for _, d := range f.documents {
			if d.appID == at(1) && d.key == key {
				f.fail(w, 409, "CONFLICT", "key taken")
				return
			}
		}
		d := &fakeDocument{id: f.id("doc"), appID: at(1), name: body.Name, key: key, collectionID: body.CollectionID}
		f.documents[d.id] = d
		f.ok(w, 201, f.docResponse(d))

	case r.Method == http.MethodPost && at(0) == "documents" && at(2) == "tags":
		d, ok := f.documents[at(1)]
		if !ok {
			f.fail(w, 404, "NOT_FOUND", "no document")
			return
		}
		var body struct {
			Tag string `json:"tag"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		d.tags = append(d.tags, body.Tag)
		f.ok(w, 200, f.docResponse(d))
	case r.Method == http.MethodGet && at(0) == "documents" && at(2) == "content":
		d, ok := f.documents[at(1)]
		if !ok {
			f.fail(w, 404, "NOT_FOUND", "no document")
			return
		}
		var content any
		if n := len(d.versions); n > 0 {
			content = map[string]any{"version_number": n, "content": d.versions[n-1]}
		}
		f.ok(w, 200, map[string]any{"document": f.docResponse(d), "content": content})
	case r.Method == http.MethodPost && at(0) == "documents" && at(2) == "versions":
		d, ok := f.documents[at(1)]
		if !ok {
			f.fail(w, 404, "NOT_FOUND", "no document")
			return
		}
		if d.key == f.rejectSave {
			f.rejectSave = ""
			f.fail(w, 500, "INTERNAL", "rejected by test")
			return
		}
		var body struct {
			BaseVersion *int   `json:"base_version"`
			Content     string `json:"content"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.BaseVersion == nil || *body.BaseVersion != len(d.versions) {
			f.fail(w, 409, "CONFLICT", "head moved")
			return
		}
		d.versions = append(d.versions, body.Content)
		f.ok(w, 201, map[string]any{"version_number": len(d.versions)})

	case r.Method == http.MethodPost && at(0) == "apps" && at(2) == "releases" && len(seg) == 3:
		entries := map[string]int{}
		for _, d := range f.documents {
			if d.appID == at(1) && len(d.versions) > 0 {
				entries[d.key] = len(d.versions)
			}
		}
		rel := fakeRelease{id: f.id("rel"), number: len(f.releases[at(1)]) + 1, entries: entries}
		f.releases[at(1)] = append(f.releases[at(1)], rel)
		f.ok(w, 201, map[string]any{"id": rel.id, "number": rel.number})

	case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/assets/object"):
		appID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/apps/"), "/assets/object")
		key := r.URL.Query().Get("key")
		if key == f.rejectAt {
			f.fail(w, 400, "BAD_REQUEST", "rejected by test")
			return
		}
		data, _ := io.ReadAll(r.Body)
		f.uploads[appID+"/"+key] = upload{contentType: r.Header.Get("Content-Type"), size: len(data)}
		f.ok(w, 200, storedAsset{Key: key, ContentType: r.Header.Get("Content-Type"), Size: int64(len(data))})
	default:
		f.fail(w, 404, "NOT_FOUND", "no route: "+r.Method+" "+r.URL.Path)
	}
}

// docByKey finds an app's document by key, or nil.
func (f *fakeAPI) docByKey(appID, key string) *fakeDocument {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.documents {
		if d.appID == appID && d.key == key {
			return d
		}
	}
	return nil
}

// expected shape of the site script, kept beside the assertions so a script
// change fails loudly here rather than silently in the studio.
const (
	wantCollections = 4
	wantDocuments   = 11
	wantVersions    = 13
	wantReleases    = 2
)

func TestSeed_BuildsTheSite(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	got := res.site
	if got.collections != wantCollections || got.documents != wantDocuments || got.versions != wantVersions || got.releases != wantReleases {
		t.Errorf("site = %+v, want %d collections, %d documents, %d versions, %d releases",
			got, wantCollections, wantDocuments, wantVersions, wantReleases)
	}

	// The tree the studio will show.
	wantStatus := map[string]string{
		"index.md":                           "published-with-newer-draft",
		"about.md":                           "published",
		"guides/getting-started.md":          "published",
		"guides/advanced/caching.md":         "published",
		"reference/api.md":                   "published",
		"blog/release-notes.md":              "published",
		"guides/advanced/troubleshooting.md": "draft",
		"blog/ideas.md":                      "draft",
	}
	for key, want := range wantStatus {
		d := api.docByKey("app-1", key)
		if d == nil {
			t.Errorf("%s: not created", key)
			continue
		}
		if got := api.status(d); got != want {
			t.Errorf("%s: status %q, want %q", key, got, want)
		}
	}
	if d := api.docByKey("app-1", "guides/getting-started.md"); d != nil && len(d.versions) != 2 {
		t.Errorf("getting-started has %d versions, want 2", len(d.versions))
	}
	if d := api.docByKey("app-1", "blog/ideas.md"); d != nil && len(d.versions) != 0 {
		t.Errorf("ideas has %d versions, want none (an empty document)", len(d.versions))
	}
	if d := api.docByKey("app-1", "guides/getting-started.md"); d != nil && strings.Join(d.tags, ",") != "guide,beginner" {
		t.Errorf("getting-started tags = %v, want [guide beginner]", d.tags)
	}

	// Release 2 carries the page added after release 1; neither carries the
	// work in progress.
	rels := api.releases["app-1"]
	if len(rels) != 2 {
		t.Fatalf("%d releases, want 2", len(rels))
	}
	if _, ok := rels[0].entries["blog/release-notes.md"]; ok {
		t.Error("release 1 carries a page written after it was cut")
	}
	if _, ok := rels[1].entries["blog/release-notes.md"]; !ok {
		t.Error("release 2 is missing blog/release-notes.md")
	}
	if _, ok := rels[1].entries["guides/advanced/troubleshooting.md"]; ok {
		t.Error("release 2 carries the uncut draft")
	}
	if rels[1].entries["index.md"] != 1 {
		t.Errorf("release 2 serves index.md v%d, want v1 (v2 is the draft on top)", rels[1].entries["index.md"])
	}
}

func TestSeed_ReplayIsANoOp(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	if _, err := seed(context.Background(), options{api: srv.URL}); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	res, err := seed(context.Background(), options{api: srv.URL})
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if res.site != (siteResult{}) {
		t.Errorf("replay created %+v, want nothing", res.site)
	}
	if n := len(api.releases["app-1"]); n != wantReleases {
		t.Errorf("%d releases after replay, want %d", n, wantReleases)
	}
	total := 0
	for _, d := range api.documents {
		total += len(d.versions)
	}
	if total != wantVersions {
		t.Errorf("%d versions after replay, want %d", total, wantVersions)
	}
}

func TestSeed_ResumesAfterAFailedStep(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	api.rejectSave = "reference/api.md" // the first save to this page fails
	srv := httptest.NewServer(api)
	defer srv.Close()

	if _, err := seed(context.Background(), options{api: srv.URL}); err == nil {
		t.Fatal("first seed succeeded, want the rejected save")
	}
	if n := len(api.releases["app-1"]); n != 0 {
		t.Fatalf("%d releases cut before the failure, want none", n)
	}

	res, err := seed(context.Background(), options{api: srv.URL})
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if res.site.releases != wantReleases {
		t.Errorf("resumed run cut %d releases, want %d", res.site.releases, wantReleases)
	}
	total := 0
	for _, d := range api.documents {
		total += len(d.versions)
	}
	if total != wantVersions {
		t.Errorf("%d versions after resume, want %d (no duplicates)", total, wantVersions)
	}
	if d := api.docByKey("app-1", "reference/api.md"); d == nil || len(d.versions) != 2 {
		t.Errorf("reference/api.md did not reach its two versions after resume")
	}
}

func TestSite_ScriptIsWellFormed(t *testing.T) {
	folders := map[string]bool{"": true}
	writes := 0
	lastKind := stepCut
	for i, s := range site() {
		switch s.kind {
		case stepFolder:
			if folders[s.path] {
				t.Errorf("step %d: folder %q declared twice", i, s.path)
			}
			parent, _ := splitPath(s.path)
			if !folders[parent] {
				t.Errorf("step %d: folder %q declared before its parent", i, s.path)
			}
			folders[s.path] = true
		case stepWrite:
			writes++
			parent, name := splitPath(s.path)
			if !folders[parent] {
				t.Errorf("step %d: write %q into an undeclared folder", i, s.path)
			}
			if !strings.HasSuffix(name, ".md") {
				t.Errorf("step %d: %q is not a markdown page", i, s.path)
			}
			if s.content != "" && !strings.HasSuffix(s.content, "\n") {
				t.Errorf("step %d: %q content does not end in a newline", i, s.path)
			}
		case stepCut:
			if lastKind == stepCut {
				t.Errorf("step %d: cut with nothing written since the last cut", i)
			}
		}
		lastKind = s.kind
	}
	if writes == 0 {
		t.Fatal("script writes nothing")
	}
	if lastKind == stepCut {
		t.Error("script ends on a cut: nothing is left in progress for the studio to show")
	}
}

func TestSeed_UploadsEveryCatalogEntryToNamedApp(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"}, app{ID: "app-2", Name: "other"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL, app: "other"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	entries := catalog()
	if res.uploaded != len(entries) {
		t.Errorf("uploaded = %d, want %d", res.uploaded, len(entries))
	}
	if res.app.ID != "app-2" {
		t.Errorf("target app = %q, want app-2 (matched by name)", res.app.ID)
	}
	if len(api.created) != 0 {
		t.Errorf("created apps %v, want none", api.created)
	}
	for _, entry := range entries {
		got, ok := api.uploads["app-2/"+entry.Key]
		if !ok {
			t.Errorf("%s: not uploaded", entry.Key)
			continue
		}
		if got.contentType != entry.ContentType {
			t.Errorf("%s: content type %q, want %q", entry.Key, got.contentType, entry.ContentType)
		}
		if got.size != len(entry.Data) {
			t.Errorf("%s: size %d, want %d", entry.Key, got.size, len(entry.Data))
		}
	}
}

func TestSeed_MatchesAppByID(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"}, app{ID: "app-2", Name: "other"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL, app: "app-2"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res.app.Name != "other" {
		t.Errorf("target app = %q, want other", res.app.Name)
	}
}

func TestSeed_DefaultsToFirstApp(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"}, app{ID: "app-2", Name: "other"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res.app.ID != "app-1" {
		t.Errorf("target app = %q, want the first app", res.app.ID)
	}
}

func TestSeed_CreatesUnknownApp(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL, app: "fresh"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if len(api.created) != 1 || api.created[0] != "fresh" {
		t.Fatalf("created apps = %v, want [fresh]", api.created)
	}
	if res.app.ID != "created-fresh" {
		t.Errorf("target app = %q, want the created app", res.app.ID)
	}
	if _, ok := api.uploads["created-fresh/images/logo.png"]; !ok {
		t.Error("assets were not uploaded to the created app")
	}
}

func TestSeed_CreatesDefaultAppWhenTenantHasNone(t *testing.T) {
	api := newFakeAPI()
	srv := httptest.NewServer(api)
	defer srv.Close()

	res, err := seed(context.Background(), options{api: srv.URL})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res.app.Name != defaultAppName {
		t.Errorf("target app = %q, want %q", res.app.Name, defaultAppName)
	}
}

func TestSeed_SurfacesAPIErrors(t *testing.T) {
	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	api.rejectAt = "images/hero.jpg"
	srv := httptest.NewServer(api)
	defer srv.Close()

	_, err := seed(context.Background(), options{api: srv.URL})
	if err == nil {
		t.Fatal("seed succeeded, want the upload rejection")
	}
	for _, want := range []string{"images/hero.jpg", "rejected by test", "BAD_REQUEST"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestSeed_UnreachableAPI(t *testing.T) {
	srv := httptest.NewServer(newFakeAPI())
	srv.Close() // closed before use: the address refuses connections

	if _, err := seed(context.Background(), options{api: srv.URL}); err == nil {
		t.Fatal("seed succeeded against a closed server")
	}
}

func TestCatalog_KeysAreUniqueAndWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range catalog() {
		if seen[entry.Key] {
			t.Errorf("duplicate key %q", entry.Key)
		}
		seen[entry.Key] = true
		if strings.HasPrefix(entry.Key, "/") || strings.HasSuffix(entry.Key, "/") || strings.Contains(entry.Key, "//") {
			t.Errorf("key %q is not a clean relative path", entry.Key)
		}
		if entry.ContentType == "" {
			t.Errorf("%s: empty content type", entry.Key)
		}
		if len(entry.Data) == 0 {
			t.Errorf("%s: empty data", entry.Key)
		}
	}
}

// The studio's browser projects folders from key prefixes; the catalog should
// give it root files, several root folders, and a nested folder to navigate.
func TestCatalog_ShapesATree(t *testing.T) {
	rootFiles, rootFolders := 0, map[string]bool{}
	deepest := 0
	for _, entry := range catalog() {
		parts := strings.Split(entry.Key, "/")
		if len(parts) == 1 {
			rootFiles++
		} else {
			rootFolders[parts[0]] = true
		}
		if len(parts) > deepest {
			deepest = len(parts)
		}
	}
	if rootFiles == 0 {
		t.Error("catalog has no root files")
	}
	if len(rootFolders) < 3 {
		t.Errorf("catalog has %d root folders, want at least 3", len(rootFolders))
	}
	if deepest < 4 {
		t.Errorf("deepest key has %d segments, want a folder at least three levels deep", deepest)
	}
}

func TestCatalog_ImagesDecodeAsDeclared(t *testing.T) {
	formats := map[string]string{typePNG: "png", typeJPEG: "jpeg"}
	checked := 0
	for _, entry := range catalog() {
		want, ok := formats[entry.ContentType]
		if !ok {
			continue
		}
		checked++
		img, format, err := image.Decode(bytes.NewReader(entry.Data))
		if err != nil {
			t.Errorf("%s: decode: %v", entry.Key, err)
			continue
		}
		if format != want {
			t.Errorf("%s: decoded as %s, declared %s", entry.Key, format, entry.ContentType)
		}
		if b := img.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
			t.Errorf("%s: empty bounds %v", entry.Key, b)
		}
	}
	if checked == 0 {
		t.Fatal("no raster images in the catalog")
	}
}

func TestCatalog_DataFilesParse(t *testing.T) {
	for _, entry := range catalog() {
		switch entry.ContentType {
		case typeJSON:
			if !json.Valid(entry.Data) {
				t.Errorf("%s: invalid JSON", entry.Key)
			}
		case typeCSV:
			rows, err := csv.NewReader(bytes.NewReader(entry.Data)).ReadAll()
			if err != nil {
				t.Errorf("%s: %v", entry.Key, err)
			} else if len(rows) < 2 {
				t.Errorf("%s: %d rows, want a header plus data", entry.Key, len(rows))
			}
		case typePDF:
			if !bytes.HasPrefix(entry.Data, []byte("%PDF-")) || !bytes.HasSuffix(entry.Data, []byte("%%EOF\n")) {
				t.Errorf("%s: missing PDF header or trailer", entry.Key)
			}
		case typeSVG:
			if !bytes.HasPrefix(entry.Data, []byte("<svg ")) {
				t.Errorf("%s: not an SVG document", entry.Key)
			}
		}
	}
}

// The xref table must point at the real byte offset of every object, or strict
// readers refuse the file.
func TestPDFDocument_XrefOffsetsAreExact(t *testing.T) {
	data := pdfDocument("Test", 2)
	// "startxref" also ends in "xref", so anchor on the preceding newline.
	xrefAt := bytes.LastIndex(data, []byte("\nxref\n"))
	if xrefAt < 0 {
		t.Fatal("no xref table")
	}
	xrefAt++
	if !strings.Contains(string(data[xrefAt:]), "startxref\n"+strconv.Itoa(xrefAt)+"\n") {
		t.Error("startxref does not point at the xref table")
	}
	// After "xref", "0 N", and the free entry, each line is "<offset> 00000 n ".
	lines := strings.Split(string(data[xrefAt:]), "\n")[3:]
	objects := 0
	for _, line := range lines {
		var off int
		if _, err := fmt.Sscanf(line, "%d 00000 n", &off); err != nil {
			break
		}
		objects++
		want := fmt.Sprintf("%d 0 obj\n", objects)
		if !bytes.HasPrefix(data[off:], []byte(want)) {
			t.Errorf("object %d: offset %d points at %q", objects, off, string(data[off:min(off+12, len(data))]))
		}
	}
	// 1 catalog + 1 pages + 1 font + (page + content) x 2.
	if objects != 7 {
		t.Errorf("xref lists %d objects, want 7", objects)
	}
}

// run parses the flags, seeds, and reports; an unknown flag is an error.
func TestRun(t *testing.T) {
	srv := httptest.NewServer(newFakeAPI(app{ID: "app-1", Name: "docs-site"}))
	defer srv.Close()

	if err := run([]string{"-api", srv.URL, "-app", "docs-site"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := run([]string{"-not-a-flag"}); err == nil {
		t.Error("run accepted an unknown flag")
	}
}

// faultyAPI fronts a fakeAPI and breaks one route: a request that matches
// fail gets the configured status and body; everything else passes through.
type faultyAPI struct {
	inner  http.Handler
	fail   func(r *http.Request) bool
	body   string
	status int
}

func (f *faultyAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.fail(r) {
		w.WriteHeader(f.status)
		_, _ = io.WriteString(w, f.body)
		return
	}
	f.inner.ServeHTTP(w, r)
}

func route(method, suffix string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return r.Method == method && strings.HasSuffix(r.URL.Path, suffix)
	}
}

// Every API call the seeder makes surfaces a failure with the step it was
// on, carrying the API's envelope when it sent one, the bare status when it
// did not, and a decode failure when the body is not the JSON expected. An
// empty success body is not an error.
func TestSeed_FailsAtEachStep(t *testing.T) {
	const envelope = `{"code":"TEAPOT","message":"short and stout"}`
	cases := []struct {
		name   string
		fail   func(*http.Request) bool
		body   string
		want   []string // substrings of the error; empty means success
		status int
	}{
		{name: "listing apps", fail: route(http.MethodGet, "/apps"), status: 500, body: envelope,
			want: []string{"listing apps", "short and stout", "TEAPOT"}},
		{name: "status without envelope", fail: route(http.MethodGet, "/apps"), status: 502, body: "bad gateway",
			want: []string{"unexpected status 502"}},
		{name: "undecodable body", fail: route(http.MethodGet, "/apps"), status: 200, body: "{not json",
			want: []string{"decoding response"}},
		{name: "listing the root", status: 500, body: envelope,
			fail: func(r *http.Request) bool { return r.Method == http.MethodGet && r.URL.Path == "/apps/app-1/contents" },
			want: []string{`listing "/"`}},
		{name: "listing a folder", status: 500, body: envelope,
			fail: func(r *http.Request) bool {
				return r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/collections/") && strings.HasSuffix(r.URL.Path, "/contents")
			},
			want: []string{`listing "guides"`}},
		{name: "creating a folder", fail: route(http.MethodPost, "/collections"), status: 500, body: envelope,
			want: []string{`creating folder "guides"`}},
		{name: "creating a document", fail: route(http.MethodPost, "/documents"), status: 500, body: envelope,
			want: []string{`creating document "index.md"`}},
		{name: "tagging", fail: route(http.MethodPost, "/tags"), status: 500, body: envelope,
			want: []string{`tagging "index.md"`}},
		{name: "saving a version", fail: route(http.MethodPost, "/versions"), status: 500, body: envelope,
			want: []string{`saving "index.md"`}},
		{name: "cutting a release", fail: route(http.MethodPost, "/releases"), status: 500, body: envelope,
			want: []string{"cutting release"}},
		{name: "uploading an asset", fail: route(http.MethodPut, "/assets/object"), status: 500, body: envelope,
			want: []string{"uploading", "TEAPOT"}},
		{name: "empty success body", fail: route(http.MethodPost, "/tags"), status: 200, body: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
			srv := httptest.NewServer(&faultyAPI{inner: api, fail: tc.fail, status: tc.status, body: tc.body})
			defer srv.Close()

			_, err := seed(context.Background(), options{api: srv.URL})
			if len(tc.want) == 0 {
				if err != nil {
					t.Fatalf("seed: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("seed succeeded, want a failure")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

// Creating the default app can fail too, and a replay that cannot read a
// document's head reports which one.
func TestSeed_FailsResolvingAndReplaying(t *testing.T) {
	const envelope = `{"code":"TEAPOT","message":"short and stout"}`

	srv := httptest.NewServer(&faultyAPI{inner: newFakeAPI(), fail: route(http.MethodPost, "/apps"), status: 500, body: envelope})
	defer srv.Close()
	_, err := seed(context.Background(), options{api: srv.URL})
	if err == nil || !strings.Contains(err.Error(), `creating app "seed-site"`) {
		t.Errorf("create-app failure = %v", err)
	}

	api := newFakeAPI(app{ID: "app-1", Name: "docs-site"})
	good := httptest.NewServer(api)
	defer good.Close()
	if _, err := seed(context.Background(), options{api: good.URL}); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	replay := httptest.NewServer(&faultyAPI{inner: api, fail: route(http.MethodGet, "/content"), status: 500, body: envelope})
	defer replay.Close()
	_, err = seed(context.Background(), options{api: replay.URL})
	if err == nil || !strings.Contains(err.Error(), `reading "index.md"`) {
		t.Errorf("replay failure = %v", err)
	}
}
