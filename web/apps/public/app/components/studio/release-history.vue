<script setup lang="ts">
import { ref, useRoute } from "#imports";

import { useApp } from "~/stores/apps";
import { useReleases } from "~/stores/releases";
import ReleaseSection from "~/components/studio/release-section.vue";
import ReviewDialog from "~/components/studio/review-dialog.vue";

const route = useRoute();
const id = String(route.params.id);

const { app } = await useApp(id);
const { releases, hasMore, loadingMore, more } = await useReleases(id);

const reviewOpen = ref(false);
</script>

<template>
  <div class="release-history">
    <p v-if="releases.length === 0" class="panel placeholder">
      No releases yet — publish something to cut the first one.
    </p>

    <template v-else>
      <ReleaseSection
        v-for="release in releases"
        :key="release.id"
        :app-id="id"
        :release="release"
        :live="release.id === app?.current_release_id"
        @restore="reviewOpen = true"
      />

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

    <ReviewDialog v-model:open="reviewOpen" />
  </div>
</template>
