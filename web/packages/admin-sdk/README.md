# @barbara/admin-sdk

The generated, fully-typed client for the Barbara **admin API** (`127.0.0.1:8081`).
Consumed by [`apps/admin`](../../apps/admin/). Scaffold only — no client exists yet.

## The pipeline

Nothing here is hand-written once the pipeline runs:

```
admin/handlers ─▶ spec dump ─▶ data/openapi.json
                                    │
                        openapi-typescript (pnpm generate)
                                    ▼
                      src/schema.ts   (generated — do not edit)
                                    │
                  src/client.ts · createAdminClient (openapi-press)
                                    ▼
                               apps/admin
```

## Scripts

| Script           | Does                                             |
| ---------------- | ------------------------------------------------ |
| `pnpm generate`  | Render `data/openapi.json` → `src/schema.ts`.    |
| `pnpm typecheck` | `tsc --noEmit`.                                  |
| `pnpm lint`      | ESLint.                                          |
| `pnpm test`      | Vitest (passes with no tests while scaffolding). |
