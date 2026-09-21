<script setup lang="ts">
import { createError, definePageMeta, useAsyncData, useRoute } from "#imports";

import ReleaseVersion from "~/components/studio/release-version.vue";
import { folderPath } from "~/utils/path";
import { useDocumentStore } from "~/stores/documents";
import { useRelease } from "~/stores/releases";

// The viewer for a page a release served: /apps/:id/releases/:release/
// guides/install.md is that path's entry in the release, shown at the
// version the release served. The release's manifest says whether the path
// was in it; anything else is 404. The version's content is its own read,
// blocking the navigation so the prose never flashes. Keyed by full path so
// each navigation remounts with its own data.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const id = String(route.params.id);
const releaseId = String(route.params.release);
const key = folderPath(route.params.path);

const { release, entries } = await useRelease(id, releaseId);
const entry = entries.value.find((e) => e.key === key);
if (!release.value || !entry) {
  throw createError({ statusCode: 404, statusMessage: "Page not found" });
}

const documents = useDocumentStore();
const { data } = await useAsyncData(`version-${entry.version_id}`, () =>
  documents.version(entry.version_id),
);
const version = data.value;
if (!version) {
  throw createError({ statusCode: 404, statusMessage: "Version not found" });
}
</script>

<template>
  <ReleaseVersion :release-id="releaseId" :entry="entry" :version="version" />
</template>
