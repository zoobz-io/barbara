<script setup lang="ts">
import { FOOTER_LINKS } from "~/constants/footer";

const year = new Date().getFullYear();
</script>

<template>
  <footer class="site-footer">
    <div class="site-footer-inner">
      <span class="site-footer-copy">© {{ year }} Barbara</span>
      <nav class="site-footer-links" aria-label="Footer">
        <template v-for="link in FOOTER_LINKS" :key="link.label">
          <a
            v-if="link.external"
            :href="link.to"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ link.label }}
          </a>
          <NuxtLink v-else :to="link.to">{{ link.label }}</NuxtLink>
        </template>
      </nav>
    </div>
  </footer>
</template>
