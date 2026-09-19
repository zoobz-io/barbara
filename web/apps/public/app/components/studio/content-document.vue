<script setup lang="ts">
import type { DocumentContent } from "~/types/documents";
import Editor from "~/components/studio/editor.vue";
import Sidebar from "~/components/studio/sidebar.vue";

/**
 * The page at a document's path: the pages tree beside the editor. The
 * route owns the document fetch and remounts this per path; the editor
 * only edits and saves.
 */
defineProps<{
  documentId: string;
  doc: DocumentContent | null;
  refresh: () => Promise<void>;
}>();
</script>

<template>
  <div class="content-grid">
    <Sidebar />

    <section class="editor">
      <Editor :document-id="documentId" :doc="doc" :refresh="refresh" />
    </section>
  </div>
</template>
