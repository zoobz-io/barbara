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

import ContentActions from "~/components/studio/content-actions.vue";
import ContentTable from "~/components/studio/content-table.vue";
import ContentToolbar from "~/components/studio/content-toolbar.vue";
import { CONTENT_LEDE } from "~/constants/content";
import { contentRoute } from "~/utils/content";
import { appMenuGroups } from "~/utils/studio";
import { useApps } from "~/stores/apps";
import { useContentStore } from "~/stores/content";

// The content landing page: the root of the tree. Folders and pages
// beneath it are the catch-all sibling route. Below the header the root
// behaves as any folder — the toolbar, then the level's rows. Keyed by
// full path so leaving a folder remounts with its level.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const router = useRouter();
const id = String(route.params.id);

// The app's name in the title is the same switcher the top bar carries;
// picking another app lands on its content root.
const { apps } = await useApps();
const app = computed(() => apps.value.find((a) => a.id === id));
const menuGroups = computed(() => appMenuGroups(apps.value, id));

function onSwitch(item: MenuItem) {
  const target = apps.value.find((a) => a.name === item.label);
  if (target) router.push(contentRoute(target.id, ""));
}

const store = useContentStore(id);
await useAsyncData(`app-content-${id}-`, () => store.load(""));
</script>

<template>
  <div class="studio-page">
    <header class="page-header content-header">
      <div class="page-title-row">
        <h1>
          Content for
          <Menu :groups="menuGroups" align="start" @select="onSwitch">
            <button type="button" class="app-name" aria-label="Switch app">
              {{ app?.name ?? "this app" }}
              <Icon class="f-icon" fill="currentColor" name="chevron-down" />
            </button>
          </Menu>
        </h1>
        <ContentActions :app-id="id" path="" />
      </div>
      <p class="page-lede">{{ CONTENT_LEDE }}</p>
    </header>
    <ContentToolbar />
    <ContentTable />
  </div>
</template>
