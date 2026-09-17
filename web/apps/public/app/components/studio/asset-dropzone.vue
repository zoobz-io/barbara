<script setup lang="ts">
import { assetIcon } from "~/utils/assets";
import { formatBytes } from "~/utils/format";
import { useAssetStore } from "~/stores/assets";
import { useAssetUpload } from "~/composables/asset-upload";

/**
 * The folder's upload section: a drop zone that uploads straight into the
 * folder, and the folder's uploads with their progress, kept until cleared.
 * Browsing for files is the header's upload button; the zone only takes
 * drops.
 */
const { appId, path } = defineProps<{ appId: string; path: string }>();

const store = useAssetStore(appId);
const uploads = store.uploadsIn(path);
const { send } = useAssetUpload(appId, path);

function onDrop(event: DragEvent) {
  send(event.dataTransfer?.files);
}
</script>

<template>
  <section class="panel panel-flush upload-panel">
    <div class="dropzone" @dragover.prevent @drop.prevent="onDrop">
      <Icon class="f-icon dropzone-icon" fill="currentColor" name="upload" />
      <p class="upload-hint">Drop files here to upload</p>
    </div>

    <table v-if="uploads.length" class="data-table upload-table">
      <tbody>
        <tr
          v-for="upload in uploads"
          :key="upload.id"
          class="upload-row"
          :class="upload.status"
        >
          <td class="col-icon">
            <Icon
              class="f-icon asset-icon"
              fill="currentColor"
              :name="assetIcon(upload.kind ?? 'file')"
            />
          </td>
          <td class="upload-name">{{ upload.name }}</td>
          <td class="upload-size">{{ formatBytes(upload.size) }}</td>
          <td
            class="upload-progress"
            :title="upload.status === 'error' ? upload.error : undefined"
          >
            <progress
              class="upload-bar"
              max="100"
              :value="upload.progress"
              :aria-label="`${upload.name} upload progress`"
            />
            <span class="upload-percent">{{ upload.progress }}%</span>
          </td>
          <td class="col-actions">
            <button
              type="button"
              class="f-button upload-close"
              :aria-label="`Clear ${upload.name}`"
              @click="store.clearUpload(upload.id)"
            >
              <Icon class="f-icon" fill="currentColor" name="close" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
