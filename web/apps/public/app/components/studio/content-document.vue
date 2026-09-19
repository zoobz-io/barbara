<script setup lang="ts">
import { computed, useAsyncData, useRoute, useTemplateRef } from "#imports";

import type { DocumentContent } from "~/types/documents";
import { CONTENT_ROOT_LABEL, VERSION_PANEL_CAP } from "~/constants/content";
import { contentCrumbs, statusLabel } from "~/utils/content";
import { formatDate, formatRelative, keyName, shortId } from "~/utils/format";
import { useDocumentStore } from "~/stores/documents";
import { useNow } from "~/composables/clock";
import Editor from "~/components/studio/editor.vue";
import PageHeader from "~/components/studio/page-header.vue";
import Sidebar from "~/components/studio/sidebar.vue";

/**
 * The page at a document's path: the pages tree beside the document, laid
 * out like an asset — the header with the path, name, and Save, then the
 * editor on the left and, on the right, the page's details and its recent
 * versions. The route owns the document fetch and remounts this per path;
 * the editor only edits and saves.
 */
const props = defineProps<{
  documentId: string;
  doc: DocumentContent | null;
  refresh: () => Promise<void>;
}>();

const route = useRoute();
const id = String(route.params.id);
const now = useNow();
const documents = useDocumentStore();

const { data: versions, refresh: refreshVersions } = await useAsyncData(
  `document-versions-${props.documentId}`,
  async () =>
    (await documents.versions(props.documentId, VERSION_PANEL_CAP)).versions,
);

// A save moves the head: the document and the versions refetch together.
async function refreshAll() {
  await Promise.all([props.refresh(), refreshVersions()]);
}

// Save lives in the header; the editor exposes its state and the action.
const editor = useTemplateRef<InstanceType<typeof Editor>>("editor");

const key = computed(() => props.doc?.document.key ?? "");
const name = computed(() => keyName(key.value));
const crumbs = computed(() => contentCrumbs(id, key.value, CONTENT_ROOT_LABEL));
const status = computed(() => props.doc?.document.status ?? "");
const head = computed(() => props.doc?.content ?? null);
</script>

<template>
  <div class="content-grid">
    <Sidebar />

    <div class="studio-page">
      <PageHeader :crumbs="crumbs" :title="name">
        <template #actions>
          <p v-if="editor?.error" class="error" role="alert">
            {{ editor.error }}
          </p>
          <button
            type="button"
            class="primary with-icon"
            :disabled="!editor?.dirty || editor?.saving"
            @click="editor?.save()"
          >
            <Icon class="f-icon" fill="currentColor" name="save" />
            {{ editor?.saving ? "Saving…" : "Save" }}
          </button>
        </template>
      </PageHeader>

      <div class="detail-grid">
        <section class="panel panel-flush document-editor">
          <Editor
            ref="editor"
            :document-id="documentId"
            :doc="doc"
            :refresh="refreshAll"
          />
        </section>

        <aside class="detail-side">
          <section class="panel">
            <h2>Details</h2>
            <table class="facts-table">
              <tbody>
                <tr>
                  <th scope="row">Name</th>
                  <td>{{ name }}</td>
                </tr>
                <tr>
                  <th scope="row">Path</th>
                  <td><code>{{ key }}</code></td>
                </tr>
                <tr>
                  <th scope="row">Status</th>
                  <td>
                    <span class="status" :class="`status-${status}`">
                      {{ statusLabel(status) }}
                    </span>
                  </td>
                </tr>
                <tr>
                  <th scope="row">Tags</th>
                  <td>{{ doc?.document.tags.join(", ") || "—" }}</td>
                </tr>
                <tr v-if="doc">
                  <th scope="row">Created</th>
                  <td :title="formatDate(doc.document.created_at)">
                    {{ formatRelative(doc.document.created_at, now) }}
                  </td>
                </tr>
                <tr v-if="doc">
                  <th scope="row">Updated</th>
                  <td :title="formatDate(doc.document.updated_at)">
                    {{ formatRelative(doc.document.updated_at, now) }}
                  </td>
                </tr>
                <tr>
                  <th scope="row">Latest</th>
                  <td>{{ head ? `v${head.version_number}` : "—" }}</td>
                </tr>
                <tr v-if="head">
                  <th scope="row">Saved</th>
                  <td :title="formatDate(head.created_at)">
                    {{ formatRelative(head.created_at, now) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </section>

          <section class="panel">
            <h2>Versions</h2>
            <p v-if="!versions?.length" class="placeholder">No versions yet.</p>
            <ol v-else class="version-list">
              <li
                v-for="version in versions"
                :key="version.id"
                :class="{ head: version.id === head?.version_id }"
              >
                <span class="version-number">v{{ version.version_number }}</span>
                <span class="version-by">{{ shortId(version.created_by) }}</span>
                <time
                  :datetime="version.created_at"
                  :title="formatDate(version.created_at)"
                >
                  {{ formatRelative(version.created_at, now) }}
                </time>
              </li>
            </ol>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>
