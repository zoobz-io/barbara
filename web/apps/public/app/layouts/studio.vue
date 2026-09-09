<script setup lang="ts">
import TopBar from "~/components/studio/top-bar.vue";
</script>

<template>
  <div class="studio">
    <TopBar />
    <main class="content">
      <slot />
    </main>
  </div>
</template>
