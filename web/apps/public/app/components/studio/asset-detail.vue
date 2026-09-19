<script setup lang="ts">
import Menu from "@zoobzio/foundation/components/core/menu.vue";

import { computed, ref, useLazyAsyncData, useRoute } from "#imports";

import { ASSET_ACTIONS, ASSET_ROOT_LABEL } from "~/constants/assets";
import {
  assetCrumbs,
  assetIcon,
  assetPreview,
  kindLabel,
} from "~/utils/assets";
import {
  formatBytes,
  formatDate,
  formatRelative,
  keyName,
} from "~/utils/format";
import { folderPath } from "~/utils/path";
import { useAssetStore } from "~/stores/assets";
import { useAssetMenu } from "~/composables/asset-menu";
import { useNow } from "~/composables/clock";
import AssetDialogs from "~/components/studio/asset-dialogs.vue";
import PageHeader from "~/components/studio/page-header.vue";

const route = useRoute();
const id = String(route.params.id);
const key = folderPath(route.params.path);

// The page has loaded the folder's level, which carries this asset.
const store = useAssetStore(id);
const asset = store.asset(key);
const now = useNow();

const name = keyName(key);
const crumbs = assetCrumbs(id, key, ASSET_ROOT_LABEL);
const preview = computed(() =>
  assetPreview(asset.value?.kind ?? "", asset.value?.content_type ?? ""),
);
const url = store.downloadUrl(key);

const publicUrl = store.publicUrl(key);
const menu = useAssetMenu(id);

// Copy flashes a check for a moment, then returns to the copy glyph.
const copied = ref(false);
async function copy() {
  await navigator.clipboard.writeText(publicUrl);
  copied.value = true;
  setTimeout(() => {
    copied.value = false;
  }, 1500);
}

// The details table: each row a label and the value the API gave.
const details = computed(() => {
  const a = asset.value;
  const rows: { label: string; value: string; title?: string; mono?: boolean }[] = [
    { label: "Name", value: name },
    { label: "Key", value: key, mono: true },
    { label: "Type", value: a?.content_type ?? "", mono: true },
    { label: "Kind", value: kindLabel(a?.kind ?? "") },
    { label: "Size", value: formatBytes(a?.size ?? 0) },
  ];
  if (a?.last_modified) {
    rows.push({
      label: "Modified",
      value: formatRelative(a.last_modified, now.value),
      title: formatDate(a.last_modified),
    });
  }
  return rows;
});

// Text previews read the bytes in the browser after the page lands.
const { data: text, status: textStatus } = useLazyAsyncData(
  `app-asset-text-${id}-${key}`,
  () => store.text(key),
  { server: false, immediate: preview.value === "text" },
);

</script>

<template>
  <div class="studio-page">
    <PageHeader :crumbs="crumbs" :title="name">
      <template #actions>
        <Menu
          :groups="ASSET_ACTIONS"
          align="end"
          @select="menu.select(key, $event)"
        >
          <button type="button" class="f-button" aria-label="Actions">
            <Icon class="f-icon" fill="currentColor" name="actions" />
          </button>
        </Menu>
      </template>
    </PageHeader>

    <div class="detail-grid">
      <section class="panel panel-flush asset-preview">
        <img v-if="preview === 'image'" :src="url" :alt="name" />
        <iframe v-else-if="preview === 'pdf'" :src="url" :title="name" />
        <video v-else-if="preview === 'video'" :src="url" controls />
        <audio v-else-if="preview === 'audio'" :src="url" controls />
        <template v-else-if="preview === 'text'">
          <p v-if="textStatus === 'pending' || textStatus === 'idle'" class="placeholder">
            Loading…
          </p>
          <p v-else-if="textStatus === 'error'" class="error" role="alert">
            Could not load this file.
          </p>
          <pre v-else class="asset-code"><code>{{ text }}</code></pre>
        </template>
        <div v-else class="asset-no-preview">
          <Icon
            class="f-icon"
            fill="currentColor"
            :name="assetIcon(asset?.kind ?? '')"
          />
          <p class="placeholder">No preview for this file type.</p>
        </div>
      </section>

      <aside class="detail-side">
        <section class="panel asset-url">
          <h2>Public URL</h2>
          <div class="asset-url-row">
            <input
              type="text"
              readonly
              :value="publicUrl"
              aria-label="Public URL"
              @focus="($event.target as HTMLInputElement).select()"
            />
            <button
              type="button"
              class="f-button"
              :aria-label="copied ? 'Copied' : 'Copy URL'"
              :title="copied ? 'Copied' : 'Copy URL'"
              @click="copy"
            >
              <Icon
                class="f-icon"
                fill="currentColor"
                :name="copied ? 'check' : 'copy'"
              />
            </button>
            <a
              class="f-button"
              :href="url"
              :download="name"
              aria-label="Download"
              title="Download"
            >
              <Icon class="f-icon" fill="currentColor" name="download" />
            </a>
          </div>
        </section>

        <section class="panel asset-facts">
          <h2>Details</h2>
          <table class="facts-table">
            <tbody>
              <tr v-for="row in details" :key="row.label">
                <th scope="row">{{ row.label }}</th>
                <td :title="row.title">
                  <code v-if="row.mono">{{ row.value }}</code>
                  <template v-else>{{ row.value }}</template>
                </td>
              </tr>
            </tbody>
          </table>
        </section>
      </aside>
    </div>
    <AssetDialogs />
  </div>
</template>
