# internal/openapispec

Dumps a surface's OpenAPI spec to disk. No server, no database — this feeds the
SDK client generators in the [web monorepo](../../web/). Barbara has two
surfaces, so the dump-and-patch logic lives here once and
[`cmd/apispec`](../../cmd/apispec/) / [`cmd/adminspec`](../../cmd/adminspec/)
stay thin.

## What `Dump` does

1. **Builds a bare engine** — `rocco.NewEngine()` carries no DB, DI, or auth.
   Endpoint registration records handler metadata only (paths, request/response
   types, error defs), and each handler resolves its dependencies lazily at
   request time — so `GenerateOpenAPI` needs no runtime boot.
2. **Applies the surface** — the caller's `ConfigureOpenAPI` (Info, tags, tag
   groups) and endpoint set: the exact metadata and endpoints the surface's
   server serves.
3. **Generates and patches** — `e.GenerateOpenAPI(nil)`, then `patch` backfills
   one component schema, `ValidationFieldError`, that rocco (through v0.1.23)
   emits a `$ref` to but never adds to `components`.
4. **Writes indented JSON** — to the output path, creating the directory as
   needed.
