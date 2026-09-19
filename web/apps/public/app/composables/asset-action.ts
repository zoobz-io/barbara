import { useState } from "#imports";

import type { AssetPendingAction } from "~/types/assets";

/**
 * The asset action awaiting a dialog — rename, move, or delete on one key —
 * shared between whatever opened it (a row menu, the detail page) and the
 * dialogs component that carries it out. One at a time per app. Call in
 * setup.
 */
export function useAssetAction(appId: string) {
  const pending = useState<AssetPendingAction | null>(
    `assets-action-${appId}`,
    () => null,
  );
  return {
    pending,
    open: (action: AssetPendingAction["action"], key: string) => {
      pending.value = { action, key };
    },
    close: () => {
      pending.value = null;
    },
  };
}
