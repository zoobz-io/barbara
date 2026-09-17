<script setup lang="ts">
import Select from "@zoobzio/foundation/components/core/select.vue";

import { computed, useRoute } from "#imports";

import { ASSET_SORT_OPTIONS } from "~/constants/assets";
import { folderPath } from "~/utils/assets";
import { useAssetView } from "~/composables/asset-view";

/**
 * The folder's view controls, one bar: a search box that narrows the rows
 * by name and a sort select. Both write the view state the table reads,
 * so the whole level is filtered and ordered in the browser.
 */
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);
const { query, sort } = useAssetView(id, path);

const selected = computed({
  get: () => ASSET_SORT_OPTIONS.find((o) => o.value === sort.value),
  set: (option) => {
    if (option) sort.value = option.value;
  },
});
</script>

<template>
  <div class="asset-toolbar">
    <label class="asset-search">
      <Icon class="f-icon" fill="currentColor" name="search" />
      <input
        v-model="query"
        type="search"
        placeholder="Search this folder…"
        aria-label="Search this folder"
      />
    </label>
    <label class="asset-sort">
      <span class="asset-sort-label">Sort by</span>
      <Select v-model="selected" :options="ASSET_SORT_OPTIONS" />
    </label>
  </div>
</template>
