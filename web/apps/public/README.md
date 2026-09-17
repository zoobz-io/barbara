# @barbara/public

The Nuxt 4 public site over the Barbara **public API**, typed end-to-end through
[`@barbara/api-sdk`](../../packages/api-sdk/).

Dev server is pinned to **:3000** so it can run alongside [`apps/admin`](../admin/)
(:3001).

## How the SDK is consumed

Through [`@openapi-press/nuxt`](https://www.npmjs.com/package/@openapi-press/nuxt):
`nuxt.config.ts` mounts the SDK's default export (its Press) as the `api` client,
and code reaches it with the `usePress("api")` composable — fully typed, no
plugin, no manual client construction.

The browser never talks to the API host directly:

```
SSR:     Nuxt server ─────────────────────▶ http://127.0.0.1:8080
Client:  browser ──▶ /api (catch-all proxy) ─▶ http://127.0.0.1:8080
```

The host lives in private runtime config (override via
`NUXT_PRESS_CLIENTS_API_HOST`); the browser only ever sees the `/api` prefix.
SSR calls forward the incoming `cookie`/`authorization` headers, which is the
seam real auth will slot into — today the API runs the dev stub authenticator,
so every request resolves to the dev tenant.

## House style

The app extends the [`@zoobzio/foundation`](https://www.npmjs.com/package/@zoobzio/foundation)
layer (design system, untheme theming, icon-sheets icons). Auto-import stays
off — everything is imported explicitly, framework symbols from `#imports`.

Module config that isn't inline lives in `config/` at the app root
(`config/icon-sheets.ts` extends foundation's icon catalog).

`app/` is stratified into a clean DAG — each layer imports only from layers
above it:

| Layer          | Holds                                                              |
| -------------- | ------------------------------------------------------------------ |
| `types/`       | Type aliases (SDK wire shapes, view types).                        |
| `constants/`   | Constants; may import types.                                       |
| `utils/`       | Single-purpose functions over plain data.                          |
| `stores/`      | The data layer: SDK fetches + mutations over system composables.   |
| `composables/` | Reactivity wiring (dialog-form state, color-mode toggle).          |
| `components/`  | Wire composables/stores to templates; visualization/interactivity. |
| `pages/`       | Assemble components per route.                                     |
| `layouts/`     | Chrome shared between pages (the studio top bar).                  |

Styling lives in `app/assets/css/`, painted with untheme tokens from the
foundation theme (`var(--surface)`, `var(--on-surface-muted)`, …) so the app
follows theme/color-mode/contrast switches. The `#build/untheme.css` static
cascade is auto-linked by `@untheme/nuxt`.

## Pages

- `/` — landing: title/description, create-app link, the tenant's apps.
- `/apps/create`, `/apps/edit?id=…` — narrow form pages (create, rename).
- `/apps/:id` — the studio (layout `studio`): top bar with app picker,
  Content/Assets/History/Settings tabs, Review & Publish modal, color-mode
  toggle. Content shows the Pages sidebar (root listing + New folder/file
  dialogs) beside the editor pane.
- `/apps/:id/assets` — the assets landing page: the app's stat tiles and
  storage meter over the root folder's drop zone and rows.
- `/apps/:id/assets/*` — one catch-all for the asset tree. A path the
  parent folder lists as an asset (`…/assets/images/logo.png`) is that
  asset's page: a preview by media type (image, PDF, video, audio, text)
  beside its details, with download and delete. Any other listed path is a
  folder: a drop zone (uploads land in the folder, with progress) over a
  table of subfolders and files, each a link, and each navigation fetches
  exactly that level.

## Scripts

| Script           | Does                                 |
| ---------------- | ------------------------------------ |
| `pnpm dev`       | `nuxi dev` on http://localhost:3000. |
| `pnpm build`     | `nuxi build`.                        |
| `pnpm typecheck` | `nuxi typecheck` (vue-tsc).          |
