<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

const open = defineModel<boolean>("open", { required: true });
</script>

<template>
  <Dialog
    v-model:open="open"
    title="Review & Publish"
    description="Review pending changes and cut a release."
  >
    <!-- pending-changes review + release cut lands here -->
    <p class="placeholder">Review &amp; publish lands here.</p>
    <div class="dialog-actions">
      <button type="button" class="ghost" @click="open = false">Close</button>
    </div>
  </Dialog>
</template>
