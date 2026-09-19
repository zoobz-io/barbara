<script setup lang="ts">
import { createError, definePageMeta, useAsyncData, useRoute } from "#imports";

import ContentDocument from "~/components/studio/content-document.vue";
import ContentFolder from "~/components/studio/content-folder.vue";
import { keyName } from "~/utils/format";
import { folderPath, parentPath } from "~/utils/path";
import { useContentStore } from "~/stores/content";
import { useDocumentStore } from "~/stores/documents";

// One catch-all for everything under content: /apps/:id/content/guides is
// a folder, /apps/:id/content/guides/install.md is a page in the editor.
// The parent level says which — a document it lists is the editor, a
// subcollection it lists is the folder browser, anything else is 404.
// Keyed by full path so each navigation remounts with its own data.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);

const store = useContentStore(id);
const { data: kind } = await useAsyncData(
  `app-content-kind-${id}-${path}`,
  async () => {
    // A missing ancestor rejects the load; that is a 404 too.
    const parent = await store.load(parentPath(path)).catch(() => undefined);
    if (!parent) return "missing";
    if (parent.documents.some((doc) => doc.key === path)) return "document";
    const name = keyName(path);
    if (parent.subcollections.some((sub) => sub.name === name)) return "folder";
    return "missing";
  },
);

if (kind.value === "missing") {
  throw createError({ statusCode: 404, statusMessage: "Page not found" });
}

// A folder's own level, for its header and rows.
if (kind.value === "folder") {
  await useAsyncData(`app-content-${id}-${path}`, () => store.load(path));
}

// A document's row is in the parent level loaded above; its content is
// its own read, blocking the navigation so the editor never flashes.
const documentId = store.document(path).value?.id ?? "";
const documents = useDocumentStore();
const { data: doc, refresh } = await useAsyncData(
  `document-${documentId}`,
  () => documents.open(documentId),
  { immediate: kind.value === "document" },
);

if (kind.value === "document" && !doc.value) {
  throw createError({ statusCode: 404, statusMessage: "Page not found" });
}
</script>

<template>
  <ContentDocument
    v-if="kind === 'document'"
    :document-id="documentId"
    :doc="doc ?? null"
    :refresh="refresh"
  />
  <ContentFolder v-else />
</template>
