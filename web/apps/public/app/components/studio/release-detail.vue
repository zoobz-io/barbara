<script setup lang="ts">
import { computed, ref, useRoute } from "#imports";

import {
  changeCounts,
  changeLabel,
  kindLabel,
  releaseCrumbs,
  releaseRoute,
  releaseTitle,
} from "~/utils/releases";
import { counted, formatDate, formatRelative, shortId } from "~/utils/format";
import { useApp } from "~/stores/apps";
import { useRelease, useReleaseChanges } from "~/stores/releases";
import { useNow } from "~/composables/clock";
import PageHeader from "~/components/studio/page-header.vue";
import ReleaseRestore from "~/components/studio/release-restore.vue";
import ReleaseTable from "~/components/studio/release-table.vue";

/**
 * One release, laid out like an asset: the header with its number and
 * Restore, then the manifest of pages it served on the left and, on the
 * right, its details and what it changed. The page has loaded the release;
 * the changes land beside it without blocking.
 */
const route = useRoute();
const id = String(route.params.id);
const releaseId = String(route.params.release);

const { app } = await useApp(id);
const { release, entries } = await useRelease(id, releaseId);
const { changes, status: changesStatus } = useReleaseChanges(id, releaseId);
const now = useNow();

const live = computed(() => app.value?.current_release_id === release.value?.id);
const crumbs = computed(() => releaseCrumbs(id, release.value));
const title = computed(() =>
  release.value ? `Release ${releaseTitle(release.value)}` : "Release",
);
const meta = computed(() => {
  const r = release.value;
  if (!r) return [];
  return [
    kindLabel(r.kind),
    `Cut ${formatRelative(r.created_at, now.value)}`,
    counted(r.entry_count, "page"),
  ];
});

// Pages the previous release served that this one does not have no row in
// the manifest, so they are listed with the changes.
const removed = computed(() =>
  changes.value.filter((c) => c.change === "removed"),
);

const restoreOpen = ref(false);
</script>

<template>
  <div class="studio-page">
    <PageHeader :crumbs="crumbs" :title="title" :meta="meta">
      <template #actions>
        <span v-if="live" class="badge">live</span>
        <button
          type="button"
          class="primary with-icon"
          :disabled="live"
          :title="live ? 'This release is live' : undefined"
          @click="restoreOpen = true"
        >
          <Icon class="f-icon" fill="currentColor" name="restore" />
          Restore
        </button>
      </template>
    </PageHeader>

    <div v-if="release" class="detail-grid">
      <ReleaseTable
        :app-id="id"
        :release-id="release.id"
        :entries="entries"
        :changes="changes"
      />

      <aside class="detail-side">
        <section class="panel">
          <h2>Details</h2>
          <table class="facts-table">
            <tbody>
              <tr>
                <th scope="row">Number</th>
                <td>{{ releaseTitle(release) }}</td>
              </tr>
              <tr>
                <th scope="row">Kind</th>
                <td>
                  <span class="kind" :class="`kind-${release.kind}`">
                    {{ kindLabel(release.kind) }}
                  </span>
                </td>
              </tr>
              <tr v-if="release.label">
                <th scope="row">Label</th>
                <td>{{ release.label }}</td>
              </tr>
              <tr v-if="release.source_release_id">
                <th scope="row">Restored</th>
                <td>
                  <NuxtLink
                    class="row-link"
                    :to="releaseRoute(id, release.source_release_id)"
                  >
                    an earlier release
                  </NuxtLink>
                </td>
              </tr>
              <tr>
                <th scope="row">Cut</th>
                <td :title="formatDate(release.created_at)">
                  {{ formatRelative(release.created_at, now) }}
                </td>
              </tr>
              <tr>
                <th scope="row">By</th>
                <td>{{ shortId(release.created_by) }}</td>
              </tr>
              <tr>
                <th scope="row">Pages</th>
                <td>{{ release.entry_count }}</td>
              </tr>
              <tr>
                <th scope="row">Status</th>
                <td>{{ live ? "Live" : "Superseded" }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section class="panel">
          <h2>Changes</h2>
          <p v-if="changeCounts(release).length === 0" class="placeholder">
            Nothing changed from the release before.
          </p>
          <table v-else class="facts-table">
            <tbody>
              <tr v-for="c in changeCounts(release)" :key="c.kind">
                <th scope="row">{{ changeLabel(c.kind) }}</th>
                <td>{{ c.count }}</td>
              </tr>
            </tbody>
          </table>
          <template v-if="removed.length">
            <h3 class="removed-title">No longer served</h3>
            <ul class="removed-list">
              <li v-for="c in removed" :key="c.document_id">
                <code>{{ c.key }}</code>
                <span v-if="c.prev_version_number" class="removed-version">
                  was v{{ c.prev_version_number }}
                </span>
              </li>
            </ul>
          </template>
          <p
            v-else-if="changesStatus === 'error'"
            class="error"
            role="alert"
          >
            Could not load this release's changes.
          </p>
        </section>
      </aside>
    </div>

    <ReleaseRestore
      v-if="release"
      v-model:open="restoreOpen"
      :app-id="id"
      :release="release"
    />
  </div>
</template>
