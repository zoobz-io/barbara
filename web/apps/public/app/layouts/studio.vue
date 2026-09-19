<script setup lang="ts">
import { onBeforeUnmount, onMounted, useTemplateRef } from "#imports";

import TopBar from "~/components/studio/top-bar.vue";
import SiteFooter from "~/components/site-footer.vue";

// The bar's height is the sidebar's sticky offset. It is measured live into
// a variable on the shell, so the two never drift as the bar's rows change.
const shell = useTemplateRef<HTMLElement>("shell");
const bar = useTemplateRef<InstanceType<typeof TopBar>>("bar");
let observer: ResizeObserver | undefined;

onMounted(() => {
  const el = bar.value?.$el as HTMLElement | undefined;
  if (!el || !shell.value) return;
  observer = new ResizeObserver(() => {
    shell.value?.style.setProperty("--topbar-height", `${el.offsetHeight}px`);
  });
  observer.observe(el);
});

onBeforeUnmount(() => observer?.disconnect());
</script>

<template>
  <div ref="shell" class="studio">
    <TopBar ref="bar" />
    <main class="content">
      <slot />
    </main>
    <SiteFooter />
  </div>
</template>
