import { useAsyncData, usePress } from "#imports";

import type { AppContents } from "~/types/apps";
import type { PageNode } from "~/types/pages";
import { pageNodes } from "~/utils/pages";

/**
 * An app's document tree, loaded whole through one blocking useAsyncData —
 * the walk recurses the per-level reads, so the sidebar never fetches after
 * page load. Creations land at the app root until folder selection exists;
 * both refresh the tree on success.
 */
export async function useAppPages(appId: string) {
  const api = usePress("api");

  const level = (collectionId: string | null): Promise<AppContents> =>
    collectionId
      ? api.collections.contents(appId, collectionId)
      : api.apps.contents(appId);

  async function buildNodes(collectionId: string | null): Promise<PageNode[]> {
    const nodes = pageNodes(await level(collectionId), appId);
    await Promise.all(
      nodes
        .filter((node) => node.kind === "folder")
        .map(async (node) => {
          node.children = await buildNodes(node.id);
        }),
    );
    return nodes;
  }

  const { data: tree, refresh } = await useAsyncData(`app-tree-${appId}`, () =>
    buildNodes(null),
  );

  return {
    tree,
    refresh,
    createFolder: async (name: string) => {
      await api.collections.create(appId, { body: { name } });
      await refresh();
    },
    createFile: async (name: string) => {
      await api.documents.create(appId, { body: { name } });
      await refresh();
    },
  };
}

/**
 * Every document in an app, gathered by walking the tree level by level —
 * the tenant-wide /documents list carries no app scope, so the table of
 * contents assembles from the same per-level reads the sidebar uses.
 * Call in setup; sorted by key.
 */
export function useAppDocuments(appId: string) {
  const api = usePress("api");

  async function collect(
    collectionId: string | null,
  ): Promise<AppContents["documents"]> {
    const contents = collectionId
      ? await api.collections.contents(appId, collectionId)
      : await api.apps.contents(appId);
    const nested = await Promise.all(
      contents.subcollections.map((sub) => collect(sub.id)),
    );
    return [...contents.documents, ...nested.flat()];
  }

  return useAsyncData(`app-documents-${appId}`, async () =>
    (await collect(null)).sort((a, b) => a.key.localeCompare(b.key)),
  );
}
