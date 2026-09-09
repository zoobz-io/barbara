import { defineEntity } from "@zoobzio/foundation/definitions/entity";

import type { AssetRow } from "~/types/assets";

const entity = defineEntity<AssetRow>();

/**
 * The asset browser definition: pure serializable config. Behavior (fetch,
 * action handlers) wires up at `useBrowser` in the asset-browser component,
 * keyed against the action names declared here.
 */
export const ASSET_BROWSER = entity.defineBrowser({
  columns: [
    { key: "name", label: "Name", type: "text", sortable: true },
    { key: "content_type", label: "Type", type: "text" },
    {
      key: "size",
      label: "Size",
      type: "filesize",
      align: "right",
      sortable: true,
    },
  ],
  fileKey: "key",
  rootLabel: "All",
  actions: {
    download: { icon: "download", label: "Download" },
    delete: { icon: "delete", label: "Delete" },
  },
  bulkActions: {
    delete: { icon: "delete", label: "Delete" },
  },
});
