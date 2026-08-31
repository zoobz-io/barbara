# @barbara/admin-sdk

The generated, fully-typed client for the Barbara **admin API** (`127.0.0.1:8081`).
Consumed by [`apps/admin`](../../apps/admin/).

`createAdminClient(config)` returns a resource-namespaced client
(`client.search.all()`) built with
[openapi-press](https://www.npmjs.com/package/openapi-press): methods take
positional path params then a trailing options object, return the response body
directly, and throw from the openapi-press error hierarchy on failure. The
surface is deliberately small — cross-tenant search is the one capability
seeded here; the platform grows from this.

## The pipeline

Nothing here is hand-written except `client.ts` (the namespace tree) — the types
all flow from the Go handlers:

```
admin/handlers ─▶ cmd/adminspec ─▶ data/openapi.json   (make openapi-admin)
                                       │
                           openapi-typescript (pnpm generate)
                                       ▼
                         src/schema.ts   (generated — do not edit)
                                       │
                   src/client.ts · createAdminClient (openapi-press)
                                       ▼
                                  apps/admin
```

Change an admin endpoint's shape and the whole chain re-runs from
`make openapi-admin`. Typecheck fails if `client.ts` names a path or method the
spec no longer has.

## Scripts

| Script           | Does                                          |
| ---------------- | --------------------------------------------- |
| `pnpm generate`  | Render `data/openapi.json` → `src/schema.ts`. |
| `pnpm typecheck` | `tsc --noEmit`.                               |
| `pnpm lint`      | ESLint.                                       |
| `pnpm test`      | Vitest — client assembly smoke tests.         |
