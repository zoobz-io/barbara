# @barbara/api-sdk

The generated, fully-typed client for the Barbara **public API** (`127.0.0.1:8080`).
Consumed by [`apps/public`](../../apps/public/). Scaffold only — no client exists yet.

## The pipeline

Nothing here is hand-written once the pipeline runs:

```
api/handlers ─▶ spec dump ─▶ data/openapi.json
                                  │
                      openapi-typescript (pnpm generate)
                                  ▼
                    src/schema.ts   (generated — do not edit)
                                  │
                  src/client.ts · createApiClient (openapi-press)
                                  ▼
                             apps/public
```

## Scripts

| Script           | Does                                             |
| ---------------- | ------------------------------------------------ |
| `pnpm generate`  | Render `data/openapi.json` → `src/schema.ts`.    |
| `pnpm typecheck` | `tsc --noEmit`.                                  |
| `pnpm lint`      | ESLint.                                          |
| `pnpm test`      | Vitest (passes with no tests while scaffolding). |
