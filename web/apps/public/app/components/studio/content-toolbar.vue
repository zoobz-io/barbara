<script setup lang="ts">
import Select from "@zoobzio/foundation/components/core/select.vue";

import { computed, useRoute } from "#imports";

import {
  CONTENT_SORT_OPTIONS,
  DEFAULT_CONTENT_SORT,
} from "~/constants/content";
import { folderPath } from "~/utils/path";
import { useFolderView } from "~/composables/folder-view";

/**
 * The content folder's view controls, one bar: a search box that narrows
 * the rows by name and a sort select. Both write the view state the table
 * reads, so the whole level is filtered and ordered in the browser.
 */
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);
const { query, sort } = useFolderView(
  "content",
  id,
  path,
  DEFAULT_CONTENT_SORT,
);

const selected = computed({
  get: () => CONTENT_SORT_OPTIONS.find((o) => o.value === sort.value),
  set: (option) => {
    if (option) sort.value = option.value;
  },
});
</script>

<template>
  <div class="folder-toolbar">
    <label class="folder-search">
      <Icon class="f-icon" fill="currentColor" name="search" />
      <input
        v-model="query"
        type="search"
        placeholder="Search this folder…"
        aria-label="Search this folder"
      />
    </label>
    <label class="folder-sort">
      <span class="folder-sort-label">Sort by</span>
      <Select v-model="selected" :options="CONTENT_SORT_OPTIONS" />
    </label>
  </div>
</template>
