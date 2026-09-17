<script setup lang="ts">
import SiteFooter from "~/components/site-footer.vue";
</script>

<template>
  <div class="page">
    <div class="page-body">
      <slot />
    </div>
    <SiteFooter />
  </div>
</template>
