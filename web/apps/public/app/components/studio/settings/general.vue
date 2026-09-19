<script setup lang="ts">
import { ref, useRoute } from "#imports";

import { apiErrorMessage } from "~/utils/errors";
import { useApp } from "~/stores/apps";

const route = useRoute();
const id = String(route.params.id);

const { app, rename } = await useApp(id);

const name = ref(app.value?.name ?? "");
const submitting = ref(false);
const error = ref("");

async function save() {
  const trimmed = name.value.trim();
  if (trimmed === "" || submitting.value) return;
  submitting.value = true;
  error.value = "";
  try {
    await rename(trimmed);
  } catch (err) {
    error.value = apiErrorMessage(err, trimmed);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="panel">
    <h2>General</h2>

    <form class="settings-form" @submit.prevent="save">
      <div class="field">
        <label for="site-name">Name</label>
        <input
          id="site-name"
          v-model="name"
          type="text"
          :disabled="submitting"
        />
      </div>

      <!-- site metadata beyond the name lands with the API -->
      <div class="field">
        <label for="site-favicon">Favicon</label>
        <input
          id="site-favicon"
          type="text"
          placeholder="/favicon.ico"
          disabled
        />
      </div>
      <div class="field">
        <label for="site-logo">Logo</label>
        <input id="site-logo" type="text" placeholder="/logo.svg" disabled />
      </div>
      <div class="field">
        <label for="site-home">Home page</label>
        <input id="site-home" type="text" placeholder="index.md" disabled />
        <p class="field-hint">
          Favicon, logo and home page land with site metadata on the API.
        </p>
      </div>

      <p v-if="error" class="error" role="alert">{{ error }}</p>

      <div class="actions">
        <button
          type="submit"
          class="primary"
          :disabled="submitting || !name.trim()"
        >
          {{ submitting ? "Saving…" : "Save" }}
        </button>
      </div>
    </form>
  </section>
</template>
