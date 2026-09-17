<script setup lang="ts">
import type {
  MenuGroup,
  MenuItem,
} from "@zoobzio/foundation/types/core/menu";

import Menu from "@zoobzio/foundation/components/core/menu.vue";

import { computed, useRoute, useRouter } from "#imports";

import { useAppearance } from "~/composables/appearance";
import { MODE_LABEL, SIGN_OUT_LABEL } from "~/constants/studio";
import { appMenuGroups, appTabs, tabActive } from "~/utils/studio";
import { useApps } from "~/stores/apps";

const route = useRoute();
const router = useRouter();

// Reactive: the layout persists across app switches.
const id = computed(() => String(route.params.id));

const { apps } = await useApps();
const appearance = useAppearance();

const current = computed(() => apps.value.find((a) => a.id === id.value));
const menuGroups = computed(() => appMenuGroups(apps.value, id.value));
const tabs = computed(() => appTabs(id.value));

function onMenuSelect(item: MenuItem) {
  const target = apps.value.find((a) => a.name === item.label);
  if (target) router.push(`/apps/${target.id}`);
}

// The user menu: the mode toggle names the mode it switches to.
const dark = computed(() => appearance.active("color") === "dark");
const userGroups = computed<MenuGroup[]>(() => [
  {
    key: "prefs",
    items: [
      {
        label: dark.value ? MODE_LABEL.light : MODE_LABEL.dark,
        icon: dark.value ? "sun" : "moon",
      },
    ],
  },
  {
    key: "session",
    items: [{ label: SIGN_OUT_LABEL, icon: "log-out" }],
  },
]);

function onUserSelect(item: MenuItem) {
  if (item.label === MODE_LABEL.light || item.label === MODE_LABEL.dark) {
    appearance.set("color", dark.value ? "light" : "dark");
    return;
  }
  if (item.label === SIGN_OUT_LABEL) {
    // Sign-out lands with auth on the API; until then, leave the studio.
    router.push("/");
  }
}
</script>

<template>
  <header class="topbar">
    <div class="topbar-row">
      <NuxtLink class="brand" to="/" aria-label="Barbara — all apps">
        <span class="brand-mark" aria-hidden="true">B</span>
        <span class="brand-name">Barbara</span>
      </NuxtLink>

      <Menu :groups="menuGroups" align="start" @select="onMenuSelect">
        <button type="button" class="f-button app-switcher">
          <span class="app-switcher-name">{{ current?.name ?? "…" }}</span>
          <Icon class="f-icon" fill="currentColor" name="chevron-down" />
        </button>
      </Menu>

      <span class="spacer" />

      <Menu :groups="userGroups" align="end" @select="onUserSelect">
        <button type="button" class="user-menu" aria-label="Account menu">
          <!-- The avatar shell only: swap for foundation's Avatar once
               accounts carry an image. -->
          <span class="f-avatar-root">
            <Icon class="f-icon" fill="currentColor" name="user" />
          </span>
        </button>
      </Menu>
    </div>

    <nav class="tabs" aria-label="App sections">
      <NuxtLink
        v-for="tab in tabs"
        :key="tab.to"
        class="tab"
        :class="{ active: tabActive(tab, route.path) }"
        :to="tab.to"
      >
        <Icon class="f-icon" fill="currentColor" :name="tab.icon" />
        {{ tab.label }}
      </NuxtLink>
    </nav>
  </header>
</template>
