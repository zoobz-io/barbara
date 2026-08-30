# Design brief — Barbara consumer UI (wireframe)

**Engagement:** low-fidelity wireframes for the Barbara consumer application.
**Deliverable stage:** wireframe only — structure and flow, not visual design.
**Prepared for:** the design consultant. **Date:** 2026-08-30.

---

## 1. What Barbara is

Barbara is a content platform. A customer's website is a **set of Markdown
documents organized in folders**. Barbara stores those documents, tracks every
change to them, and publishes them to the live site.

The mental model is borrowed from version control, but the user never sees git:

- A **document** is a single Markdown page. Editing it creates a new immutable
  **version** — the old content is never overwritten.
- A **collection** is a folder. Folders nest. A document lives in one folder.
- A document has a **key** — its full path, e.g. `docs/guides/intro`. The path
  is derived from the folder tree.
- An **app** is one website. A customer can own several apps (several sites).
  An app holds one folder tree and one publish history.
- A **release** is a snapshot of the whole app at a moment in time. Publishing
  = cutting a release. The live website always serves the current release.

The important consequence for the UI: **editing is not publishing.** A customer
can edit ten pages, add a folder, and delete a page, and the live site does not
change at all until they publish. Publishing takes everything that changed and
makes it live at once. Nothing partial is ever live.

## 2. Who this UI is for

The **consumer**: a non-technical person who owns and maintains a website. A
marketing manager, a small-business owner, a docs writer. They are comfortable
with tools like Notion, Google Docs, or a simple CMS. They are **not**
developers and do not know what git, a "commit", or a "release snapshot" is —
so the interface must teach the model through plain language, not jargon.

Their jobs, in priority order:

1. **Change the content** of their site — edit a page, add a page, reorganize
   folders.
2. **Publish** those changes when ready, and trust that the live site matches.
3. **Manage assets** — upload and organize images and files the pages use.
4. **Recover** — see what changed, and roll back if a publish was wrong.
5. **Configure** the app — starting with just its name.

## 3. Scope of this engagement

**In scope — wireframe the consumer authoring application:**

- Signing in and choosing which app to work on.
- Editing pages and organizing the folder tree.
- Uploading and managing assets.
- Reviewing and publishing changes; viewing publish history; rolling back.
- Viewing a page's version history and restoring an earlier version.
- A minimal settings surface.

**Out of scope — do not design these now:**

- The **rendered public website** itself. Barbara publishes to it; it is a
  separate product surface. This brief is the *management* app only.
- Visual/brand design, color, typography, illustration. This pass is
  **grayscale, low-fidelity structure**.
- The sign-in provider screens (SSO handles authentication upstream). Assume
  the user arrives already authenticated; the first screen we own is the app
  list.
- Team/roles/permissions management, billing, and onboarding flows.

## 4. Concepts the UI must make legible

These are the ideas the wireframe has to communicate clearly. They are the hard
part of this design.

- **Draft vs. live.** At any moment a page is in one of three states, and the
  UI must show which, everywhere a page appears:
  - **Draft** — never published; not on the live site.
  - **Published** — the live site matches the current content.
  - **Published, newer draft** — live, but with unpublished edits waiting.
- **Pending changes.** The app as a whole has a set of changes not yet
  published (edited pages, new pages, deleted pages, moved folders). The user
  needs a persistent, honest signal of "you have N unpublished changes" and a
  way to review exactly what they are before publishing.
- **Publishing is one action for the whole app.** There is no per-page "go
  live" button. The primitive is **Publish changes**, which makes every pending
  change live together as one release.
- **History is safe.** Every publish is kept forever as a release. Rolling back
  never destroys anything — it creates a new release that restores an earlier
  state. Publish numbers only ever move forward.

## 5. Information architecture

```
Sign in (upstream, not designed)
   │
   ▼
App list  ──────────────────────────────►  Create app
   │  (pick an app)
   ▼
App workspace  ┌─ App switcher (top, switch between apps)
               │
               ├─ Pages        (folder tree + page editor + version history)
               ├─ Assets       (upload + organize files)
               ├─ Settings      (app name; danger zone)
               │
               └─ Publish        (pending-changes indicator + Publish action)
                     ├─ Review changes  (what will go live)
                     └─ Publish history (past releases + roll back)
```

## 6. Screen-by-screen requirements

### 6.1 App list (home after sign-in)

- Lists the customer's apps as cards or rows. Each shows the app name and,
  ideally, a light status line (e.g. last published date).
- A clear **Create app** action — a new app needs only a name to start.
- Selecting an app enters the app workspace.
- **Empty state:** a first-time user with no apps. Guide them to create their
  first one.

### 6.2 App workspace shell

- A persistent top bar carries: the **app switcher** (current app name; opens a
  menu to jump to another app or the app list), the primary navigation
  (**Pages / Assets / Settings**), and the **publish control**.
- The publish control is always visible and always honest: it shows the count
  of pending changes and the **Publish changes** action. When nothing is
  pending, it reads as "Everything published" / disabled.

### 6.3 Pages — folder tree + editor

This is the primary workspace. A two-pane layout is the expected shape, but the
consultant should propose what serves the model best.

- **Folder tree (left):** the app's collections and documents, nested. Supports:
  - Create a folder; create a page inside a folder or at the root.
  - Rename and move (drag or menu) folders and pages. Moving/renaming a folder
    changes the paths of everything inside it — the UI should make that
    consequence visible before it is confirmed.
  - Delete a folder (only when empty) and delete a page.
  - Each page shows its **status** (draft / published / published-with-newer-
    draft) at a glance.
