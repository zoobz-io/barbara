# @barbara/public

The Nuxt 4 public site over the Barbara **public API** (`127.0.0.1:8080`), typed
end-to-end through [`@barbara/api-sdk`](../../packages/api-sdk/). Scaffold only —
one placeholder page.

Dev server is pinned to **:3000** so it can run alongside [`apps/admin`](../admin/)
(:3001).

## Scripts

| Script           | Does                                       |
| ---------------- | ------------------------------------------ |
| `pnpm dev`       | `nuxi dev` on http://localhost:3000.       |
| `pnpm build`     | `nuxi build`.                              |
| `pnpm typecheck` | `nuxi typecheck` (vue-tsc).                |
