<script setup lang="ts">
import { computed, useRoute } from "#imports";

import {
  ASSET_STORAGE_LIMIT_BYTES,
  STORAGE_WARN_SHARE,
} from "~/constants/assets";
import { storageShares } from "~/utils/assets";
import { formatBytes } from "~/utils/format";
import { useAssetStore } from "~/stores/assets";

/**
 * What the app's storage holds, as one bar: a segment per kind for the
 * largest kinds, one for the rest, and the free space to the limit as the
 * unfilled track. Each kind is at least a visible tick, so a small app's
 * breakdown still reads; the track absorbs the difference. The legend
 * beneath names every segment with its size,
 * so nothing is color-alone, and the head states the space left. Shares of
 * what is used live in each segment's tooltip. Reads the stats the page
 * loaded into the store.
 */
const route = useRoute();
const id = String(route.params.id);
const store = useAssetStore(id);
const stats = store.stats;

const shares = computed(() =>
  storageShares(stats.value?.kinds ?? [], ASSET_STORAGE_LIMIT_BYTES),
);
const nearLimit = computed(
  () => shares.value.used / shares.value.limit >= STORAGE_WARN_SHARE,
);
// A sliver reads as "<1%" rather than a rounded zero beside a real size.
const percent = (fraction: number) =>
  fraction > 0 && fraction < 0.005 ? "<1%" : `${Math.round(fraction * 100)}%`;
const width = (share: number) => `${(share * 100).toFixed(2)}%`;
</script>

<template>
  <section class="storage" aria-labelledby="storage-title">
    <div class="storage-head">
      <h2 id="storage-title">Storage</h2>
      <span class="storage-summary" :class="{ warn: nearLimit }">
        {{ formatBytes(shares.free) }} free of
        {{ formatBytes(shares.limit) }}
      </span>
    </div>

    <div
      class="storage-bar"
      role="img"
      :aria-label="`${formatBytes(shares.used)} of ${formatBytes(shares.limit)} used`"
    >
      <span
        v-for="seg in shares.segments"
        :key="seg.key"
        class="storage-segment"
        :class="`slot-${seg.slot}`"
        :style="{ width: width(seg.share) }"
        :title="`${seg.label} · ${formatBytes(seg.size)} · ${percent(seg.fraction)} of used`"
      />
      <span
        v-if="shares.freeShare > 0"
        class="storage-segment slot-free"
        :title="`Free · ${formatBytes(shares.free)}`"
      />
    </div>

    <ul class="storage-legend">
      <li v-for="seg in shares.segments" :key="seg.key" :title="seg.detail">
        <span class="legend-swatch" :class="`slot-${seg.slot}`" aria-hidden="true" />
        <span class="legend-label">{{ seg.label }}</span>
        <span class="legend-value">{{ formatBytes(seg.size) }}</span>
      </li>
      <li v-if="!shares.segments.length" class="legend-empty">
        Nothing stored yet.
      </li>
    </ul>
  </section>
</template>
