<script setup lang="ts">
import { BRAND_GROUPS } from "~/constants/appearance";
import { useAppearance } from "~/composables/appearance";
import type { BrandRamp } from "~/types/appearance";
import { isHex } from "~/utils/ramps";

const appearance = useAppearance();

const anyBranded = () =>
  BRAND_GROUPS.some((group) =>
    group.settings.some((setting) => appearance.branded(setting.ramp)),
  );

function pick(ramp: BrandRamp, event: Event) {
  const value = (event.target as HTMLInputElement).value;
  if (isHex(value)) {
    appearance.brand(ramp, value);
  }
}
</script>

<template>
  <section class="panel">
    <div class="panel-heading">
      <h2>Brand</h2>
      <button
        type="button"
        class="ghost"
        :disabled="!anyBranded()"
        @click="appearance.resetBrand()"
      >
        Reset all
      </button>
    </div>
    <p class="panel-description">
      Pick a colour per role and every shade in that family — fills, text,
      containers, muted and vivid — is recalculated to match, in light and
      dark.
    </p>

    <div
      v-for="group in BRAND_GROUPS"
      :key="group.label"
      class="appearance-group"
    >
      <h3>{{ group.label }}</h3>

      <div
        v-for="setting in group.settings"
        :key="setting.ramp"
        class="setting-row"
      >
        <div class="setting-copy">
          <p class="setting-label">{{ setting.label }}</p>
          <p class="setting-description">{{ setting.description }}</p>
        </div>

        <div class="brand-control">
          <div class="brand-swatches" aria-hidden="true">
            <span
              class="brand-swatch"
              :style="{
                background: `var(--${setting.swatch.fill})`,
                color: `var(--${setting.swatch.on})`,
              }"
              >Aa</span
            >
            <span
              class="brand-swatch"
              :style="{
                background: `var(--${setting.swatch.container})`,
                color: `var(--${setting.swatch.onContainer})`,
              }"
              >Aa</span
            >
          </div>

          <label class="brand-picker" :title="setting.label">
            <input
              type="color"
              :value="appearance.seed(setting.ramp)"
              :aria-label="`${setting.label} colour`"
              @input="pick(setting.ramp, $event)"
            />
            <code>{{ appearance.seed(setting.ramp) }}</code>
          </label>

          <button
            type="button"
            class="ghost brand-clear"
            :disabled="!appearance.branded(setting.ramp)"
            :aria-label="`Reset ${setting.label}`"
            title="Reset to theme default"
            @click="appearance.unbrand(setting.ramp)"
          >
            ×
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
