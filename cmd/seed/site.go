package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// The site script. A seed run replays these steps against the app in order,
// the way an author would build a site: make folders, write pages, cut a
// release, keep writing. Every step is an "ensure" — it looks before it acts —
// so replaying the script against an already-seeded app changes nothing, and
// a run that died halfway picks up where it left off.
//
// Steps come in three kinds:
//
//   - folder: a collection exists at the path.
//   - write: a document exists at the path and holds this content as its
//     next version. Repeating a path in the script appends a version, which is
//     how the seed produces edit history and drafts over a published page.
//   - cut: a release snapshots everything written so far. A cut is skipped
//     when nothing was written since the previous cut in this run, so a
//     replay never stacks identical releases.
//
// Read top to bottom, the script yields two releases, a page edited after the
// second (published, with a newer draft), a page never released (a draft),
// and a page with no content at all (empty).
type step struct {
	path    string   // collection path for folder; document path for write
	content string   // write only
	label   string   // cut only: the release's label
	tags    []string // write only, applied when the document is created
	kind    stepKind
}

type stepKind int

const (
	stepFolder stepKind = iota
	stepWrite
	stepCut
)

func folder(path string) step { return step{kind: stepFolder, path: path} }

func write(path, content string, tags ...string) step {
	return step{kind: stepWrite, path: path, content: content, tags: tags}
}

func cut(label string) step { return step{kind: stepCut, label: label} }

// site is the script for the seed app: a small product documentation site.
func site() []step {
	return []step{
		// Release 1: the initial site.
		folder("guides"),
		folder("guides/advanced"),
		folder("reference"),
		folder("blog"),
		write("index.md", indexV1, "landing"),
		write("about.md", aboutV1),
		write("guides/getting-started.md", gettingStartedV1, "guide", "beginner"),
		write("guides/installation.md", installationV1, "guide"),
		write("guides/advanced/caching.md", cachingV1, "guide", "advanced"),
		write("reference/api.md", apiV1, "reference"),
		write("reference/cli.md", cliV1, "reference"),
		write("blog/hello-world.md", helloWorldV1, "blog"),
		cut("Initial site"),

		// Release 2: two pages revised, one page added.
		write("guides/getting-started.md", gettingStartedV2),
		write("reference/api.md", apiV2),
		write("blog/release-notes.md", releaseNotesV1, "blog", "changelog"),
		cut("2.3 docs: revised guides, release notes"),

		// Work in progress after the second release, never cut: the landing
		// page has a newer draft over its published version, a new guide is
		// only a draft, and a blog post exists with nothing written yet.
		write("index.md", indexV2),
		write("guides/advanced/troubleshooting.md", troubleshootingV1, "guide", "advanced"),
		write("blog/ideas.md", ""),
	}
}

// siteResult counts what a site run created.
type siteResult struct {
	collections int
	documents   int
	versions    int
	releases    int
}

// seedSite replays the site script against the app.
func seedSite(ctx context.Context, c *client, appID string, steps []step) (siteResult, error) {
	t := &tree{c: c, appID: appID, folders: map[string]string{"": ""}, contents: map[string]*contents{}}
	var res siteResult
	dirty := false // written since the last cut
	for _, s := range steps {
		switch s.kind {
		case stepFolder:
			created, err := t.ensureFolder(ctx, s.path)
			if err != nil {
				return res, err
			}
			if created {
				res.collections++
				log.Printf("  folder   %s/", s.path)
			}
		case stepWrite:
			docCreated, versionSaved, err := t.write(ctx, s)
			if err != nil {
				return res, err
			}
			if docCreated {
				res.documents++
			}
			if versionSaved {
				res.versions++
				dirty = true
			}
			if docCreated || versionSaved {
				log.Printf("  write    %s", s.path)
			}
		case stepCut:
			if !dirty {
				continue
			}
			rel, err := c.cutRelease(ctx, appID, s.label)
			if err != nil {
				return res, fmt.Errorf("cutting release: %w", err)
			}
			res.releases++
			dirty = false
			log.Printf("  release  #%d", rel.Number)
		}
	}
	return res, nil
}

