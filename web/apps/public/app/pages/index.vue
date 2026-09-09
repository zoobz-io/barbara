<script setup lang="ts">
import { useApps } from "~/stores/apps";

const { apps } = await useApps();
</script>

<template>
  <main class="landing">
    <header class="hero">
      <div class="hero-copy">
        <h1>Barbara</h1>
        <p>
          Markdown documents as a service — versioned, published, and searchable
          through one API.
        </p>
      </div>
      <div class="hero-action">
        <NuxtLink class="primary" to="/apps/create">Create app</NuxtLink>
      </div>
    </header>

    <section class="apps" aria-label="Your apps">
      <p v-if="apps.length === 0" class="placeholder">
        No apps yet — create one to get started.
      </p>
      <ul v-else class="app-list">
        <li v-for="app in apps" :key="app.id">
          <NuxtLink class="app-item" :to="`/apps/${app.id}`">
            <span class="app-name">{{ app.name }}</span>
            <span class="app-meta">
              {{ app.current_release_id ? "live" : "no release" }}
            </span>
          </NuxtLink>
        </li>
      </ul>
    </section>
  </main>
</template>
