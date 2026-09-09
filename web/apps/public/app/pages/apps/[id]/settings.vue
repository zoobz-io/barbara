<script setup lang="ts">
import { definePageMeta } from "#imports";

import General from "~/components/studio/settings/general.vue";
import Appearance from "~/components/studio/settings/appearance.vue";
import Danger from "~/components/studio/settings/danger.vue";

definePageMeta({ layout: "studio" });
</script>

<template>
  <div class="studio-page">
    <h1>Settings</h1>
    <p class="page-description">
      How this site presents itself — metadata, appearance, and the point of
      no return.
    </p>
    <General />
    <Appearance />
    <Danger />
  </div>
</template>