- **Page editor (right):** a Markdown editing surface. Details of the editor
  (raw Markdown, rich text, or split preview) are the consultant's
  recommendation — flag the trade-off. Must include:
  - A clear **Save draft** action. Saving stores a new version; it does **not**
    publish.
  - The page's current status and path.
  - A **conflict** case: if the page changed underneath the user since they
    opened it, saving is refused and they must be shown a clear, recoverable
    message (not a silent overwrite).
- **Version history (tab or panel on the page):**
  - Lists the page's versions, newest first, with who changed it and when.
  - View an earlier version, and **restore** it (restoring creates a new draft
    version from the old content — it does not itself publish).
- **Empty states:** an app with no pages yet; a folder with no contents.

### 6.4 Publish flow

- **Review changes:** opened from the publish control. Lists every pending
  change grouped by type — pages added, pages edited, pages removed, folders
  moved/renamed. The user confirms from here. This is the safety surface: it is
  the last honest look before the live site changes.
- **Publish action:** cuts a release. After it completes, pending count returns
  to zero and affected pages read as published. Note for the consultant: the
  live site updates a few seconds after publish (asynchronous) — the UI should
  set that expectation rather than imply instant.
- **Publish history:** a list of past releases (number, date, who published).
  Each can be **rolled back** — which publishes a new release restoring that
  earlier state. Make clear rollback moves *forward* (it is a new publish), not
  a deletion of later history.

### 6.5 Assets

- Upload files (images and other binaries the pages reference).
- Organize assets in a **folder-like view by key prefix** — note this is a
  naming convention, not the same tree as pages. Assets live outside the page
  folder tree.
- List, preview (where sensible), and delete assets. Show enough metadata to be
  useful (name/key, type, size).
- **Assets are not versioned and not part of releases.** Uploading over an
  existing key **overwrites immediately and goes live immediately** — unlike
  pages, there is no draft/publish step for assets. This asymmetry must be
  obvious in the UI so a user is never surprised that a replaced image changed
  the live site instantly. Flag this as a UX risk to solve.
- **Empty state:** no assets uploaded yet.

### 6.6 Settings (minimal)

- **General:** rename the app.
- **Danger zone:** delete the app — allowed only when it has no releases (i.e.
  never published). The UI must explain why delete is unavailable once
  published.
- Everything else customers will eventually want — custom domain, theme,
  redirects — is **not built yet**. Represent it honestly as "coming soon" or
  omit it; do not design controls that do nothing. (If the consultant wants to
  sketch the eventual shape as a separate, clearly-labeled "future" appendix,
  that is welcome but must not read as shippable.)

## 7. Cross-cutting states

Please address, not just the happy path:

- The three page statuses, shown consistently wherever a page is listed.
- The app-level pending-changes indicator in every state (some / none).
- Empty states for: no apps, no pages, empty folder, no assets, no publish
  history.
- Loading and error states for the primary actions (save, publish, upload).
- The save **conflict** case (section 6.3).
- The asset **overwrite-goes-live-instantly** case (section 6.5).

## 8. Constraints & non-goals — do not design past the product

The wireframe must stay inside what the platform actually does. Please do **not**
introduce:

- **Branching, staging environments, or "preview then merge" flows.** History
  is linear: edit → publish → (maybe) roll back. There is exactly one live
  release per app.
- **Per-page publishing.** Publishing is one app-wide action. (A page's status
  reflects the last app publish; there is no per-page go-live button.)
- **Scheduled publishing, approvals/workflow, or multi-author review.** Not in
  scope for v1.
- **Asset versioning or asset drafts.** Assets are live-on-upload (section 6.5).
- **Settings fields with no backing** (domain, theme, SEO, analytics). See 6.6.
- **Nested apps or shared folders across apps.** An app is a self-contained
  site.

## 9. Deliverables

- Low-fidelity, grayscale wireframes covering every screen in section 6 and the
  states in section 7.
- A simple screen-flow/sitemap tying them together (annotated section 5).
- Annotations that call out the model-legibility decisions: how draft-vs-live
  is shown, how pending changes are surfaced, how the asset asymmetry is
  handled.
- A short written rationale for the two open editor/layout recommendations
  (Markdown editing surface; Pages layout).

Fidelity: **wireframe only.** No brand, color, or final typography this pass.
We will review the wireframe together, then break it into build tickets.

## 10. Questions we expect you to raise

We would rather you flag these than assume:

- The Markdown editing surface: raw, rich-text, or split-preview — and why.
- Whether the Pages tree and the editor share one screen or are separate.
- How to present "moving a folder changes every path inside it" without
  alarming a non-technical user.
- The clearest language for "draft / published / published-with-newer-draft"
  for someone who has never heard the word "publish" in this sense.

---

### Appendix A — operations the UI can rely on

The backend already supports exactly these actions. Design to them; anything not
listed is not available yet.

- **Apps:** create (name only), get, list, rename, delete (only if never
  published).
- **Folders (collections):** create under a parent or at root, rename, move,
  delete (empty only), list a folder's contents (subfolders + pages + each
  page's status) in one call.
- **Pages (documents):** create in a folder, open with latest content, move/
  rename, delete, list, list by tag, read status.
- **Content (versions):** save a new version (with conflict detection against
  the version the user started from), list versions newest-first, view a
  version.
- **Publishing (releases):** publish the whole app (cut a release), list
  releases newest-first, view a release and its contents, roll back to an
  earlier release.
- **Assets:** upload/overwrite by key, download, list by key-prefix, delete.
  Not versioned; not part of releases.
- **Tags (optional surface):** add/remove an organizational tag on a page.

### Appendix B — glossary for the wireframe's plain language

Internal term → suggested consumer-facing word (consultant may improve):

- app → **site** (or **project**)
- collection → **folder**
- document → **page**
- version → **revision** / **change**
- release → **publish**
- key → **path**
