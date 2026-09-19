<script setup lang="ts">
import { createError, definePageMeta, useAsyncData, useRoute } from "#imports";

import AssetDetail from "~/components/studio/asset-detail.vue";
import AssetFolder from "~/components/studio/asset-folder.vue";
import { keyName } from "~/utils/format";
import { folderPath, parentPath } from "~/utils/path";
import { useAssetStore } from "~/stores/assets";

// One catch-all for everything under assets: /apps/:id/assets/images is a
// folder, /apps/:id/assets/images/logo.png is an asset. The parent level
// says which — an asset it lists is the asset page, a folder it lists is
// the browser, anything else is 404. Keyed by full path so each navigation
// remounts with its own level loaded.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);

const store = useAssetStore(id);
const { data: kind } = await useAsyncData(
  `app-assets-kind-${id}-${path}`,
  async () => {
    const parent = await store.load(parentPath(path));
    const name = keyName(path);
    if (parent.assets.some((asset) => asset.key === path)) return "asset";
    if (parent.folders.some((folder) => folder.name === name)) return "folder";
    return "missing";
  },
);

if (kind.value === "missing") {
  throw createError({ statusCode: 404, statusMessage: "Asset not found" });
}

// A folder's own level, for its header and rows. An asset's metadata is in
// the parent level already loaded above.
if (kind.value === "folder") {
  await useAsyncData(`app-assets-${id}-${path}`, () => store.load(path));
}
</script>

<template>
  <AssetDetail v-if="kind === 'asset'" />
  <AssetFolder v-else />
</template>
