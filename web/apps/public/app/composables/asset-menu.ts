import type { MenuItem } from "@zoobzio/foundation/types/core/menu";

import { ASSET_ACTION } from "~/constants/assets";
import { useAssetStore } from "~/stores/assets";
import { useAssetAction } from "~/composables/asset-action";

/**
 * Dispatches a pick from the asset action menu: the reads act at once, the
 * changes and the delete hand off to the dialogs. Call in setup.
 */
export function useAssetMenu(appId: string) {
  const store = useAssetStore(appId);
  const { open } = useAssetAction(appId);

  return {
    select: (key: string, item: MenuItem) => {
      switch (item.label) {
        case ASSET_ACTION.copyUrl:
          void navigator.clipboard.writeText(store.publicUrl(key));
          break;
        case ASSET_ACTION.download:
          window.open(store.downloadUrl(key), "_blank");
          break;
        case ASSET_ACTION.rename:
          open("rename", key);
          break;
        case ASSET_ACTION.move:
          open("move", key);
          break;
        case ASSET_ACTION.delete:
          open("delete", key);
          break;
      }
    },
  };
}
