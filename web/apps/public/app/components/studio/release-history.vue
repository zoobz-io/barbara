<script setup lang="ts">
import { ref, useRoute } from "#imports";

import { formatDate, shortId } from "~/utils/format";
import { useApp } from "~/stores/apps";
import { useReleases } from "~/stores/releases";
import ReviewDialog from "~/components/studio/review-dialog.vue";

const route = useRoute();
const id = String(route.params.id);

const { app } = await useApp(id);
const { releases } = await useReleases(id);

const reviewOpen = ref(false);
</script>

<template>
  <section class="panel">
    <p v-if="releases.length === 0" class="placeholder">
      No releases yet — publish something to cut the first one.
    </p>

    <ul v-else class="release-list">
      <li v-for="release in releases" :key="release.id" class="release-row">
        <div class="release-copy">
          <p class="release-title">
            Release #{{ release.number }}
            <span
              v-if="release.id === app?.current_release_id"
              class="badge"
            >
              live
            </span>
          </p>
          <p class="release-meta">
            {{ formatDate(release.created_at) }} · by
            {{ shortId(release.created_by) }}
          </p>
        </div>

        <div class="release-actions">
          <button
            type="button"
            class="ghost"
            :disabled="release.id === app?.current_release_id"
            @click="reviewOpen = true"
          >
            Restore
          </button>
        </div>
      </li>
    </ul>

    <ReviewDialog v-model:open="reviewOpen" />
  </section>
</template>
