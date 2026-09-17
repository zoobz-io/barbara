import type { Option } from "@zoobzio/foundation/types/core/common";
import type { MenuGroup } from "@zoobzio/foundation/types/core/menu";

import type { AssetKind, AssetSort } from "~/types/assets";

/** The root crumb of the asset tree. */
export const ASSET_ROOT_LABEL = "Assets";

/** The landing page's one-line description of the feature. */
export const ASSETS_LEDE =
  "Images, documents, and other files your pages reference. Folders are just key prefixes: upload into any level, and every asset serves from the published URL as soon as it lands.";

/**
 * The storage an app may use, in bytes. A placeholder until plans and quotas
 * land on the API, at which point the stats response carries the limit and
 * this constant goes.
 */
export const ASSET_STORAGE_LIMIT_BYTES = 1024 ** 3;

/** The storage meter shows this many kinds by name; the rest fold into Other. */
export const STORAGE_KIND_SLOTS = 3;

/** The legend name of the folded tail of the storage meter. */
export const STORAGE_OTHER_LABEL = "Other";

/** Past this share of the limit, the free space reads as a warning. */
export const STORAGE_WARN_SHARE = 0.9;

/** The user-facing name of each kind, in the API's display order. */
export const KIND_LABEL: Record<AssetKind, string> = {
  image: "Images",
  video: "Video",
  audio: "Audio",
  code: "Code",
  spreadsheet: "Spreadsheets",
  archive: "Archives",
  text: "Documents",
  file: "Other files",
};

/** The per-file action labels; the menus dispatch on them. */
export const ASSET_ACTION = {
  copyUrl: "Copy URL",
  download: "Download",
  rename: "Rename",
  move: "Move",
  delete: "Delete",
} as const;

/** The per-file action menu: the reads, the moves, then the one-way door. */
export const ASSET_ACTIONS: MenuGroup[] = [
  {
    key: "read",
    items: [
      { label: ASSET_ACTION.copyUrl, icon: "copy" },
      { label: ASSET_ACTION.download, icon: "download" },
    ],
  },
  {
    key: "change",
    items: [
      { label: ASSET_ACTION.rename, icon: "pencil" },
      { label: ASSET_ACTION.move, icon: "folder-input" },
    ],
  },
  {
    key: "remove",
    items: [{ label: ASSET_ACTION.delete, icon: "delete" }],
  },
];

/** The sort choices of the folder toolbar, as the select lists them. */
export const ASSET_SORT_OPTIONS: (Option & { value: AssetSort })[] = [
  { value: "name", label: "Name" },
  { value: "size", label: "Size" },
  { value: "modified", label: "Modified" },
];

/** The sort a folder opens with. */
export const DEFAULT_ASSET_SORT: AssetSort = "name";