// tree resolves script paths against the app's real tree, caching what it has
// already looked up so the script's repeated paths cost one listing each.
type tree struct {
	c *client
	// folders maps a collection path to its id; "" is the app root, whose id
	// is "" (a nil parent on the wire).
	folders map[string]string
	// contents caches a folder's listing by path.
	contents map[string]*contents
	appID    string
}

// listing returns a folder's contents, fetching on first use.
func (t *tree) listing(ctx context.Context, path string) (*contents, error) {
	if cached, ok := t.contents[path]; ok {
		return cached, nil
	}
	id, ok := t.folders[path]
	if !ok {
		return nil, fmt.Errorf("folder %q not resolved", path)
	}
	got, err := t.c.listContents(ctx, t.appID, optionalID(id))
	if err != nil {
		return nil, fmt.Errorf("listing %q: %w", displayPath(path), err)
	}
	t.contents[path] = &got
	return &got, nil
}

// ensureFolder resolves a collection path, creating any missing segment.
// Reports whether it created the leaf.
func (t *tree) ensureFolder(ctx context.Context, path string) (bool, error) {
	if _, ok := t.folders[path]; ok {
		return false, nil
	}
	parent, name := splitPath(path)
	if _, err := t.ensureFolder(ctx, parent); err != nil {
		return false, err
	}
	parentContents, err := t.listing(ctx, parent)
	if err != nil {
		return false, err
	}
	for _, col := range parentContents.Subcollections {
		if col.Name == name {
			t.folders[path] = col.ID
			return false, nil
		}
	}
	created, err := t.c.createCollection(ctx, t.appID, optionalID(t.folders[parent]), name)
	if err != nil {
		return false, fmt.Errorf("creating folder %q: %w", path, err)
	}
	t.folders[path] = created.ID
	parentContents.Subcollections = append(parentContents.Subcollections, created)
	return true, nil
}

// write ensures the document exists and that the step's content is a version
// it holds. Versions are counted per path across the run: the n-th write to a
// path is version n, and it is saved only when the document's head is still
// below n. An empty write creates the document and leaves it without content.
// Reports whether the document was created and whether a version was saved.
func (t *tree) write(ctx context.Context, s step) (docCreated, versionSaved bool, err error) {
	parent, name := splitPath(s.path)
	if _, err = t.ensureFolder(ctx, parent); err != nil {
		return false, false, err
	}
	parentContents, err := t.listing(ctx, parent)
	if err != nil {
		return false, false, err
	}

	doc := parentContents.document(s.path)
	if doc == nil {
		created, cerr := t.c.createDocument(ctx, t.appID, optionalID(t.folders[parent]), name)
		if cerr != nil {
			return false, false, fmt.Errorf("creating document %q: %w", s.path, cerr)
		}
		for _, tag := range s.tags {
			if _, terr := t.c.addTag(ctx, created.ID, tag); terr != nil {
				return false, false, fmt.Errorf("tagging %q: %w", s.path, terr)
			}
		}
		created.head, created.headKnown = 0, true // just created: no versions
		parentContents.Documents = append(parentContents.Documents, created)
		doc = &parentContents.Documents[len(parentContents.Documents)-1]
		docCreated = true
	}
	if s.content == "" {
		return docCreated, false, nil
	}

	if !doc.headKnown {
		head, herr := t.c.headVersion(ctx, doc.ID)
		if herr != nil {
			return docCreated, false, fmt.Errorf("reading %q: %w", s.path, herr)
		}
		doc.head, doc.headKnown = head, true
	}
	doc.writes++
	if doc.head >= doc.writes {
		return docCreated, false, nil
	}
	if _, verr := t.c.saveVersion(ctx, doc.ID, s.content, doc.head); verr != nil {
		return docCreated, false, fmt.Errorf("saving %q: %w", s.path, verr)
	}
	doc.head++
	return docCreated, true, nil
}

// splitPath separates a path into its parent folder and last segment.
func splitPath(path string) (parent, name string) {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i], path[i+1:]
	}
	return "", path
}

// displayPath names a folder path in messages, with "" as the root.
func displayPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

// optionalID turns the root's empty id into the nil the wire expects.
func optionalID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
