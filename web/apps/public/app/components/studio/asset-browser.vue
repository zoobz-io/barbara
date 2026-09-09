<script setup lang="ts">
import { useBrowser } from "@zoobzio/foundation/factories/browser";

import { useRoute } from "#imports";

import type { AssetRow } from "~/types/assets";
import { ASSET_BROWSER } from "~/constants/asset-browser";
import { assetLevel } from "~/utils/assets";
import { keyName } from "~/utils/format";
import { useAssets } from "~/stores/assets";

const route = useRoute();
const id = String(route.params.id);

const store = await useAssets(id);

// The blocking useAsyncData above seeds the first projection; later widget
// fetches (folder navigation, the refresh fab) re-read through its refresh.
let seeded = false;

const browser = useBrowser(`assets-${id}`, ASSET_BROWSER, {
  fetch: async ({ path, sortField, sortDirection }) => {
    if (seeded) {
      await store.refresh();
    }
    seeded = true;

    const folderPath = path.join("/");
    const level = assetLevel(store.assets.value ?? [], folderPath);

    const files: AssetRow[] = level.files.map((asset) => ({
      ...asset,
      name: keyName(asset.key),
    }));
    if (sortField === "name" || sortField === "size") {
      const dir = sortDirection === "desc" ? -1 : 1;
      files.sort((a, b) =>
        sortField === "size"
          ? dir * (a.size - b.size)
          : dir * a.name.localeCompare(b.name),
      );
    }

    return {
      folders: level.folders.map((folder) => ({
        key: folder.name,
        label: folder.name,
        count: folder.count,
      })),
      files,
    };
  },
  actions: {
    download: (row) => {
      window.open(store.downloadUrl(row.key), "_blank");
    },
    delete: async (row) => {
      await store.remove(row.key);
      await browser.service.fetch();
    },
  },
  bulkActions: {
    delete: async (selected) => {
      await Promise.all([...selected].map((key) => store.remove(key)));
      await browser.service.fetch();
    },
  },
});
</script>

<template>
  <section class="panel">
    <component :is="browser.component" :service="browser.service" />
  </section>
</template>
