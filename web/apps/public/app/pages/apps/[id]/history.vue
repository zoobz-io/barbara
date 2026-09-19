<script setup lang="ts">
import { definePageMeta } from "#imports";

import ReleaseHistory from "~/components/studio/release-history.vue";

definePageMeta({ layout: "studio" });
</script>

<template>
  <div class="studio-page">
    <h1>History</h1>
    <ReleaseHistory />
  </div>
</template>
