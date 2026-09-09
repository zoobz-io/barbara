<script setup lang="ts">
import type { MenuItem } from "@zoobzio/foundation/types/core/menu";

import Menu from "@zoobzio/foundation/components/core/menu.vue";

import ReviewDialog from "~/components/studio/review-dialog.vue";

import { computed, ref, useRoute, useRouter } from "#imports";

import { ALL_APPS_LABEL } from "~/constants/studio";
import { appMenuGroups, appTabs, tabActive } from "~/utils/studio";
import { useApps } from "~/stores/apps";

const route = useRoute();
const router = useRouter();

// Reactive: the layout persists across app switches.
const id = computed(() => String(route.params.id));

const { apps } = await useApps();

const current = computed(() => apps.value.find((a) => a.id === id.value));
const menuGroups = computed(() => appMenuGroups(apps.value, id.value));
const tabs = computed(() => appTabs(id.value));

function onMenuSelect(item: MenuItem) {
  if (item.label === ALL_APPS_LABEL) {
    router.push("/");
    return;
  }
  const target = apps.value.find((a) => a.name === item.label);
  if (target) router.push(`/apps/${target.id}`);
}

const reviewOpen = ref(false);
</script>

<template>
  <header class="topbar">
    <Menu
      :label="current?.name ?? '…'"
      :groups="menuGroups"
      align="start"
      @select="onMenuSelect"
    />

    <nav class="tabs" aria-label="App sections">
      <NuxtLink
        v-for="tab in tabs"
        :key="tab.to"
        class="tab"
        :class="{ active: tabActive(tab, route.path) }"
        :to="tab.to"
      >
        {{ tab.label }}
      </NuxtLink>
    </nav>

    <span class="spacer" />

    <button type="button" class="primary" @click="reviewOpen = true">
      Review &amp; Publish
    </button>

    <ReviewDialog v-model:open="reviewOpen" />
  </header>
</template>
