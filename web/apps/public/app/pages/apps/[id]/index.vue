<script setup lang="ts">
import { navigateTo, useRoute } from "#imports";

import { contentRoute } from "~/utils/content";

// The app's front door is its content root; every section is a sibling
// route beneath the app, so this only forwards.
const route = useRoute();
await navigateTo(contentRoute(String(route.params.id), ""), {
  redirectCode: 302,
  replace: true,
});
</script>

<template>
  <div />
</template>
