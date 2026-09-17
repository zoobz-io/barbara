<script setup lang="ts">
import { definePageMeta } from "#imports";

import General from "~/components/studio/settings/general.vue";
import Brand from "~/components/studio/settings/brand.vue";
import Appearance from "~/components/studio/settings/appearance.vue";
import Danger from "~/components/studio/settings/danger.vue";

definePageMeta({ layout: "studio" });
</script>

<template>
  <div class="studio-page">
    <h1>Settings</h1>
    <General />
    <Brand />
    <Appearance />
    <Danger />
  </div>
</template>
