<script setup lang="ts">
import { computed, useRoute } from "#imports";

import { ASSET_ROOT_LABEL } from "~/constants/assets";
import { assetCrumbs, folderPath } from "~/utils/assets";
import { counted, formatBytes, formatRelative, keyName } from "~/utils/format";
import { useAssetStore } from "~/stores/assets";
import { useNow } from "~/composables/clock";
import AssetActions from "~/components/studio/asset-actions.vue";
import AssetDialogs from "~/components/studio/asset-dialogs.vue";
import AssetDropzone from "~/components/studio/asset-dropzone.vue";
import AssetTable from "~/components/studio/asset-table.vue";
import AssetToolbar from "~/components/studio/asset-toolbar.vue";
import PageHeader from "~/components/studio/page-header.vue";

// A folder beneath the root — the root is the landing page, with its own
// header. The page has loaded this folder's level; this reads it for the
// header.
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);
const store = useAssetStore(id);
const level = store.level(path);
const now = useNow();

const crumbs = assetCrumbs(id, path, ASSET_ROOT_LABEL);
const title = keyName(path);
// The size is the folder's direct files — a level knows nothing of what
// its subfolders hold. The updated time is the newest write anywhere
// beneath: a subfolder carries its own newest, so the maximum over the
// level is the folder's. A zero or an unknown is left out rather than shown.
const meta = computed(() => {
  const folders = level.value?.folders ?? [];
  const assets = level.value?.assets ?? [];
  const size = assets.reduce((sum, asset) => sum + asset.size, 0);
  const updated = [
    ...folders.map((f) => f.last_written_at),
    ...assets.map((a) => a.last_modified),
  ]
    .filter((t): t is string => Boolean(t))
    .sort()
    .at(-1);
  const pieces: string[] = [];
  if (folders.length) pieces.push(counted(folders.length, "folder"));
  if (assets.length) pieces.push(counted(assets.length, "file"));
  if (size) pieces.push(formatBytes(size));
  if (updated) pieces.push(`Updated ${formatRelative(updated, now.value)}`);
  return pieces;
});
</script>

<template>
  <div class="studio-page">
    <PageHeader :crumbs="crumbs" :title="title" :meta="meta">
      <template #actions>
        <AssetActions :app-id="id" :path="path" />
      </template>
    </PageHeader>
    <AssetDropzone :app-id="id" :path="path" />
    <AssetToolbar />
    <AssetTable />
    <AssetDialogs />
  </div>
</template>
