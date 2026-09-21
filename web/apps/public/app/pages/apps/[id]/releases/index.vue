<script setup lang="ts">
import type { MenuItem } from "@zoobzio/foundation/types/core/menu";

import Menu from "@zoobzio/foundation/components/core/menu.vue";

import { computed, definePageMeta, useRoute, useRouter } from "#imports";

import ReleaseTimeline from "~/components/studio/release-timeline.vue";
import { RELEASES_LEDE } from "~/constants/releases";
import { releasesRoute } from "~/utils/releases";
import { appMenuGroups } from "~/utils/studio";
import { useApps } from "~/stores/apps";

// The releases landing page: the app's release timeline. Each release is
// the sibling route beneath it.
definePageMeta({ layout: "studio" });

const route = useRoute();
const router = useRouter();
const id = String(route.params.id);

// The app's name in the title is the same switcher the top bar carries;
// picking another app lands on its releases.
const { apps } = await useApps();
const app = computed(() => apps.value.find((a) => a.id === id));
const menuGroups = computed(() => appMenuGroups(apps.value, id));

function onSwitch(item: MenuItem) {
  const target = apps.value.find((a) => a.name === item.label);
  if (target) router.push(releasesRoute(target.id));
}
</script>

<template>
  <div class="studio-page">
    <header class="page-header">
      <div class="page-title-row">
        <h1>
          Releases for
          <Menu :groups="menuGroups" align="start" @select="onSwitch">
            <button type="button" class="app-name" aria-label="Switch app">
              {{ app?.name ?? "this app" }}
              <Icon class="f-icon" fill="currentColor" name="chevron-down" />
            </button>
          </Menu>
        </h1>
      </div>
      <p class="page-lede">{{ RELEASES_LEDE }}</p>
    </header>
    <ReleaseTimeline />
  </div>
</template>
