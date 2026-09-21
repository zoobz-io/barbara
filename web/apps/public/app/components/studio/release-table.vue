<script setup lang="ts">
import { computed } from "#imports";

import type { ReleaseChange, ReleaseEntry } from "~/types/releases";
import { changeByDocument, changeLabel, releaseEntryRoute } from "~/utils/releases";
import { keyName } from "~/utils/format";

/**
 * A release's manifest as a table: every page it served, by path, each a
 * link to the viewer for the version it served, with the version number
 * and — once the changes land — how the page differed from the release
 * before. A page that was not in the previous release, or was at another
 * path or version, is marked; an unchanged one is not.
 */
const { appId, releaseId, entries, changes } = defineProps<{
  appId: string;
  releaseId: string;
  entries: ReleaseEntry[];
  changes: ReleaseChange[];
}>();

const marks = computed(() => changeByDocument(changes));
</script>

<template>
  <section class="panel panel-flush">
    <table class="data-table" aria-label="Pages in this release">
      <thead>
        <tr>
          <th class="col-icon"><span class="sr-only">Type</span></th>
          <th>Page</th>
          <th class="col-status">Change</th>
          <th class="col-size">Version</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="entries.length === 0">
          <td class="placeholder-cell" colspan="4">
            This release served no pages.
          </td>
        </tr>
        <tr v-for="entry in entries" :key="entry.key">
          <td class="col-icon">
            <Icon class="f-icon row-icon" fill="currentColor" name="file-text" />
          </td>
          <td>
            <NuxtLink
              class="row-link"
              :to="releaseEntryRoute(appId, releaseId, entry.key)"
            >
              {{ keyName(entry.key) }}
            </NuxtLink>
            <span class="row-path">{{ entry.key }}</span>
          </td>
          <td class="col-status">
            <span
              v-if="marks.get(entry.document_id)"
              class="change"
              :class="`change-${marks.get(entry.document_id)?.change}`"
            >
              {{ changeLabel(marks.get(entry.document_id)?.change ?? "") }}
            </span>
          </td>
          <td class="col-size">
            <code>v{{ entry.version_number }}</code>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
