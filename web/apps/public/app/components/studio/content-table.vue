<script setup lang="ts">
import { computed, useRoute } from "#imports";

import { DEFAULT_CONTENT_SORT } from "~/constants/content";
import {
  contentRoute,
  sortCollections,
  sortDocuments,
  statusLabel,
} from "~/utils/content";
import { formatDate, formatRelative, keyName } from "~/utils/format";
import { childPath, folderPath } from "~/utils/path";
import { useContentStore } from "~/stores/content";
import { useFolderView } from "~/composables/folder-view";
import { useNow } from "~/composables/clock";

/**
 * One content folder's rows as a table: subfolders first, then pages, each
 * a link to its own route, with the page's status and when it last
 * changed. A folder carries no rollups by design — no status, no date —
 * so those cells stay empty. Reads the level the page loaded into the
 * store, narrowed and ordered by the toolbar's view state, and follows
 * both as they change.
 */
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);

const store = useContentStore(id);
const level = store.level(path);
const now = useNow();
const { query, sort } = useFolderView(
  "content",
  id,
  path,
  DEFAULT_CONTENT_SORT,
);
const folders = computed(() =>
  sortCollections(level.value?.subcollections ?? [], sort.value, query.value),
);
const pages = computed(() =>
  sortDocuments(level.value?.documents ?? [], sort.value, query.value),
);
const empty = computed(() => !folders.value.length && !pages.value.length);
const searching = computed(() => query.value.trim() !== "");
</script>

<template>
  <section class="panel panel-flush">
    <table class="data-table">
      <thead>
        <tr>
          <th class="col-icon"><span class="sr-only">Type</span></th>
          <th>Name</th>
          <th class="col-status">Status</th>
          <th class="col-date">Updated</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="empty">
          <td class="placeholder-cell" colspan="4">
            {{
              searching ? `Nothing matches "${query.trim()}".` : "Empty folder"
            }}
          </td>
        </tr>
        <tr v-for="folder in folders" :key="`folder:${folder.id}`">
          <td class="col-icon">
            <Icon class="f-icon row-icon" fill="currentColor" name="folder" />
          </td>
          <td>
            <NuxtLink
              class="row-link"
              :to="contentRoute(id, childPath(path, folder.name))"
            >
              {{ folder.name }}
            </NuxtLink>
          </td>
          <td class="col-status" />
          <td class="col-date" />
        </tr>
        <tr v-for="doc in pages" :key="doc.id">
          <td class="col-icon">
            <Icon
              class="f-icon row-icon"
              fill="currentColor"
              name="file-text"
            />
          </td>
          <td>
            <NuxtLink class="row-link" :to="contentRoute(id, doc.key)">
              {{ keyName(doc.key) }}
            </NuxtLink>
          </td>
          <td class="col-status">
            <span class="status" :class="`status-${doc.status}`">
              {{ statusLabel(doc.status) }}
            </span>
          </td>
          <td class="col-date">
            <time
              :datetime="doc.updated_at"
              :title="formatDate(doc.updated_at)"
            >
              {{ formatRelative(doc.updated_at, now) }}
            </time>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
