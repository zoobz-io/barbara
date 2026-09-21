<script setup lang="ts">
import { Markdown } from "@tiptap/markdown";
import StarterKit from "@tiptap/starter-kit";
import { EditorContent, useEditor } from "@tiptap/vue-3";

import { computed, useLazyAsyncData, useRoute } from "#imports";

import type { ReleaseEntry } from "~/types/releases";
import type { Version } from "~/types/documents";
import { contentRoute } from "~/utils/content";
import { formatDate, formatRelative, keyName, shortId } from "~/utils/format";
import { releaseCrumbs, releaseTitle } from "~/utils/releases";
import { parentPath } from "~/utils/path";
import { useContentStore } from "~/stores/content";
import { useDocumentStore } from "~/stores/documents";
import { useRelease } from "~/stores/releases";
import { useNow } from "~/composables/clock";
import PageHeader from "~/components/studio/page-header.vue";
import Sidebar from "~/components/studio/sidebar.vue";

/**
 * The page a release served at a path, laid out like the editor: the pages
 * tree beside the prose, read-only, with the version's details on the
 * right. "Open in editor" takes the version to the page's editor as an
 * unsaved draft, so saving it lands a new version — the way everything is
 * restored, never in place. The route has loaded the release and the
 * version; this only shows them.
 */
const props = defineProps<{
  releaseId: string;
  entry: ReleaseEntry;
  version: Version;
}>();

const route = useRoute();
const id = String(route.params.id);
const now = useNow();

const { release } = await useRelease(id, props.releaseId);
const name = keyName(props.entry.key);
const crumbs = computed(() => releaseCrumbs(id, release.value, name));

// The read-only rendering: the same tiptap pipeline as the editor, with
// editing off, so the prose reads exactly as it would there.
const editor = useEditor({
  extensions: [StarterKit, Markdown],
  contentType: "markdown",
  content: props.version.content,
  editable: false,
});

// The page's current place in the tree, for the editor link: the document
// may have moved since this release, or been deleted — then there is no
// editor to open, and the link says so instead.
const documents = useDocumentStore();
const content = useContentStore(id);
const { data: current, status: currentStatus } = useLazyAsyncData(
  `release-entry-current-${props.entry.document_id}`,
  async () => {
    const doc = await documents.get(props.entry.document_id);
    const level = await content.load(parentPath(doc.key)).catch(() => undefined);
    const listed = level?.documents.some((d) => d.id === doc.id) ?? false;
    return { key: doc.key, listed };
  },
);
const editorLink = computed(() =>
  current.value?.listed
    ? {
        path: contentRoute(id, current.value.key),
        query: { version: props.version.id },
      }
    : undefined,
);
const moved = computed(
  () => current.value?.listed && current.value.key !== props.entry.key,
);
</script>

<template>
  <div class="content-grid">
    <Sidebar />

    <div class="studio-page">
      <PageHeader
        :crumbs="crumbs"
        :title="name"
        :meta="[`v${version.version_number}`, `as served by release ${release ? releaseTitle(release) : ''}`]"
      >
        <template #actions>
          <NuxtLink
            v-if="editorLink"
            class="primary with-icon"
            :to="editorLink"
          >
            <Icon class="f-icon" fill="currentColor" name="edit" />
            Open in editor
          </NuxtLink>
          <p
            v-else-if="currentStatus === 'success' || currentStatus === 'error'"
            class="placeholder"
          >
            This page is no longer in the content tree.
          </p>
        </template>
      </PageHeader>

      <div class="detail-grid">
        <section class="panel panel-flush document-editor version-prose">
          <div class="editor-body">
            <div class="editor-container">
              <EditorContent :editor="editor" />
            </div>
          </div>
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
                  <td><code>{{ entry.key }}</code></td>
                </tr>
                <tr v-if="moved && current">
                  <th scope="row">Now at</th>
                  <td><code>{{ current.key }}</code></td>
                </tr>
                <tr>
                  <th scope="row">Version</th>
                  <td>v{{ version.version_number }}</td>
                </tr>
                <tr>
                  <th scope="row">Saved</th>
                  <td :title="formatDate(version.created_at)">
                    {{ formatRelative(version.created_at, now) }}
                  </td>
                </tr>
                <tr>
                  <th scope="row">By</th>
                  <td>{{ shortId(version.created_by) }}</td>
                </tr>
                <tr v-if="release">
                  <th scope="row">Release</th>
                  <td>{{ releaseTitle(release) }}</td>
                </tr>
              </tbody>
            </table>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>
