<script setup lang="ts">
import { computed, useRoute } from "#imports";

import { ASSET_STORAGE_LIMIT_BYTES } from "~/constants/assets";
import { writtenWithin } from "~/utils/assets";
import { formatBytes, formatRelative } from "~/utils/format";
import { useAssetStore } from "~/stores/assets";
import { useNow } from "~/composables/clock";

/**
 * The headline numbers of the app's assets as a row of stat tiles: how many
 * files, how much of the storage limit they use, how many were written this
 * week, and when the last one landed. Reads the stats the page loaded into
 * the store.
 */
const route = useRoute();
const id = String(route.params.id);
const store = useAssetStore(id);
const stats = store.stats;
const now = useNow();

const tiles = computed(() => {
  const s = stats.value;
  return [
    {
      key: "count",
      label: "Files",
      value: (s?.count ?? 0).toLocaleString("en-US"),
      note: "",
    },
    {
      key: "size",
      label: "Storage used",
      value: formatBytes(s?.size ?? 0),
      note: `of ${formatBytes(ASSET_STORAGE_LIMIT_BYTES)}`,
    },
    {
      key: "week",
      label: "Uploaded this week",
      value: writtenWithin(s?.days ?? [], 7, now.value).toLocaleString("en-US"),
      note: "",
    },
    {
      key: "last",
      label: "Last upload",
      value: s?.last_written_at ? formatRelative(s.last_written_at, now.value) : "—",
      note: "",
    },
  ];
});
</script>

<template>
  <ul class="stat-tiles">
    <li v-for="tile in tiles" :key="tile.key" class="stat-tile">
      <span class="stat-label">{{ tile.label }}</span>
      <span class="stat-value">{{ tile.value }}</span>
      <span v-if="tile.note" class="stat-note">{{ tile.note }}</span>
    </li>
  </ul>
</template>
