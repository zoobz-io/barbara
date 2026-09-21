<script setup lang="ts">
import { createError, definePageMeta, useRoute } from "#imports";

import ReleaseDetail from "~/components/studio/release-detail.vue";
import { useRelease } from "~/stores/releases";

// One release's page. The release loads here, blocking, so the detail never
// flashes; one that is not the app's is a 404. Keyed by full path so each
// release remounts with its own data.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const id = String(route.params.id);
const releaseId = String(route.params.release);

const { release } = await useRelease(id, releaseId);
if (!release.value) {
  throw createError({ statusCode: 404, statusMessage: "Release not found" });
}
</script>

<template>
  <ReleaseDetail />
</template>
