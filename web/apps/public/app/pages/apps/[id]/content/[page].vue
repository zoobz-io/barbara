<script setup lang="ts">
import { createError, definePageMeta, useAsyncData, useRoute } from "#imports";

import Sidebar from "~/components/studio/sidebar.vue";
import Editor from "~/components/studio/editor.vue";
import { useDocumentStore } from "~/stores/documents";

// Keyed by full path so navigating between documents remounts the page —
// the document fetch below blocks every navigation, never flashes.
definePageMeta({
  layout: "studio",
  key: (route) => route.fullPath,
});

const route = useRoute();
const documentId = String(route.params.page);
const store = useDocumentStore();

const { data: doc, refresh } = await useAsyncData(
  `document-${documentId}`,
  () => store.open(documentId),
);

if (!doc.value) {
  throw createError({ statusCode: 404, statusMessage: "Document not found" });
}
</script>

<template>
  <div class="content-grid">
    <Sidebar />

    <section class="editor">
      <Editor :document-id="documentId" :doc="doc ?? null" :refresh="refresh" />
    </section>
  </div>
</template>
