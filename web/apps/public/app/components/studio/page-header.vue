<script setup lang="ts">
import type { BreadcrumbItem } from "@zoobzio/foundation/types/core/breadcrumb";

import Breadcrumb from "@zoobzio/foundation/components/core/breadcrumb.vue";

/**
 * The studio page header: the path as crumbs, the title beside its actions,
 * and a line of metadata separated by dots.
 */
const { crumbs, title, meta = [] } = defineProps<{
  crumbs: BreadcrumbItem[];
  title: string;
  meta?: string[];
}>();
</script>

<template>
  <header class="page-header">
    <Breadcrumb :items="crumbs" class="crumbs" />
    <div class="page-title-row">
      <h1>{{ title }}</h1>
      <div v-if="$slots.actions" class="page-actions">
        <slot name="actions" />
      </div>
    </div>
    <p v-if="meta.length" class="page-meta">
      <template v-for="(item, i) in meta" :key="i">
        <span v-if="i > 0" class="page-meta-dot" aria-hidden="true">·</span>
        <span>{{ item }}</span>
      </template>
    </p>
  </header>
</template>
