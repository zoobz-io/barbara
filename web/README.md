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

> Scaffold status: the workspace, configs, and SDKs are in place; each app is still a
> single placeholder page that doesn't consume its SDK yet.

## The OpenAPI → SDK pipeline

The clients are typed by generation, not by hand. Per SDK:

```
{api,admin}/handlers ─▶ cmd/{api,admin}spec ─▶ packages/*-sdk/data/openapi.json
                                                    │            (make openapi-{api,admin})
                                        openapi-typescript (pnpm generate)
                                                    ▼
                                      *-sdk/src/schema.ts   (generated — do not edit)
                                                    │
                                    src/client.ts · openapi-press factory
                                                    ▼
                                           apps/{public,admin}
```

Change an endpoint's shape and the chain re-runs from `make openapi-api` /
`make openapi-admin`; typecheck fails if a client names a path or method the
spec no longer has.

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
make web-install    # pnpm install
make web-check      # pnpm run typecheck
make web-lint       # pnpm run lint
make web-test       # pnpm run test
make web-build      # pnpm run build
make openapi-api    # regenerate api-sdk/data/openapi.json from the handlers
make openapi-admin  # regenerate admin-sdk/data/openapi.json from the handlers
```

`make check` folds `web-check`, `web-lint`, and `web-test` into the quick gate;
`make ci` adds `web-build`. The CI workflow runs the same set (plus a Prettier
check) as its own `Web` job.
