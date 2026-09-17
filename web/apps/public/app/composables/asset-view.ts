import { useState } from "#imports";

import type { AssetSort } from "~/types/assets";
import { DEFAULT_ASSET_SORT } from "~/constants/assets";

/**
 * How one folder is being looked at: the search query and the sort, shared
 * between the toolbar that sets them and the table that applies them. The
 * sort is per app, so it carries from folder to folder; the query is per
 * folder, so leaving and returning finds it as it was. Call in setup.
 */
export function useAssetView(appId: string, path: string) {
  const query = useState<string>(`assets-query-${appId}-${path}`, () => "");
  const sort = useState<AssetSort>(
    `assets-sort-${appId}`,
    () => DEFAULT_ASSET_SORT,
  );
  return { query, sort };
}
