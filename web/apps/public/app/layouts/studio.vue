<script setup lang="ts">
import TopBar from "~/components/studio/top-bar.vue";
import SiteFooter from "~/components/site-footer.vue";
</script>

<template>
  <div class="studio">
    <TopBar />
    <main class="content">
      <slot />
    </main>
    <SiteFooter />
  </div>
</template>
