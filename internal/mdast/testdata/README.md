# mdast golden fixtures

Each fixture is a Markdown file (`<name>.md`) paired with the mdast tree a
[remark] parser produces for it (`<name>.json`). The Go converter in
`internal/mdast` (#93) must produce byte-equal JSON for every one, which makes
"any remark renderer can render a Barbara page" a tested contract rather than a
claim.

The `.json` files are generated, never edited by hand. They are normalized so
the Go side has a stable target:

- source `position` fields are stripped,
- `yaml` frontmatter nodes are removed (frontmatter is page metadata, not a body
  node — see #94),
- object keys are sorted and the indent is two spaces.

The generator lives in the web workspace at
[`web/scripts/mdast-fixtures.mjs`](../../../web/scripts/mdast-fixtures.mjs) and
pins the remark package versions, since a version bump can change the tree.

## Adding a fixture

1. Write `<name>.md` here. Keep it focused on one area (see the existing files).
2. From `web/`, run `pnpm fixtures:mdast`. It writes `<name>.json` for every
   `.md` file, including yours.
3. Read the generated `<name>.json` and confirm it is the tree you expected.
4. Run `pnpm fixtures:mdast` a second time and confirm it produces no diff.
5. Commit both `<name>.md` and `<name>.json`.

## Regenerating

Run `pnpm fixtures:mdast` from `web/`. It regenerates every `.json` from its
`.md`. A regeneration should produce no diff unless a `.md` file or a pinned
remark version changed.

[remark]: https://github.com/remarkjs/remark
