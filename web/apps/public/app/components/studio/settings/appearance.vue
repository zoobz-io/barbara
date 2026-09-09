<script setup lang="ts">
import { APPEARANCE_GROUPS } from "~/constants/appearance";
import { useAppearance } from "~/composables/appearance";

const appearance = useAppearance();
</script>

<template>
  <section class="panel">
    <h2>Appearance</h2>

    <div
      v-for="group in APPEARANCE_GROUPS"
      :key="group.label"
      class="appearance-group"
    >
      <h3>{{ group.label }}</h3>

      <div
        v-for="setting in group.settings"
        :key="setting.modifier"
        class="setting-row"
      >
        <div class="setting-copy">
          <p class="setting-label">{{ setting.label }}</p>
          <p class="setting-description">{{ setting.description }}</p>
        </div>

        <div class="segmented" role="group" :aria-label="setting.label">
          <button
            v-for="option in setting.options"
            :key="option.context"
            type="button"
            class="segment"
            :class="{
              active: appearance.active(setting.modifier) === option.context,
            }"
            :aria-pressed="
              appearance.active(setting.modifier) === option.context
            "
            @click="appearance.set(setting.modifier, option.context)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
