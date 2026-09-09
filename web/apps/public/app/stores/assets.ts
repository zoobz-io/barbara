import { useAsyncData, usePress } from "#imports";

import { API_PROXY_PREFIX } from "~~/config/press";

/**
 * The asset data layer for one app: the full listing behind a blocking
 * useAsyncData (the browser widget projects folder levels from it without
 * further requests), removal, and download URLs. Call in setup.
 */
export async function useAssets(appId: string) {
  const api = usePress("api");
  const { data, refresh } = await useAsyncData(
    `app-assets-${appId}`,
    async () => (await api.assets.list(appId)).assets,
  );

  return {
    assets: data,
    refresh,
    remove: (key: string) => api.assets.delete(appId, { query: { key } }),
    /** The proxied download URL for a key — same origin, browser-safe. */
    downloadUrl: (key: string) =>
      `${API_PROXY_PREFIX}/apps/${appId}/assets/object?key=${encodeURIComponent(key)}`,
  };
}
