import {
  computed,
  ref,
  refreshNuxtData,
  useAsyncData,
  useLazyAsyncData,
  usePress,
} from "#imports";

import type { Release } from "~/types/releases";
import { RELEASE_PAGE_SIZE } from "~/constants/releases";

/**
 * An app's releases, newest first, with the app's total. The first page
 * blocks the page render; older pages append on demand. Call in setup.
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
  const total = computed(() => data.value?.total ?? 0);
  const hasMore = computed(() => releases.value.length < total.value);

  async function more() {
    if (!hasMore.value || loadingMore.value) return;
    loadingMore.value = true;
    try {
      const next = await page(releases.value.length);
      older.value = [...older.value, ...next.releases];
    } finally {
      loadingMore.value = false;
    }
  }

  return {
    releases,
    total,
    hasMore,
    loadingMore,
    more,
    refresh: async () => {
      older.value = [];
      await refresh();
    },
  };
}

/**
 * One release with its entries — the manifest of what it served. Blocks the
 * page render; a release that is not the app's rejects, which the page maps
 * to a 404.
 *
 * A release is immutable, so it is fetched once per session: Nuxt's default
 * only reuses data while hydrating, which would refetch the release on each
 * visit to it and to the pages it served. Serving the payload cache keeps a
 * fetched release warm across navigations; only a new, unseen release hits
 * the API. Call in setup.
 */
export async function useRelease(appId: string, releaseId: string) {
  const api = usePress("api");
  const { data, error } = await useAsyncData(
    `release-${releaseId}`,
    () => api.releases.get(appId, releaseId),
    { getCachedData: (key, nuxtApp) => nuxtApp.payload.data[key] },
  );

  return {
    release: computed(() => data.value?.release),
    entries: computed(() => data.value?.entries ?? []),
    error,
  };
}

/**
 * One release's changes against the release before it, fetched client-side
 * without blocking navigation: the manifest renders at once and the change
 * marks land beside it. Cached for the session like the release. Call in
 * setup.
 */
export function useReleaseChanges(appId: string, releaseId: string) {
  const api = usePress("api");
  const { data, status, error } = useLazyAsyncData(
    `release-changes-${releaseId}`,
    async () => (await api.releases.changes(appId, releaseId)).changes,
    { getCachedData: (key, nuxtApp) => nuxtApp.payload.data[key] },
  );

  return {
    changes: computed(() => data.value ?? []),
    status,
    error,
  };
}

/**
 * The release mutations: restore cuts a new release copying an old one's
 * pages forward, then refreshes the timeline and the app (whose live
 * release moved). Call in setup.
 */
export function useReleaseActions(appId: string) {
  const api = usePress("api");

  return {
    restore: async (releaseId: string, label: string): Promise<Release> => {
      const release = await api.releases.rollback(appId, releaseId, {
        body: label ? { label } : {},
      });
      await Promise.all([
        refreshNuxtData(`app-releases-${appId}`),
        refreshNuxtData(`app-${appId}`),
      ]);
      return release;
    },
  };
}
