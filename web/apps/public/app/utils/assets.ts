import type { Asset } from "~/types/assets";

/** One level of the asset tree: subfolders (with counts) and direct files. */
export type AssetLevel = {
  folders: { name: string; count: number }[];
  files: Asset[];
};

/**
 * Projects flat path-like keys onto one directory level. An asset folder is
 * a key prefix by convention — `images/logo.png` puts `images` at the root.
 * Folder counts tally every key under the folder, not just direct children.
 */
export function assetLevel(assets: Asset[], path: string): AssetLevel {
  const prefix = path === "" ? "" : `${path}/`;
  const folders = new Map<string, number>();
  const files: Asset[] = [];

  for (const asset of assets) {
    if (!asset.key.startsWith(prefix)) continue;
    const rest = asset.key.slice(prefix.length);
    const slash = rest.indexOf("/");
    if (slash === -1) {
      files.push(asset);
    } else {
      const folder = rest.slice(0, slash);
      folders.set(folder, (folders.get(folder) ?? 0) + 1);
    }
  }

  return {
    folders: [...folders.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, count]) => ({ name, count })),
    files: files.sort((a, b) => a.key.localeCompare(b.key)),
  };
}
