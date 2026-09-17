import {
  computed,
  ref,
  useAsyncData,
  useLazyAsyncData,
  usePress,
} from "#imports";

import type { Release } from "~/types/releases";
import { RELEASE_PAGE_SIZE } from "~/constants/history";

/**
 * An app's release history, newest first. The first page blocks the page
 * render; older pages append on demand. Call in setup.
 */
export async function useReleases(appId: string) {
  const api = usePress("api");
  const older = ref<Release[]>([]);
  const loadingMore = ref(false);

  const page = (offset: number) =>
    api.releases.list(appId, {
      query: { limit: String(RELEASE_PAGE_SIZE), offset: String(offset) },
    });

  const { data, refresh } = await useAsyncData(`app-releases-${appId}`, () =>
    page(0),
  );

  const releases = computed<Release[]>(() => [
    ...(data.value?.releases ?? []),
    ...older.value,
  ]);

  // The API reports only the count returned, so a full page is the signal
  // that older releases may exist.
  const lastPageSize = ref(data.value?.releases.length ?? 0);
  const hasMore = computed(() => lastPageSize.value === RELEASE_PAGE_SIZE);

  async function more() {
    if (!hasMore.value || loadingMore.value) return;
    loadingMore.value = true;
    try {
      const next = await page(releases.value.length);
      older.value = [...older.value, ...next.releases];
      lastPageSize.value = next.releases.length;
    } finally {
      loadingMore.value = false;
    }
  }

  return {
    releases,
    hasMore,
    loadingMore,
    more,
    refresh: async () => {
      older.value = [];
      await refresh();
      lastPageSize.value = data.value?.releases.length ?? 0;
    },
  };
}

/**
 * One release's entries, fetched client-side without blocking navigation —
 * each release section calls this, so the sections fan out in parallel.
 *
 * A release is immutable, so its entries are fetched once per session:
 * Nuxt's default only reuses data while hydrating, which would refetch
 * every section on each visit to History. Serving the payload cache keeps
 * a fetched release warm across navigations; only a new, unseen release
 * hits the API.
 */
export function useReleaseEntries(appId: string, releaseId: string) {
  const api = usePress("api");
  const { data, status, error } = useLazyAsyncData(
    `release-${releaseId}`,
    async () => (await api.releases.get(appId, releaseId)).entries,
    { getCachedData: (key, nuxtApp) => nuxtApp.payload.data[key] },
  );

  return {
    entries: computed(() => data.value ?? []),
    status,
    error,
  };
}
