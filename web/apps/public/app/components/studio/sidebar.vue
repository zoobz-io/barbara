<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";
import Tree from "@zoobzio/foundation/components/core/tree.vue";

import { ref, useRoute } from "#imports";

import type { PageNode } from "~/types/pages";
import { useAppPages } from "~/stores/pages";
import { useDialogForm } from "~/composables/dialog-form";

const route = useRoute();
const id = String(route.params.id);

const { tree, createFolder, createFile } = await useAppPages(id);

const expanded = ref<string[]>([]);
// Document nodes navigate via their links; selection is folder highlight only.
const selected = ref<PageNode>();

const folder = useDialogForm(createFolder);
const file = useDialogForm(createFile);
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <p class="sidebar-title">Pages</p>
      <div class="sidebar-actions">
        <button type="button" class="ghost" @click="folder.open.value = true">
          + Folder
        </button>
        <button type="button" class="ghost" @click="file.open.value = true">
          + File
        </button>
      </div>
    </div>

    <p v-if="!tree?.length" class="placeholder">No pages yet.</p>
    <Tree
      v-else
      v-model="selected"
      v-model:expanded="expanded"
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

    <Dialog
      v-model:open="folder.open.value"
      title="New folder"
      description="Create a folder at the root of this app."
    >
      <form class="dialog-form" @submit.prevent="folder.submit">
        <input
          v-model="folder.name.value"
          type="text"
          placeholder="guides"
          :disabled="folder.submitting.value"
        />
        <p v-if="folder.error.value" class="error" role="alert">
          {{ folder.error.value }}
        </p>
        <div class="dialog-actions">
          <button
            type="submit"
            class="primary"
            :disabled="folder.submitting.value || !folder.name.value.trim()"
          >
            {{ folder.submitting.value ? "Creating…" : "Create" }}
          </button>
          <button
            type="button"
            class="ghost"
            @click="folder.open.value = false"
          >
            Cancel
          </button>
        </div>
      </form>
    </Dialog>

    <Dialog
      v-model:open="file.open.value"
      title="New file"
      description="Create a document at the root of this app."
    >
      <form class="dialog-form" @submit.prevent="file.submit">
        <input
          v-model="file.name.value"
          type="text"
          placeholder="install.md"
          :disabled="file.submitting.value"
        />
        <p v-if="file.error.value" class="error" role="alert">
          {{ file.error.value }}
        </p>
        <div class="dialog-actions">
          <button
            type="submit"
            class="primary"
            :disabled="file.submitting.value || !file.name.value.trim()"
          >
            {{ file.submitting.value ? "Creating…" : "Create" }}
          </button>
          <button type="button" class="ghost" @click="file.open.value = false">
            Cancel
          </button>
        </div>
      </form>
    </Dialog>
  </aside>
</template>
