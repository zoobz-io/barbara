# @barbara/web

The TypeScript half of Barbara: two Nuxt apps and the generated clients they stand on.
This is a private pnpm workspace — nothing here publishes. It mirrors the two Go API
surfaces one-to-one, so each app talks to exactly one API through exactly one SDK.

Four members, each with its own README:

| Member                                      | Package              | What it is                                               |
| ------------------------------------------- | -------------------- | -------------------------------------------------------- |
| [`apps/public`](apps/public/)               | `@barbara/public`    | The Nuxt 4 public site over the public API.              |
| [`apps/admin`](apps/admin/)                 | `@barbara/admin`     | The Nuxt 4 admin console over the admin API.             |
| [`packages/api-sdk`](packages/api-sdk/)     | `@barbara/api-sdk`   | The generated typed client for the public API (`:8080`). |
| [`packages/admin-sdk`](packages/admin-sdk/) | `@barbara/admin-sdk` | The generated typed client for the admin API (`:8081`).  |

```
apps/public ── @barbara/api-sdk ────▶ public API  (:8080)
apps/admin ── @barbara/admin-sdk ──▶ admin API   (:8081)
```

> Scaffold status: the workspace, configs, and package shapes are in place; the SDKs
> export nothing yet and each app is a single placeholder page. The OpenAPI → SDK
> pipeline (spec dump → `openapi-typescript` → press client) is wired but waiting on
> spec dumps from the Go surfaces.

## The OpenAPI → SDK pipeline

The clients are not written; they are generated. Per SDK:

```
{api,admin}/handlers ─▶ spec dump ─▶ packages/*-sdk/data/openapi.json
                                          │
                              openapi-typescript (pnpm generate)
                                          ▼
                            *-sdk/src/schema.ts   (generated — do not edit)
                                          │
                          src/client.ts · openapi-press factory
                                          ▼
                                     apps/{public,admin}
```

## Layout

```
web/
├── apps/
│   ├── public/         # @barbara/public — Nuxt site  (dev :3000)
│   └── admin/          # @barbara/admin — Nuxt console (dev :3001)
├── packages/
│   ├── api-sdk/        # @barbara/api-sdk — generated public-API client
│   └── admin-sdk/      # @barbara/admin-sdk — generated admin-API client
├── pnpm-workspace.yaml # globs: packages/* and apps/*
├── tsconfig.base.json  # shared compiler options
└── vitest.config.ts    # root coverage config
```

## Toolchain

- **pnpm 10.27.0** (pinned via `packageManager`), **Node ≥ 22** (`engines`).
- Workspace globs `packages/*` and `apps/*`.

## Scripts

Run from `web/`. Everything with `-r` fans out across all four members.

| Script                         | Does                                             |
| ------------------------------ | ------------------------------------------------ |
| `pnpm build`                   | `pnpm -r run build` — SDKs, then the apps.       |
| `pnpm dev`                     | `pnpm -r --parallel run dev` — both apps.        |
| `pnpm lint`                    | ESLint over the workspace.                       |
| `pnpm format` / `pnpm inspect` | Prettier write / check.                          |
| `pnpm test`                    | `pnpm -r run test` — each member's Vitest suite. |
| `pnpm coverage`                | Root Vitest coverage run.                        |
| `pnpm typecheck`               | `pnpm -r run typecheck` — tsc + vue-tsc.         |

## From the repo root

The [top-level Makefile](../Makefile) drives this workspace so a Go-only checkout needs
no `cd`:

```
make web-install   # pnpm install
make web-check     # pnpm run typecheck
make web-lint      # pnpm run lint
make web-test      # pnpm run test
make web-build     # pnpm run build
```

`make check` folds `web-check`, `web-lint`, and `web-test` into the quick gate;
`make ci` adds `web-build`. The CI workflow runs the same set (plus a Prettier
check) as its own `Web` job.
