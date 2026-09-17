import { computed, usePress, useRequestURL, useState } from "#imports";

import type {
  Asset,
  AssetFolder,
  AssetStats,
  AssetUpload,
} from "~/types/assets";
import { API_PROXY_PREFIX } from "~~/config/press";
import { isAssetKind, parentPath } from "~/utils/assets";

/**
 * The asset data layer for one app: every level loaded so far, keyed by
 * folder path ("" is the root), the uploads made this session, plus
 * removal and download URLs. State is the source of truth — `load` answers
 * from it when the level is already here, so revisiting a folder costs
 * nothing. Call in setup.
 */
export function useAssetStore(appId: string) {
  const api = usePress("api");
  const levels = useState<Record<string, AssetFolder>>(
    `assets-${appId}`,
    () => ({}),
  );
  const uploads = useState<AssetUpload[]>(`assets-uploads-${appId}`, () => []);
  const stats = useState<AssetStats | undefined>(
    `assets-stats-${appId}`,
    () => undefined,
  );

  const patch = (id: string, change: Partial<AssetUpload>) => {
    uploads.value = uploads.value.map((u) =>
      u.id === id ? { ...u, ...change } : u,
    );
  };

  const downloadUrl = (key: string) =>
    `${API_PROXY_PREFIX}/apps/${appId}/assets/object?key=${encodeURIComponent(key)}`;

  // The public URL is absolute — it is what an author pastes elsewhere —
  // so the request's origin fronts the proxied published read.
  const origin = useRequestURL().origin;
  const publicUrl = (key: string) =>
    `${origin}${API_PROXY_PREFIX}/published/apps/${appId}/assets/object?key=${encodeURIComponent(key)}`;

  async function reload(path: string): Promise<AssetFolder> {
    const level = await api.assets.folder(appId, { query: { path } });
    levels.value = { ...levels.value, [path]: level };
    return level;
  }

  async function reloadStats(): Promise<AssetStats> {
    const next = await api.assets.stats(appId);
    stats.value = next;
    return next;
  }

  // A write or delete changed what every level and the stats say; drop them
  // so each refetches on its next read.
  function invalidate() {
    levels.value = {};
    stats.value = undefined;
  }

  return {
    /** The loaded level at path, if any. */
    level: (path: string) => computed(() => levels.value[path]),
    /** One asset's metadata, once its folder's level is loaded. */
    asset: (key: string) =>
      computed(() =>
        levels.value[parentPath(key)]?.assets.find((a) => a.key === key),
      ),
    /** An asset's bytes as text, for the page preview. Browser-only. */
    text: async (key: string): Promise<string> => {
      const response = await fetch(downloadUrl(key));
      if (!response.ok) throw new Error(`Could not load (${response.status}).`);
      return response.text();
    },
    /** The level at path: from state when loaded, fetched once otherwise. */
    load: async (path: string) => levels.value[path] ?? reload(path),
    /** The app's asset stats, if loaded. */
    stats: computed(() => stats.value),
    /** The app's asset stats: from state when loaded, fetched once otherwise. */
    loadStats: async () => stats.value ?? reloadStats(),
    /**
     * Creates an explicit folder at path (its ancestors with it) and returns
     * its level. The parent's level and the root's are dropped, since they
     * now list a folder they did not.
     */
    createFolder: async (path: string): Promise<AssetFolder> => {
      const level = await api.assets.createFolder(appId, { body: { path } });
      levels.value = { ...levels.value, [path]: level };
      for (let parent = parentPath(path); ; parent = parentPath(parent)) {
        const { [parent]: _dropped, ...rest } = levels.value;
        levels.value = rest;
        if (parent === "") break;
      }
      return level;
    },
    /**
     * Removes an asset. Its own level reloads; every other loaded level is
     * dropped, since an ancestor's folder counts just changed and will
     * refetch on its next visit.
     */
    remove: async (path: string, key: string) => {
      await api.assets.delete(appId, { query: { key } });
      invalidate();
      await reload(path);
    },
    /** The session's uploads into one folder, newest first. */
    uploadsIn: (path: string) =>
      computed(() =>
        uploads.value.filter((u) => parentPath(u.key) === path).reverse(),
      ),
    /** Drops one upload from the list. */
    clearUpload: (id: string) => {
      uploads.value = uploads.value.filter((u) => u.id !== id);
    },
    /**
     * Stores a file at key (overwriting any asset there), tracked in the
     * uploads list with progress. The SDK encodes every body as JSON and
     * fetch reports no upload progress, so the raw bytes go by XHR straight
     * to the same proxy the download URL uses, typed with the file's own
     * media type. On success every loaded level and the stats are dropped:
     * the key's folder gained an asset and its ancestors may have gained a
     * folder, so each refetches on its next load. Browser-only.
     */
    upload: (key: string, file: File): Promise<Asset> => {
      const id = crypto.randomUUID();
      uploads.value = [
        ...uploads.value,
        {
          id,
          key,
          name: file.name,
          size: file.size,
          contentType: file.type,
          progress: 0,
          status: "uploading",
        },
      ];
      return new Promise<Asset>((resolve, reject) => {
        const fail = (message: string) => {
          patch(id, { status: "error", error: message });
          reject(new Error(message));
        };
        const xhr = new XMLHttpRequest();
        xhr.open("PUT", downloadUrl(key));
        xhr.setRequestHeader(
          "Content-Type",
          file.type || "application/octet-stream",
        );
        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            patch(id, {
              progress: Math.round((event.loaded / event.total) * 100),
            });
          }
        };
        xhr.onerror = () => fail("Upload failed — network error.");
        xhr.onload = () => {
          if (xhr.status < 200 || xhr.status > 299) {
            let message = `Upload failed (${xhr.status}).`;
            try {
              const body = JSON.parse(xhr.responseText) as { message?: string };
              if (body.message) message = body.message;
            } catch {
              // Not a JSON envelope; the status line stands.
            }
            fail(message);
            return;
          }
          const stored = JSON.parse(xhr.responseText) as Asset;
          patch(id, {
            progress: 100,
            status: "done",
            kind: isAssetKind(stored.kind) ? stored.kind : "file",
          });
          invalidate();
          resolve(stored);
        };
        xhr.send(file);
      });
    },
    /**
     * Moves an asset to a new key — another folder, another name, or both.
     * Every loaded level is dropped, since two folders changed.
     */
    move: async (key: string, newKey: string): Promise<Asset> => {
      const moved = await api.assets.move(appId, {
        query: { key },
        body: { key: newKey },
      });
      invalidate();
      return moved;
    },
    /** The proxied download URL for a key — same origin, browser-safe. */
    downloadUrl,
    /** The absolute published URL for a key — what a page references. */
    publicUrl,
  };
}
