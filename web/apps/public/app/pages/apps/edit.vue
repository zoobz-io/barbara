<script setup lang="ts">
import { navigateTo, ref, useRoute } from "#imports";

import { apiErrorMessage } from "~/utils/errors";
import { useApp } from "~/stores/apps";

const route = useRoute();
const id = String(route.query.id ?? "");

const { app, rename } = await useApp(id);

const name = ref(app.value?.name ?? "");
const submitting = ref(false);
const error = ref("");

async function submit() {
  const trimmed = name.value.trim();
  if (trimmed === "" || submitting.value) return;
  submitting.value = true;
  error.value = "";
  try {
    await rename(trimmed);
    await navigateTo(`/apps/${id}`);
  } catch (err) {
    error.value = apiErrorMessage(err, trimmed);
    submitting.value = false;
  }
}
</script>

<template>
  <main class="form-page">
    <h1>Edit app</h1>
    <p class="hint">Rename {{ app?.name }}.</p>

    <form @submit.prevent="submit">
      <label for="app-name">Name</label>
      <input id="app-name" v-model="name" type="text" :disabled="submitting" />
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="actions">
        <button
          type="submit"
          class="primary"
          :disabled="submitting || !name.trim()"
        >
          {{ submitting ? "Saving…" : "Save" }}
        </button>
        <NuxtLink class="cancel" :to="`/apps/${id}`">Cancel</NuxtLink>
      </div>
    </form>

    <!-- danger zone (delete app) lands here -->
  </main>
</template>
