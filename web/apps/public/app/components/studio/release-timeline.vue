<script setup lang="ts">
import { useRoute } from "#imports";

import { changeCounts, changeLabel, kindLabel, releaseRoute } from "~/utils/releases";
import { counted, formatDate, formatRelative, shortId } from "~/utils/format";
import { useApp } from "~/stores/apps";
import { useReleases } from "~/stores/releases";
import { useNow } from "~/composables/clock";

/**
 * The app's releases as a timeline, newest first: each item a link to its
 * release, headlined by its number and kind, with when and who, how many
 * pages it served, and its change counts against the release before it.
 * The pages themselves are the release's own page. The live release — always
 * the newest, since restoring cuts forward — is marked.
 */
const route = useRoute();
const id = String(route.params.id);

const { app } = await useApp(id);
const { releases, hasMore, loadingMore, more } = await useReleases(id);
const now = useNow();
</script>

<template>
  <div class="release-timeline">
    <p v-if="releases.length === 0" class="panel placeholder">
      No releases yet — publish something to cut the first one.
    </p>

    <template v-else>
      <ol class="release-list">
        <li
          v-for="release in releases"
          :key="release.id"
          class="release-item"
          :class="{ live: release.id === app?.current_release_id }"
        >
          <NuxtLink class="release-card" :to="releaseRoute(id, release.id)">
            <div class="release-title">
              <span class="release-number">#{{ release.number }}</span>
              <span class="kind" :class="`kind-${release.kind}`">
                {{ kindLabel(release.kind) }}
              </span>
              <span v-if="release.id === app?.current_release_id" class="badge">
                live
              </span>
              <span v-if="release.label" class="release-label">
                {{ release.label }}
              </span>
            </div>
            <p class="release-meta">
              <time
                :datetime="release.created_at"
                :title="formatDate(release.created_at)"
              >
                {{ formatRelative(release.created_at, now) }}
              </time>
              · by {{ shortId(release.created_by) }} ·
              {{ counted(release.entry_count, "page") }}
            </p>
            <p class="release-stats">
              <span
                v-for="c in changeCounts(release)"
                :key="c.kind"
                class="change"
                :class="`change-${c.kind}`"
              >
                {{ c.count }} {{ changeLabel(c.kind).toLowerCase() }}
              </span>
              <span v-if="changeCounts(release).length === 0" class="change">
                no changes
              </span>
            </p>
          </NuxtLink>
        </li>
      </ol>

      <button
        v-if="hasMore"
        type="button"
        class="ghost release-older"
        :disabled="loadingMore"
        @click="more"
      >
        {{ loadingMore ? "Loading…" : "Show older releases" }}
      </button>
    </template>
  </div>
</template>
