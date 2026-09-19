<script setup lang="ts">
import { computed, ref } from "#imports";

import type { Release } from "~/types/releases";
import { RELEASE_ENTRY_CAP } from "~/constants/history";
import { contentRoute } from "~/utils/content";
import { formatDate, shortId } from "~/utils/format";
import { useReleaseEntries } from "~/stores/releases";

const { appId, release, live } = defineProps<{
  appId: string;
  release: Release;
  live: boolean;
}>();

const emit = defineEmits<{ restore: [release: Release] }>();

const { entries, status, error } = useReleaseEntries(appId, release.id);

// The live release shows every page; the rest fold past the cap.
const showAll = ref(live);
const folded = computed(
  () => !showAll.value && entries.value.length > RELEASE_ENTRY_CAP,
);
const visible = computed(() =>
  folded.value ? entries.value.slice(0, RELEASE_ENTRY_CAP) : entries.value,
);
</script>

<template>
  <section class="release-section" :class="{ live }">
    <header class="release-header">
      <div class="release-copy">
        <h2 class="release-title">
          Release #{{ release.number }}
          <span v-if="live" class="badge">live</span>
        </h2>
        <p class="release-meta">
          {{ formatDate(release.created_at) }} · by
          {{ shortId(release.created_by) }}
          <template v-if="status === 'success'">
            · {{ entries.length }} {{ entries.length === 1 ? "page" : "pages" }}
          </template>
        </p>
      </div>

      <button
        type="button"
        class="ghost"
        :disabled="live"
        @click="emit('restore', release)"
      >
        Restore
      </button>
    </header>

    <div class="release-card">
      <p v-if="status === 'pending' || status === 'idle'" class="placeholder">
        Loading pages…
      </p>
      <p v-else-if="error" class="error" role="alert">
        Could not load this release's pages.
      </p>
      <p v-else-if="entries.length === 0" class="placeholder">
        This release has no pages.
      </p>

      <template v-else>
        <table class="release-table" aria-label="Pages in this release">
          <tbody>
            <tr v-for="entry in visible" :key="entry.key">
              <td>
                <NuxtLink
                  class="release-path"
                  :to="contentRoute(appId, entry.key)"
                >
                  <Icon class="f-icon" fill="currentColor" name="file-text" />
                  {{ entry.key }}
                </NuxtLink>
              </td>
              <td class="release-col-version">
                <code>{{ shortId(entry.version_id) }}</code>
              </td>
            </tr>
          </tbody>
        </table>

        <button
          v-if="entries.length > RELEASE_ENTRY_CAP"
          type="button"
          class="ghost release-more"
          @click="showAll = !showAll"
        >
          {{
            folded
              ? `Show all ${entries.length} pages`
              : `Show first ${RELEASE_ENTRY_CAP}`
          }}
        </button>
      </template>
    </div>
  </section>
</template>
