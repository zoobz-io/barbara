<script setup lang="ts">
import Tree from "@zoobzio/foundation/components/core/tree.vue";

import { ref, useRoute } from "#imports";

import type { ContentNode } from "~/types/content";
import { CONTENT_ROOT_LABEL } from "~/constants/content";
import { contentRoute } from "~/utils/content";
import { useContentTree } from "~/stores/content";

/**
 * The pages sidebar beside the editor: the app's whole tree, each page a
 * link to its own route. Creating folders and pages lives on the folder
 * pages, which the title links back to.
 */
const route = useRoute();
const id = String(route.params.id);

const { tree } = await useContentTree(id);

const expanded = ref<string[]>([]);
// Document nodes navigate via their links; selection is folder highlight only.
const selected = ref<ContentNode>();
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <NuxtLink class="sidebar-title" :to="contentRoute(id, '')">
        <Icon class="f-icon" fill="currentColor" name="folder-up" />
        {{ CONTENT_ROOT_LABEL }}
      </NuxtLink>
    </div>

    <p v-if="!tree?.length" class="placeholder">No pages yet.</p>
    <Tree v-else v-model="selected" v-model:expanded="expanded" :items="tree">
      <template #itemIcon="{ item, isExpanded }">
        <Icon
          class="f-icon"
          fill="currentColor"
          :name="
            item.value.kind === 'folder'
              ? isExpanded
                ? 'folder-open'
                : 'folder'
              : 'file'
          "
        />
      </template>
    </Tree>
  </aside>
</template>
