import { computed, useAsyncData, usePress, useState } from "#imports";

import type { ContentLevel, ContentNode } from "~/types/content";
import { contentNodes } from "~/utils/content";
import { keyName } from "~/utils/format";
import { childPath, parentPath } from "~/utils/path";

/**
 * The content data layer for one app: every level loaded so far, keyed by
 * folder path ("" is the root), plus creation into a folder. The API
 * addresses a level by its collection id, so a path resolves through its
 * parent: the root is one read, and each segment beneath it is the
 * subcollection of that name in the level above. State is the source of
 * truth — `load` answers from it when the level is already here, so
 * revisiting a folder costs nothing and a deep first visit costs one read
 * per ancestor. Call in setup.
 */
export function useContentStore(appId: string) {
  const api = usePress("api");
  const levels = useState<Record<string, ContentLevel>>(
    `content-${appId}`,
    () => ({}),
  );

  async function fetchLevel(path: string): Promise<ContentLevel> {
    if (path === "") {
      return { id: null, ...(await api.apps.contents(appId)) };
    }
    const parent = await load(parentPath(path));
    const name = keyName(path);
    const sub = parent.subcollections.find((c) => c.name === name);
    if (!sub) throw new Error(`No folder at ${path}`);
    return { id: sub.id, ...(await api.collections.contents(appId, sub.id)) };
  }

  async function reload(path: string): Promise<ContentLevel> {
    const level = await fetchLevel(path);
    levels.value = { ...levels.value, [path]: level };
    return level;
  }

  async function load(path: string): Promise<ContentLevel> {
    return levels.value[path] ?? reload(path);
  }

  // Everything is dropped, so each level refetches on its next read.
  function invalidate() {
    levels.value = {};
  }

  return {
    /** The loaded level at path, if any. */
    level: (path: string) => computed(() => levels.value[path]),
    /** One document's row, once its folder's level is loaded. */
    document: (key: string) =>
      computed(() =>
        levels.value[parentPath(key)]?.documents.find((d) => d.key === key),
      ),
    /**
     * The level at path: from state when loaded, fetched once otherwise. A
     * path with no folder at it rejects; the page maps that to a 404.
     */
    load,
    /**
     * Creates a folder in the folder at path and returns the parent's
     * reloaded level. The rest of the tree is dropped, since the sidebar
     * tree lists every level and now shows a folder it did not.
     */
    createFolder: async (path: string, name: string): Promise<ContentLevel> => {
      const parent = await load(path);
      await api.collections.create(appId, {
        body: { name, parent_id: parent.id ?? undefined },
      });
      invalidate();
      return reload(path);
    },
    /**
     * Creates a document in the folder at path and returns the folder's
     * reloaded level. Its first save gives it content; until then it is an
     * empty draft.
     */
    createFile: async (path: string, name: string): Promise<ContentLevel> => {
      const parent = await load(path);
      await api.documents.create(appId, {
        body: { name, collection_id: parent.id ?? undefined },
      });
      invalidate();
      return reload(path);
    },
  };
}

/**
 * An app's whole content tree for the sidebar, built from the store's
 * levels — the walk recurses `load`, so levels a page already fetched are
 * free and the rest are fetched once, one read per folder. One blocking
 * useAsyncData, so the sidebar never fetches after page load. Call in
 * setup.
 */
export async function useContentTree(appId: string) {
  const store = useContentStore(appId);

  async function buildNodes(path: string): Promise<ContentNode[]> {
    const nodes = contentNodes(await store.load(path), appId, path);
    await Promise.all(
      nodes
        .filter((node) => node.kind === "folder")
        .map(async (node) => {
          node.children = await buildNodes(node.path);
        }),
    );
    return nodes;
  }

  const { data: tree, refresh } = await useAsyncData(
    `content-tree-${appId}`,
    () => buildNodes(""),
  );

  return { tree, refresh };
}
