<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

import { computed, useTemplateRef } from "#imports";

import { childPath } from "~/utils/assets";
import { useAssetStore } from "~/stores/assets";
import { useAssetUpload } from "~/composables/asset-upload";
import { useDialogForm } from "~/composables/dialog-form";

/**
 * A folder's header actions: New folder, which opens a one-field dialog and
 * creates the folder empty in place, and Upload, which opens the file
 * browser and sends the picked files into the folder. Both act on the
 * folder the page is showing.
 */
const { appId, path } = defineProps<{ appId: string; path: string }>();

const store = useAssetStore(appId);
const { send } = useAssetUpload(appId, path);
const picker = useTemplateRef<HTMLInputElement>("picker");

// Creating drops the current level (it gained a folder), so reload it
// before the dialog closes and the table reads the store again.
const folder = useDialogForm(async (name) => {
  await store.createFolder(childPath(path, name));
  await store.load(path);
});

// A folder name is one segment: the slash would make it a path.
const slashed = computed(() => folder.name.value.includes("/"));
const disabled = computed(
  () => folder.submitting.value || slashed.value || !folder.name.value.trim(),
);

function onPicked(event: Event) {
  const input = event.target as HTMLInputElement;
  send(input.files);
  input.value = "";
}
</script>

<template>
  <div class="page-actions">
    <button type="button" class="ghost" @click="folder.open.value = true">
      <Icon class="f-icon" fill="currentColor" name="folder-plus" />
      New folder
    </button>
    <button type="button" class="primary with-icon" @click="picker?.click()">
      <Icon class="f-icon" fill="currentColor" name="upload" />
      Upload
    </button>
    <input ref="picker" type="file" multiple hidden @change="onPicked" />

    <Dialog
      v-model:open="folder.open.value"
      title="New folder"
      :description="
        path ? `Create a folder inside ${path}.` : 'Create a folder at the root.'
      "
    >
      <form class="dialog-form" @submit.prevent="folder.submit">
        <input
          v-model="folder.name.value"
          type="text"
          placeholder="images"
          :disabled="folder.submitting.value"
        />
        <p v-if="slashed" class="error" role="alert">
          A folder name can't contain a slash.
        </p>
        <p v-else-if="folder.error.value" class="error" role="alert">
          {{ folder.error.value }}
        </p>
        <div class="dialog-actions">
          <button type="submit" class="primary" :disabled="disabled">
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
  </div>
</template>
