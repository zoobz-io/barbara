# seed

Local-development seeder entrypoint.

## Purpose

A one-shot command that loads mock data into a **running** public API so the
studio has something to show. It builds one app the way an author would:

- **Collections** — `guides/` (with `advanced/` nested), `reference/`, and
  `blog/`.
- **Documents** — eleven markdown pages for a fictional product, with edit
  history: some pages carry two versions, one is a draft over a published
  version, one is a draft that was never released, and one is empty.
- **Releases** — two cut releases, so the history view has entries and the
  second release differs from the first.
- **Assets** — a tree of images, PDFs, data files, and stylesheets under the
  same app. Every file is genuine for its type: the images decode, the PDFs
  open, the CSV and JSON parse.

It goes **through the API, not the database**. Using the same endpoints the
studio uses keeps the seed honest about what the surface accepts, and lets
the stores enforce the release semantics a SQL insert would bypass.

## Behavior

- Resolves the target app against `GET /apps` on the dev tenant (the stub
  authenticator resolves every request to it, so no credentials are needed):
  - `-app <id|name>` matches an existing app by id or name; an unmatched name
    is created.
  - No `-app` takes the tenant's first app, creating `seed-site` when there
    are none.
- Replays the site script (`site.go`) against the app. Every step looks
  before it acts: a folder or page that already exists is reused, a version
  is saved only when the page's head is still behind the script, and a
  release is cut only when something was written since the previous cut. So
  a re-run against a seeded app changes nothing, and a run that failed
  halfway resumes where it stopped.
- Uploads every catalog entry (`catalog.go`) with
  `PUT /apps/{id}/assets/object?key=…`, declaring the real content type.
  Re-running overwrites the same keys.
- Stops at the first failed call and reports the API's error envelope.

## Run

```sh
make dev-api                  # the API, Postgres, and MinIO must be up
make seed                     # first app on the dev tenant
make seed APP=docs-site       # by name (created if missing)
make seed APP=<uuid>          # by id
make seed API=http://localhost:8080
```

Or directly: `go run ./cmd/seed -app docs-site -api http://localhost:8080`.
Then open `/apps/<id>` in the public app.

After `make dev-reset`, run `make dev` and then `make seed` again for a fresh
database with the same content.

## Test

```sh
go test -tags testing ./cmd/seed
```

The tests run the seeder against an in-process fake of the API that models
the tree, version head checking, and release snapshots; no dev stack is
needed.
