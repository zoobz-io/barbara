<script setup lang="ts">
import { computed, definePageMeta, ref, useRoute } from "#imports";

import { formatDate } from "~/utils/format";
import { useAppDocuments } from "~/stores/pages";

definePageMeta({ layout: "studio" });

const route = useRoute();
const id = String(route.params.id);

const { data: documents } = await useAppDocuments(id);

const query = ref("");
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  const docs = documents.value ?? [];
  return q === "" ? docs : docs.filter((d) => d.key.toLowerCase().includes(q));
});
</script>

<template>
  <div class="studio-page">
    <h1>Content</h1>
    <p class="page-description">
      Every page in this app, by path. Open one to edit it.
    </p>

    <section class="panel">
      <input
        v-model="query"
        type="text"
        class="toc-search"
        placeholder="Filter pages…"
        aria-label="Filter pages"
      />

      <p v-if="!documents?.length" class="placeholder">
        No pages yet — open the tree on any page to create one.
      </p>
      <p v-else-if="!filtered.length" class="placeholder">
        Nothing matches "{{ query }}".
      </p>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Page</th>
            <th>Status</th>
            <th>Updated</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="doc in filtered" :key="doc.id">
            <td>
              <NuxtLink class="toc-link" :to="`/apps/${id}/content/${doc.id}`">
                {{ doc.key }}
              </NuxtLink>
            </td>
            <td class="muted">{{ doc.status }}</td>
            <td class="muted">{{ formatDate(doc.updated_at) }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
