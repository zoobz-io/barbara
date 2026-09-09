import { computed, useAsyncData, usePress } from "#imports";

/** An app's release history, newest first. Call in setup. */
export async function useReleases(appId: string) {
  const api = usePress("api");
  const { data, refresh } = await useAsyncData(`app-releases-${appId}`, () =>
    api.releases.list(appId),
  );

  return {
    releases: computed(() => data.value?.releases ?? []),
    refresh,
  };
}
