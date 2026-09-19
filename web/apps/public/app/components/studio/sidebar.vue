<script setup lang="ts">
import Tree from "@zoobzio/foundation/components/core/tree.vue";

import { computed, useRoute, useState } from "#imports";

import { ancestorKeys, findNode } from "~/utils/content";
import { folderPath } from "~/utils/path";
import { useContentTree } from "~/stores/content";

/**
 * The pages sidebar beside the editor: the app's whole tree, each page a
 * link to its own route. Its view state outlives the page: which folders
 * are open is kept per app, so a folder opened on one page is still open
 * on the next, and the page at the route is the selected node — a folder
 * click toggles the folder but never takes the selection.
 */
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);

const { tree } = await useContentTree(id);

// Open folders persist per app across navigations; the current page's
// ancestors join them so the selected page is never folded away.
const expanded = useState<string[]>(`content-tree-expanded-${id}`, () => []);
expanded.value = [
  ...new Set([...expanded.value, ...ancestorKeys(tree.value ?? [], path)]),
];

const selected = computed(() => findNode(tree.value ?? [], path));
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-scroll">
      <p v-if="!tree?.length" class="placeholder">No pages yet.</p>
      <Tree
        v-else
        v-model:expanded="expanded"
        :model-value="selected"
        :items="tree"
      >
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
    </div>
  </aside>
</template>
