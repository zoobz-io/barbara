<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

import { ref, useRouter } from "#imports";

import type { Release } from "~/types/releases";
import { releaseRoute, releaseTitle } from "~/utils/releases";
import { apiErrorMessage } from "~/utils/errors";
import { counted } from "~/utils/format";
import { useReleaseActions } from "~/stores/releases";

/**
 * The restore confirmation: cutting a new release that copies this one's
 * pages forward, with an optional label for it. On success the page moves
 * to the new release, now the live one.
 */
const { appId, release } = defineProps<{ appId: string; release: Release }>();
const open = defineModel<boolean>("open", { required: true });

const router = useRouter();
const { restore } = useReleaseActions(appId);

const label = ref("");
const submitting = ref(false);
const error = ref("");

async function submit() {
  if (submitting.value) return;
  submitting.value = true;
  error.value = "";
  try {
    const restored = await restore(release.id, label.value.trim());
    open.value = false;
    label.value = "";
    await router.push(releaseRoute(appId, restored.id));
  } catch (err) {
    error.value = apiErrorMessage(err, releaseTitle(release));
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Dialog
    v-model:open="open"
    :title="`Restore release ${releaseTitle(release)}`"
    :description="`Cuts a new release serving the ${counted(release.entry_count, 'page')} this one served. The live site changes to match; nothing is deleted.`"
  >
    <form class="dialog-form" @submit.prevent="submit">
      <input
        v-model="label"
        type="text"
        placeholder="Label (optional)"
        aria-label="Label for the new release"
        :disabled="submitting"
      />
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="dialog-actions">
        <button type="submit" class="primary" :disabled="submitting">
          {{ submitting ? "Restoring…" : "Restore" }}
        </button>
        <button type="button" class="ghost" @click="open = false">
          Cancel
        </button>
      </div>
    </form>
  </Dialog>
</template>
