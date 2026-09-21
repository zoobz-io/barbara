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
- `/apps/:id` — forwards to the app's content root.
- `/apps/:id/content` — the studio (layout `studio`): top bar with app
  picker, Content/Assets/Releases/Settings tabs, color-mode toggle. The
  content landing page is the root folder of the app's document tree: New
  folder/New page dialogs over a searchable, sortable table of subfolders
  and pages with each page's status.
- `/apps/:id/content/*` — one catch-all for the content tree, mirroring
  assets. A path the parent level lists as a document
  (`…/content/guides/install.md`) is that page's editor: the Pages sidebar
  tree beside the tiptap editor, with Save. Any other listed path is a
  folder page: crumbs, counts, New folder/New page, and the level's rows.
  The API addresses a level by collection id, so the store resolves each
  path through its parent level and caches every level it has seen.
- `/apps/:id/assets` — the assets landing page: the app's stat tiles and
  storage meter over the root folder's drop zone and rows.
- `/apps/:id/assets/*` — one catch-all for the asset tree. A path the
  parent folder lists as an asset (`…/assets/images/logo.png`) is that
  asset's page: a preview by media type (image, PDF, video, audio, text)
  beside its details, with download and delete. Any other listed path is a
  folder: a drop zone (uploads land in the folder, with progress) over a
  table of subfolders and files, each a link, and each navigation fetches
  exactly that level.
- `/apps/:id/releases` — the releases timeline: every release newest first,
  each a card with its number, kind, label, when and by whom, how many
  pages it served, and its change counts against the release before; the
  live one is marked. Older pages append on demand.
- `/apps/:id/releases/:release` — one release, laid out like an asset: the
  manifest of pages it served (each marked added, changed, or moved once
  the changes load) beside its details and what it changed, with Restore,
  which cuts a new release copying this one forward.
- `/apps/:id/releases/:release/*` — the viewer for a page a release served:
  the pages sidebar beside the prose, read-only, at the version the release
  served, with its details. "Open in editor" takes that version to the
  page's editor (`…/content/<key>?version=…`) as an unsaved draft, so
  saving lands a new version.

## Scripts

| Script           | Does                                 |
| ---------------- | ------------------------------------ |
| `pnpm dev`       | `nuxi dev` on http://localhost:3000. |
| `pnpm build`     | `nuxi build`.                        |
| `pnpm typecheck` | `nuxi typecheck` (vue-tsc).          |
