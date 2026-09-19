<script setup lang="ts">
import { computed, useRoute } from "#imports";

import { CONTENT_ROOT_LABEL } from "~/constants/content";
import { contentCrumbs } from "~/utils/content";
import { counted, formatRelative, keyName } from "~/utils/format";
import { folderPath } from "~/utils/path";
import { useContentStore } from "~/stores/content";
import { useNow } from "~/composables/clock";
import ContentActions from "~/components/studio/content-actions.vue";
import ContentTable from "~/components/studio/content-table.vue";
import ContentToolbar from "~/components/studio/content-toolbar.vue";
import PageHeader from "~/components/studio/page-header.vue";

// A folder beneath the root — the root is the landing page, with its own
// header. The page has loaded this folder's level; this reads it for the
// header.
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);
const store = useContentStore(id);
const level = store.level(path);
const now = useNow();

const crumbs = contentCrumbs(id, path, CONTENT_ROOT_LABEL);
const title = keyName(path);
// The counts are the folder's direct rows, and the updated time is the
// newest change among its direct pages — a level knows nothing of what
// its subfolders hold, by design. Nothing is shown when there is nothing.
const meta = computed(() => {
  const folders = level.value?.subcollections ?? [];
  const pages = level.value?.documents ?? [];
  const updated = pages
    .map((d) => d.updated_at)
    .sort()
    .at(-1);
  const pieces: string[] = [];
  if (folders.length) pieces.push(counted(folders.length, "folder"));
  if (pages.length) pieces.push(counted(pages.length, "page"));
  if (updated) pieces.push(`Updated ${formatRelative(updated, now.value)}`);
  return pieces;
});
</script>

<template>
  <div class="studio-page">
    <PageHeader :crumbs="crumbs" :title="title" :meta="meta">
      <template #actions>
        <ContentActions :app-id="id" :path="path" />
      </template>
    </PageHeader>
    <ContentToolbar />
    <ContentTable />
  </div>
</template>
