# @barbara/api-sdk

The generated, fully-typed client for the Barbara **public API** (`127.0.0.1:8080`).
Consumed by [`apps/public`](../../apps/public/).

`createApiClient(config)` returns a resource-namespaced client
(`client.documents.tags.add(documentId)`) built with
[openapi-press](https://www.npmjs.com/package/openapi-press): methods take
positional path params then a trailing options object, return the response body
directly, and throw from the openapi-press error hierarchy on failure.

## The pipeline

Nothing here is hand-written except `client.ts` (the namespace tree) — the types
all flow from the Go handlers:

```
api/handlers ─▶ cmd/apispec ─▶ data/openapi.json      (make openapi-api)
                                    │
                        openapi-typescript (pnpm generate)
                                    ▼
                      src/schema.ts   (generated — do not edit)
                                    │
                  src/client.ts · createApiClient (openapi-press)
                                    ▼
                               apps/public
```

Change a public endpoint's shape and the whole chain re-runs from
`make openapi-api`. Typecheck fails if `client.ts` names a path or method the
spec no longer has.

## Scripts

| Script           | Does                                          |
| ---------------- | --------------------------------------------- |
| `pnpm generate`  | Render `data/openapi.json` → `src/schema.ts`. |
| `pnpm typecheck` | `tsc --noEmit`.                               |
| `pnpm lint`      | ESLint.                                       |
| `pnpm test`      | Vitest — client assembly smoke tests.         |
