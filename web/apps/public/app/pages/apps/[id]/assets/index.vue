<script setup lang="ts">
import type { MenuItem } from "@zoobzio/foundation/types/core/menu";

import Menu from "@zoobzio/foundation/components/core/menu.vue";

import {
  computed,
  definePageMeta,
  useAsyncData,
  useRoute,
  useRouter,
} from "#imports";

import AssetActions from "~/components/studio/asset-actions.vue";
import AssetDialogs from "~/components/studio/asset-dialogs.vue";
import AssetDropzone from "~/components/studio/asset-dropzone.vue";
import AssetStats from "~/components/studio/asset-stats.vue";
import AssetStorage from "~/components/studio/asset-storage.vue";
import AssetTable from "~/components/studio/asset-table.vue";
import AssetToolbar from "~/components/studio/asset-toolbar.vue";
import { ASSETS_LEDE } from "~/constants/assets";
import { useApps } from "~/stores/apps";
import { useAssetStore } from "~/stores/assets";
import { appMenuGroups } from "~/utils/studio";

// The assets landing page: the root of the tree, led by the app's numbers.
// Folders and assets beneath it are the catch-all sibling route. The header
// carries the stat tiles and the storage meter; below it the root behaves
// as any folder — the drop zone, then the level's rows. Keyed by full path
// so leaving a folder remounts with its level.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const router = useRouter();
const id = String(route.params.id);

// The app's name in the title is the same switcher the top bar carries;
// picking another app lands on its assets page, not its content.
const { apps } = await useApps();
const app = computed(() => apps.value.find((a) => a.id === id));
const menuGroups = computed(() => appMenuGroups(apps.value, id));

function onSwitch(item: MenuItem) {
  const target = apps.value.find((a) => a.name === item.label);
  if (target) router.push(`/apps/${target.id}/assets`);
}

const store = useAssetStore(id);
await useAsyncData(`app-assets-${id}-`, () => store.load(""));
await useAsyncData(`app-assets-stats-${id}`, () => store.loadStats());
</script>

<template>
  <div class="studio-page">
    <header class="page-header assets-header">
      <div class="page-title-row">
        <h1>
          Assets for
          <Menu :groups="menuGroups" align="start" @select="onSwitch">
            <button type="button" class="app-name" aria-label="Switch app">
              {{ app?.name ?? "this app" }}
              <Icon class="f-icon" fill="currentColor" name="chevron-down" />
            </button>
          </Menu>
        </h1>
        <AssetActions :app-id="id" path="" />
      </div>
      <p class="page-lede">{{ ASSETS_LEDE }}</p>
      <AssetStats />
      <AssetStorage />
    </header>
    <AssetDropzone :app-id="id" path="" />
    <AssetToolbar />
    <AssetTable />
    <AssetDialogs />
  </div>
</template>
