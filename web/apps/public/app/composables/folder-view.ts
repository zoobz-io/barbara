import { useState } from "#imports";

/**
 * How one folder is being looked at: the search query and the sort, shared
 * between the toolbar that sets them and the table that applies them. The
 * sort is per app, so it carries from folder to folder; the query is per
 * folder, so leaving and returning finds it as it was. `feature` keeps the
 * asset tree's view apart from the content tree's. Call in setup.
 */
export function useFolderView<S extends string>(
  feature: string,
  appId: string,
  path: string,
  defaultSort: S,
) {
  const query = useState<string>(`${feature}-query-${appId}-${path}`, () => "");
  const sort = useState<S>(`${feature}-sort-${appId}`, () => defaultSort);
  return { query, sort };
}
