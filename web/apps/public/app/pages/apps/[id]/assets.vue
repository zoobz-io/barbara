<script setup lang="ts">
import { definePageMeta } from "#imports";

import AssetBrowser from "~/components/studio/asset-browser.vue";

definePageMeta({ layout: "studio" });
</script>

<template>
  <div class="studio-page">
    <h1>Assets</h1>
    <p class="page-description">
      Everything uploaded to this app — images, styles, downloads — organized
      by folder.
    </p>
    <AssetBrowser />
  </div>
</template>
