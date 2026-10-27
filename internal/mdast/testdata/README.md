# mdast and hast golden fixtures

Each fixture is a Markdown file (`<name>.md`) paired with the two trees a
[remark]/[rehype] toolchain produces for it:

- `<name>.json` — the [mdast] tree (`mdast-util-from-markdown`). The Go
  converter in [`internal/mdast`](../..) must produce byte-equal JSON.
- `<name>.hast.json` — the [hast] tree (`mdast-util-to-hast`). The Go converter
  in [`internal/hast`](../../../hast) must produce byte-equal JSON.

Byte-equality for both makes "any remark renderer can render a Barbara page" and
"any hast renderer can render a Barbara page" tested contracts rather than
claims.

The JSON files are generated, never edited by hand. They are normalized so the
Go side has a stable target:

- source `position` fields are stripped (and `data` bookkeeping, for hast),
- `yaml` frontmatter nodes are removed (frontmatter is page metadata, not a body
  node — see #94),
- object keys are sorted and the indent is two spaces.

The mdast → hast conversion deviates from `mdast-util-to-hast`'s defaults in two
ways, so the `.hast.json` reflects them: raw HTML is passed through as `raw`
nodes (`allowDangerousHtml`), and a fence's `meta` is kept as the `dataMeta`
property. See [`internal/hast`](../../../hast) for the detail.

The generator lives in the web workspace at
[`web/scripts/mdast-fixtures.mjs`](../../../web/scripts/mdast-fixtures.mjs) and
pins the remark and `mdast-util-to-hast` versions, since a version bump can
change a tree.

## Adding a fixture

1. Write `<name>.md` here. Keep it focused on one area (see the existing files).
2. From `web/`, run `pnpm fixtures:mdast`. It writes `<name>.json` and
   `<name>.hast.json` for every `.md` file, including yours.
3. Read the generated trees and confirm they are what you expected.
4. Run `pnpm fixtures:mdast` a second time and confirm it produces no diff.
5. Commit `<name>.md`, `<name>.json`, and `<name>.hast.json`.

## Regenerating

Run `pnpm fixtures:mdast` from `web/`. It regenerates every `.json` and
`.hast.json` from its `.md`. A regeneration should produce no diff unless a
`.md` file or a pinned package version changed.

[remark]: https://github.com/remarkjs/remark
[rehype]: https://github.com/rehypejs/rehype
[mdast]: https://github.com/syntax-tree/mdast
[hast]: https://github.com/syntax-tree/hast
