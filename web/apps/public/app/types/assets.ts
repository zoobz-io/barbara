import type { components } from "@barbara/api-sdk";

/** An asset's metadata as the public API serves it (bytes live elsewhere). */
export type Asset = components["schemas"]["AssetResponse"];

/**
 * An asset's media family, as the API classifies it. One vocabulary for
 * icons, previews, and the stats breakdown; the API decides, the studio maps.
 */
export type AssetKind =
  | "image"
  | "video"
  | "audio"
  | "code"
  | "spreadsheet"
  | "archive"
  | "text"
  | "file";

/** One level of an app's asset tree: its subfolders and direct assets. */
export type AssetFolder = components["schemas"]["AssetFolderResponse"];

/** A subfolder at one level: its name and the count beneath it. */
export type AssetSubfolder = components["schemas"]["AssetSubfolderResponse"];

/**
 * The app-level view of its assets: totals, the breakdown by kind, and
 * writes per UTC day — what the asset landing page charts. Kept by
 * bookkeeping rows, not counted live; `computed_at` is absent until the
 * rows were first rebuilt from object storage.
 */
export type AssetStats = components["schemas"]["AssetStatsResponse"];

/** One kind's rollup within the stats. */
export type AssetKindStats = components["schemas"]["AssetKindStatsResponse"];

/** One UTC day of the series: its total and its split by kind. */
export type AssetDayStats = components["schemas"]["AssetDayStatsResponse"];

/** One upload in flight or finished, kept until cleared. */
export type AssetUpload = {
  id: string;
  key: string;
  name: string;
  size: number;
  contentType: string;
  /** The API's classification, known once the upload lands. */
  kind?: AssetKind;
  /** 0–100. */
  progress: number;
  status: "uploading" | "done" | "error";
  error?: string;
};

/** How the asset page previews a media type. */
export type AssetPreview = "image" | "pdf" | "video" | "audio" | "text" | "none";

/**
 * One segment of the storage meter: a named kind (slot 1 and up, its hue)
 * or the folded tail (slot 0, gray). Share is of the bar's full width;
 * fraction is of the storage in use.
 */
export type StorageSegment = {
  key: string;
  label: string;
  /** What the folded tail holds, by name; absent on a named kind. */
  detail?: string;
  size: number;
  slot: number;
  share: number;
  fraction: number;
};

/** The storage meter's data: its segments and the space left. */
export type StorageShares = {
  segments: StorageSegment[];
  used: number;
  free: number;
  limit: number;
  /** The free space's share of the bar's full width. */
  freeShare: number;
};

/** How a folder's rows order: by name, by size, or by when they changed. */
export type AssetSort = "name" | "size" | "modified";

/** An action on one asset that a dialog must confirm or complete. */
export type AssetPendingAction = {
  action: "rename" | "move" | "delete";
  key: string;
};
