// Package main is the local-development seeder: it loads mock data into a
// running public API so the studio has something to show. It builds one app
// the way an author would — folders, pages with edit history, two cut
// releases and some work in progress on top — and fills its asset tree with
// images, documents, data files, and stylesheets, all through the same
// endpoints the studio uses.
//
// It is a one-shot command against a running API, not a database script:
// going through the API keeps the seeder honest about what the surface
// accepts, and lets the stores enforce the release semantics a SQL insert
// would bypass. Re-running is idempotent — the site script only creates what
// is missing, and every asset key overwrites its previous bytes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

// options are the command-line inputs.
type options struct {
	// api is the public API base URL.
	api string
	// app names the target app by id or name; empty picks the tenant's first
	// app (creating a default one when there are none), and an unknown name
	// is created.
	app string
}

func run(args []string) error {
	var opts options
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.StringVar(&opts.api, "api", "http://localhost:8080", "public API base URL")
	fs.StringVar(&opts.app, "app", "", "target app by id or name (default: the first app; unknown names are created)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	res, err := seed(context.Background(), opts)
	if err != nil {
		return err
	}
	log.Printf("seed complete: app %q (%s)", res.app.Name, res.app.ID)
	log.Printf("  created %d collections, %d documents, %d versions, %d releases",
		res.site.collections, res.site.documents, res.site.versions, res.site.releases)
	log.Printf("  uploaded %d assets", res.uploaded)
	return nil
}

// result reports what a seed run did.
type result struct {
	app      app
	site     siteResult
	uploaded int
}

// seed resolves the target app, replays the site script into it, and uploads
// every catalog entry. Split from run so the wiring is testable without the
// process exiting.
func seed(ctx context.Context, opts options) (result, error) {
	c := newClient(opts.api)
	target, err := resolveApp(ctx, c, opts.app)
	if err != nil {
		return result{}, err
	}

	log.Printf("seeding site into app %q...", target.Name)
	siteRes, err := seedSite(ctx, c, target.ID, site())
	if err != nil {
		return result{}, err
	}

	log.Println("seeding assets...")
	entries := catalog()
	for _, entry := range entries {
		stored, err := c.putAsset(ctx, target.ID, entry)
		if err != nil {
			return result{}, fmt.Errorf("uploading %q: %w", entry.Key, err)
		}
		log.Printf("  %-40s %-16s %8d B", stored.Key, stored.ContentType, stored.Size)
	}
	return result{app: target, site: siteRes, uploaded: len(entries)}, nil
}

// defaultAppName is the app created when the tenant has none and no target is
// named.
const defaultAppName = "seed-site"

// resolveApp picks the app the seed lands in. A ref matches an existing app by
// id or name; an unmatched non-empty ref is created under that name. An empty
// ref takes the first app, creating the default when the tenant has none.
func resolveApp(ctx context.Context, c *client, ref string) (app, error) {
	apps, err := c.listApps(ctx)
	if err != nil {
		return app{}, fmt.Errorf("listing apps: %w", err)
	}
	if ref == "" {
		if len(apps) > 0 {
			return apps[0], nil
		}
		return c.createApp(ctx, defaultAppName)
	}
	for _, a := range apps {
		if a.ID == ref || a.Name == ref {
			return a, nil
		}
	}
	return c.createApp(ctx, ref)
}
