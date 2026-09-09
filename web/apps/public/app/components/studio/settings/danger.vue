<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

import { navigateTo, ref, useRoute } from "#imports";

import { apiErrorMessage } from "~/utils/errors";
import { useApp } from "~/stores/apps";

const route = useRoute();
const id = String(route.params.id);

const { app, remove } = await useApp(id);

const open = ref(false);
const deleting = ref(false);
const error = ref("");

async function confirmDelete() {
  if (deleting.value) return;
  deleting.value = true;
  error.value = "";
  try {
    await remove();
    await navigateTo("/");
  } catch (err) {
    error.value = apiErrorMessage(err, app.value?.name ?? "");
    deleting.value = false;
  }
}
</script>

<template>
  <section class="panel panel-danger">
    <h2>Delete this site</h2>
    <p class="setting-description">
      Deleting {{ app?.name }} permanently removes it and everything in it.
      This cannot be undone.
    </p>
    <div class="actions">
      <button type="button" class="danger" @click="open = true">
        Delete site
      </button>
    </div>

    <Dialog
      v-model:open="open"
      title="Delete this site"
      description="This cannot be undone."
    >
      <p>
        Permanently delete <strong>{{ app?.name }}</strong
        >?
      </p>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="dialog-actions">
        <button
          type="button"
          class="danger"
          :disabled="deleting"
          @click="confirmDelete"
        >
          {{ deleting ? "Deleting…" : "Delete" }}
        </button>
        <button type="button" class="ghost" @click="open = false">
          Cancel
        </button>
      </div>
    </Dialog>
  </section>
</template>
