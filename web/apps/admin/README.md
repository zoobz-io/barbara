# @barbara/admin

The Nuxt 4 admin console over the Barbara **admin API** (`127.0.0.1:8081`), typed
end-to-end through [`@barbara/admin-sdk`](../../packages/admin-sdk/). Scaffold only —
one placeholder page.

Dev server is pinned to **:3001** so it can run alongside [`apps/public`](../public/)
(:3000).

## Scripts

| Script           | Does                                       |
| ---------------- | ------------------------------------------ |
| `pnpm dev`       | `nuxi dev` on http://localhost:3001.       |
| `pnpm build`     | `nuxi build`.                              |
| `pnpm typecheck` | `nuxi typecheck` (vue-tsc).                |
