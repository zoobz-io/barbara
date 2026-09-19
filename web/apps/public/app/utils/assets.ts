import type { BreadcrumbItem } from "@zoobzio/foundation/types/core/breadcrumb";
import type { IconAlias } from "@zoobzio/foundation/types/icon";

import type {
  Asset,
  AssetDayStats,
  AssetKind,
  AssetKindStats,
  AssetPreview,
  AssetSort,
  AssetSubfolder,
  StorageSegment,
  StorageShares,
} from "~/types/assets";
import {
  KIND_LABEL,
  STORAGE_KIND_SLOTS,
  STORAGE_OTHER_LABEL,
} from "~/constants/assets";
import { pathCrumbs, pathRoute } from "~/utils/path";

/** The route for a path under an app's assets — a folder or an asset key,
 * the page tells them apart; "" is the root. */
export function assetRoute(appId: string, path: string): string {
  return pathRoute(`/apps/${appId}/assets`, path);
}

/** The breadcrumb trail for a path under an app's assets. */
export function assetCrumbs(
  appId: string,
  path: string,
  rootLabel: string,
): BreadcrumbItem[] {
  return pathCrumbs(path, rootLabel, (p) => assetRoute(appId, p));
}

/** The browser icon for each kind; the API classifies, this only maps. */
const KIND_ICON: Record<AssetKind, IconAlias> = {
  image: "file-image",
  video: "file-video",
  audio: "file-audio",
  code: "file-code",
  spreadsheet: "file-spreadsheet",
  archive: "file-archive",
  text: "file-text",
  file: "file",
};

/** True when a string is one of the API's kinds. */
export function isAssetKind(kind: string): kind is AssetKind {
  return kind in KIND_ICON;
}

/** The browser icon for an asset's kind, the plain file for anything else. */
export function assetIcon(kind: string): IconAlias {
  return isAssetKind(kind) ? KIND_ICON[kind] : KIND_ICON.file;
}

/**
 * How the asset page previews an asset: images, video, and audio render
 * natively; the text family shows its contents, except a PDF, which the
 * browser renders itself; code shows as text; the rest offer only a
 * download. Kind carries the family; the content type only tells PDF apart.
 */
export function assetPreview(kind: string, contentType: string): AssetPreview {
  switch (kind) {
    case "image":
    case "video":
    case "audio":
      return kind;
    case "code":
      return "text";
    case "text": {
      const type = contentType.split(";")[0]?.trim().toLowerCase();
      return type === "application/pdf" ? "pdf" : "text";
    }
    default:
      return "none";
  }
}

/** The kinds in the API's display order; the meter assigns hues along it. */
const KIND_ORDER: AssetKind[] = [
  "image",
  "video",
  "audio",
  "code",
  "spreadsheet",
  "archive",
  "text",
  "file",
];

/** The user-facing name of a kind; an unknown one is the plain file. */
export function kindLabel(kind: string): string {
  return isAssetKind(kind) ? KIND_LABEL[kind] : KIND_LABEL.file;
}

/**
 * The storage meter's segments: the largest kinds by bytes, each named,
 * and the rest folded into one Other segment; then the free space against
 * the limit. Named kinds take their hue slot in the kinds' fixed display
 * order, not by size, so an app's colors hold still as its numbers move.
 * Each segment's share is of the limit — what the bar draws — and its
 * fraction is of what is used — what the legend states. When usage exceeds
 * the limit the bar scales to the usage instead, so it still fits, and the
 * free space reads as zero.
 */
export function storageShares(
  kinds: AssetKindStats[],
  limit: number,
): StorageShares {
  const present = kinds.filter((k) => k.size > 0);
  const used = present.reduce((sum, k) => sum + k.size, 0);
  const bySize = [...present].sort((a, b) => b.size - a.size);
  const named = bySize
    .slice(0, STORAGE_KIND_SLOTS)
    .sort(
      (a, b) =>
        KIND_ORDER.indexOf(a.kind as AssetKind) -
        KIND_ORDER.indexOf(b.kind as AssetKind),
    );
  const rest = bySize.slice(STORAGE_KIND_SLOTS);
  const scale = Math.max(limit, used) || 1;
  const segment = (
    key: string,
    label: string,
    size: number,
    slot: number,
    detail?: string,
  ): StorageSegment => ({
    key,
    label,
    size,
    slot,
    detail,
    share: size / scale,
    fraction: used ? size / used : 0,
  });
  const segments = named.map((k, i) =>
    segment(k.kind, kindLabel(k.kind), k.size, i + 1),
  );
  const restSize = rest.reduce((sum, k) => sum + k.size, 0);
  if (restSize > 0) {
    segments.push(
      segment(
        "other",
        STORAGE_OTHER_LABEL,
        restSize,
        0,
        rest.map((k) => kindLabel(k.kind)).join(", "),
      ),
    );
  }
  const free = Math.max(0, limit - used);
  return { segments, used, free, limit, freeShare: free / scale };
}

/**
 * The assets last written within the past `days` UTC days, today included,
 * summed over the daily series. A day row is keyed on the UTC date, so the
 * window is counted in whole UTC days from the reference time.
 */
export function writtenWithin(
  series: AssetDayStats[],
  days: number,
  now: number,
): number {
  const today = Date.UTC(
    new Date(now).getUTCFullYear(),
    new Date(now).getUTCMonth(),
    new Date(now).getUTCDate(),
  );
  const from = today - (days - 1) * 86_400_000;
  return series.reduce((sum, d) => {
    const day = Date.parse(`${d.day}T00:00:00Z`);
    return day >= from && day <= today ? sum + d.count : sum;
  }, 0);
}

/** True when a row's name contains the query, case-insensitively. */
export function matchesQuery(name: string, query: string): boolean {
  const q = query.trim().toLowerCase();
  return q === "" || name.toLowerCase().includes(q);
}

/**
 * Orders a level's rows. Name sorts A to Z; size and modified sort largest
 * and newest first, since those are what the reader is looking for when
 * they pick them. A row without a timestamp sorts last under modified.
 * Ties fall back to the name, so the order is stable.
 */
function compareRows(
  sort: AssetSort,
  a: { name: string; size: number; at?: string },
  b: { name: string; size: number; at?: string },
): number {
  const byName = a.name.localeCompare(b.name, "en", { sensitivity: "base" });
  if (sort === "size") return b.size - a.size || byName;
  if (sort === "modified") {
    return (b.at ?? "").localeCompare(a.at ?? "") || byName;
  }
  return byName;
}

/** The subfolders matching the query, in sort order. */
export function sortFolders(
  folders: AssetSubfolder[],
  sort: AssetSort,
  query = "",
): AssetSubfolder[] {
  return folders
    .filter((f) => matchesQuery(f.name, query))
    .sort((a, b) =>
      compareRows(
        sort,
        { name: a.name, size: a.size, at: a.last_written_at },
        { name: b.name, size: b.size, at: b.last_written_at },
      ),
    );
}

/** The assets matching the query by file name, in sort order. */
export function sortAssets(
  assets: Asset[],
  sort: AssetSort,
  query = "",
): Asset[] {
  const name = (a: Asset) => a.key.slice(a.key.lastIndexOf("/") + 1);
  return assets
    .filter((a) => matchesQuery(name(a), query))
    .sort((a, b) =>
      compareRows(
        sort,
        { name: name(a), size: a.size, at: a.last_modified },
        { name: name(b), size: b.size, at: b.last_modified },
      ),
    );
}
