<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

import { computed } from "#imports";

import { useContentStore } from "~/stores/content";
import { useDialogForm } from "~/composables/dialog-form";

/**
 * A content folder's header actions: New folder and New page, each a
 * one-field dialog that creates in the folder the page is showing. The
 * store reloads the folder's level before the dialog closes, so the table
 * reads the new row at once.
 */
const { appId, path } = defineProps<{ appId: string; path: string }>();

const store = useContentStore(appId);
const folder = useDialogForm((name) => store.createFolder(path, name).then());
const file = useDialogForm((name) => store.createFile(path, name).then());

// A name is one segment: the slash would make it a path.
const folderSlashed = computed(() => folder.name.value.includes("/"));
const fileSlashed = computed(() => file.name.value.includes("/"));
const where = path ? `inside ${path}` : "at the root";
</script>

<template>
  <div class="page-actions">
    <button type="button" class="ghost" @click="folder.open.value = true">
      <Icon class="f-icon" fill="currentColor" name="folder-plus" />
      New folder
    </button>
    <button
      type="button"
      class="primary with-icon"
      @click="file.open.value = true"
    >
      <Icon class="f-icon" fill="currentColor" name="file-plus" />
      New page
    </button>

    <Dialog
      v-model:open="folder.open.value"
      title="New folder"
      :description="`Create a folder ${where}.`"
    >
      <form class="dialog-form" @submit.prevent="folder.submit">
        <input
          v-model="folder.name.value"
          type="text"
          placeholder="guides"
          :disabled="folder.submitting.value"
        />
        <p v-if="folderSlashed" class="error" role="alert">
          A folder name can't contain a slash.
        </p>
        <p v-else-if="folder.error.value" class="error" role="alert">
          {{ folder.error.value }}
        </p>
        <div class="dialog-actions">
          <button
            type="submit"
            class="primary"
            :disabled="
              folder.submitting.value ||
              folderSlashed ||
              !folder.name.value.trim()
            "
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
      title="New page"
      :description="`Create a page ${where}. It opens in the editor once saved.`"
    >
      <form class="dialog-form" @submit.prevent="file.submit">
        <input
          v-model="file.name.value"
          type="text"
          placeholder="install.md"
          :disabled="file.submitting.value"
        />
        <p v-if="fileSlashed" class="error" role="alert">
          A page name can't contain a slash.
        </p>
        <p v-else-if="file.error.value" class="error" role="alert">
          {{ file.error.value }}
        </p>
        <div class="dialog-actions">
          <button
            type="submit"
            class="primary"
            :disabled="
              file.submitting.value || fileSlashed || !file.name.value.trim()
            "
          >
            {{ file.submitting.value ? "Creating…" : "Create" }}
          </button>
          <button type="button" class="ghost" @click="file.open.value = false">
            Cancel
          </button>
        </div>
      </form>
    </Dialog>
  </div>
</template>
