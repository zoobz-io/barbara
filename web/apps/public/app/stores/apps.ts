import { computed, refreshNuxtData, useAsyncData, usePress } from "#imports";

/**
 * The tenant's apps: the cached list plus mutations. Call in setup — the
 * list is keyed once ("apps") so every consumer shares the same fetch.
 */
export async function useApps() {
  const api = usePress("api");
  const { data, refresh } = await useAsyncData("apps", () => api.apps.list());

  return {
    apps: computed(() => data.value?.apps ?? []),
    refresh,
    create: (name: string) => api.apps.create({ body: { name } }),
  };
}

/** A single app by id, with its mutations. Call in setup. */
export async function useApp(id: string) {
  const api = usePress("api");
  const { data, refresh } = await useAsyncData(`app-${id}`, () =>
    api.apps.get(id),
  );

  return {
    app: data,
    refresh,
    // Mutations refresh both this app and the shared list ("apps") — the
    // top bar's picker reads the list, so it must not go stale.
    rename: async (name: string) => {
      await api.apps.rename(id, { body: { name } });
      await Promise.all([refresh(), refreshNuxtData("apps")]);
    },
    remove: async () => {
      await api.apps.delete(id);
      await refreshNuxtData("apps");
    },
  };
}
