// Regenerate the golden mdast fixtures under internal/mdast/testdata/.
//
// Each fixture is a `<name>.md` file. This script parses it with the same
// remark building blocks a browser mdast renderer uses and writes the tree to
// `<name>.json`. The Go converter in internal/mdast must produce byte-equal
// JSON, so the JSON here is the reference truth for "any remark renderer works".
//
// Run from web/: `pnpm fixtures:mdast`.
//
// The JSON is normalized so the Go side has a simple, stable target:
//   - `position` fields are stripped (source offsets are not part of the tree).
//   - `yaml` nodes are removed; frontmatter is page metadata, not a body node
//     (see #94), so the golden body must not contain it.
//   - object keys are sorted and the indent is two spaces, so a diff between
//     two runs, or between remark and Go, is a content diff and nothing else.

import { readdir, readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { fromMarkdown } from "mdast-util-from-markdown";
import { gfm } from "micromark-extension-gfm";
import { gfmFromMarkdown } from "mdast-util-gfm";
import { frontmatter } from "micromark-extension-frontmatter";
import { frontmatterFromMarkdown } from "mdast-util-frontmatter";
import { removePosition } from "unist-util-remove-position";
import { parse as parseYaml } from "yaml";

const here = dirname(fileURLToPath(import.meta.url));
const testdata = join(here, "..", "..", "internal", "mdast", "testdata");

// Parse markdown to an mdast tree with GFM and YAML frontmatter recognized.
function parse(markdown) {
  return fromMarkdown(markdown, {
    extensions: [gfm(), frontmatter(["yaml"])],
    mdastExtensions: [gfmFromMarkdown(), frontmatterFromMarkdown(["yaml"])],
  });
}

// Drop yaml nodes anywhere in the tree. Frontmatter is exposed as metadata by
// the converter (#94), not as a body node, so the golden body omits it.
function removeYaml(node) {
  if (Array.isArray(node.children)) {
    node.children = node.children.filter((child) => child.type !== "yaml");
    for (const child of node.children) removeYaml(child);
  }
  return node;
}

// Recursively sort object keys so the JSON output is stable regardless of the
// order remark happens to build fields in. Arrays keep their order.
function sortKeys(value) {
  if (Array.isArray(value)) return value.map(sortKeys);
  if (value && typeof value === "object") {
    const out = {};
    for (const key of Object.keys(value).sort())
      out[key] = sortKeys(value[key]);
    return out;
  }
  return value;
}

async function main() {
  const entries = await readdir(testdata);
  const bases = entries
    .filter((name) => name.endsWith(".md") && name !== "README.md")
    .map((name) => name.slice(0, -".md".length))
    .sort();

  if (bases.length === 0) {
    throw new Error(`no .md fixtures found in ${testdata}`);
  }

  for (const base of bases) {
    const markdown = await readFile(join(testdata, `${base}.md`), "utf8");
    const tree = parse(markdown);
    removePosition(tree, { force: true });

    // A recognized frontmatter block becomes a yaml node at the top of the
    // tree. The converter (#94) returns its decoded YAML as page metadata, so
    // write a <base>.meta.json beside the fixture for the metadata to match.
    const yaml = tree.children.find((child) => child.type === "yaml");
    if (yaml) {
      // Malformed YAML in a fenced block is not an error: the converter returns
      // empty metadata for it, so no .meta.json is written and the fixture test
      // asserts the empty map.
      let data;
      try {
        data = parseYaml(yaml.value);
      } catch {
        data = undefined;
      }
      if (data !== undefined) {
        const meta = `${JSON.stringify(sortKeys(data), null, 2)}\n`;
        await writeFile(join(testdata, `${base}.meta.json`), meta);
        console.log(`wrote ${base}.meta.json`);
      }
    }

    removeYaml(tree);
    const json = `${JSON.stringify(sortKeys(tree), null, 2)}\n`;
    await writeFile(join(testdata, `${base}.json`), json);
    console.log(`wrote ${base}.json`);
  }
}

await main();
