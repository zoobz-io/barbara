<script setup lang="ts">
import { navigateTo, ref } from "#imports";

import { apiErrorMessage } from "~/utils/errors";
import { useApps } from "~/stores/apps";

const { create } = await useApps();

const name = ref("");
const submitting = ref(false);
const error = ref("");

async function submit() {
  const trimmed = name.value.trim();
  if (trimmed === "" || submitting.value) return;
  submitting.value = true;
  error.value = "";
  try {
    const created = await create(trimmed);
    await navigateTo(`/apps/${created.id}`);
  } catch (err) {
    error.value = apiErrorMessage(err, trimmed);
    submitting.value = false;
  }
}
</script>

<template>
  <main class="form-page">
    <h1>Create app</h1>
    <p class="hint">
      An app is a site's document tree — name it after the site.
    </p>

    <form @submit.prevent="submit">
      <label for="app-name">Name</label>
      <input
        id="app-name"
        v-model="name"
        type="text"
        placeholder="docs-site"
        :disabled="submitting"
      />
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="actions">
        <button
          type="submit"
          class="primary"
          :disabled="submitting || !name.trim()"
        >
          {{ submitting ? "Creating…" : "Create" }}
        </button>
        <NuxtLink class="cancel" to="/">Cancel</NuxtLink>
      </div>
    </form>
  </main>
</template>
