<script setup lang="ts">
import Dialog from "@zoobzio/foundation/components/core/dialog.vue";

import { computed, ref, useRoute, useRouter, watch } from "#imports";

import type { AssetSubfolder } from "~/types/assets";
import { ASSET_ROOT_LABEL } from "~/constants/assets";
import { assetRoute, childPath, parentPath } from "~/utils/assets";
import { apiErrorMessage } from "~/utils/errors";
import { keyName } from "~/utils/format";
import { useAssetStore } from "~/stores/assets";
import { useAssetAction } from "~/composables/asset-action";

/**
 * The dialogs behind the asset actions that need one: rename (a new name in
 * the same folder), move (a folder picked from the ones that exist), and
 * delete (a confirmation). Mounted once per page; whichever action is
 * pending decides which dialog is open. When the action lands on the asset
 * the page is showing, the page follows it — to the new key, or back to the
 * folder after a delete.
 */
const route = useRoute();
const router = useRouter();
const id = String(route.params.id);
const store = useAssetStore(id);
const { pending, close } = useAssetAction(id);

const key = computed(() => pending.value?.key ?? "");
const name = computed(() => keyName(key.value));
const folder = computed(() => parentPath(key.value));
const onAssetPage = computed(() => route.path === assetRoute(id, key.value));

const busy = ref(false);
const error = ref("");
const isOpen = (action: string) =>
  computed({
    get: () => pending.value?.action === action,
    set: (open: boolean) => {
      if (!open) close();
    },
  });
const renameOpen = isOpen("rename");
const moveOpen = isOpen("move");
const deleteOpen = isOpen("delete");

// Every dialog starts clean for its key.
watch(pending, (next) => {
  error.value = "";
  busy.value = false;
  if (next?.action === "rename") newName.value = keyName(next.key);
  if (next?.action === "move") pick.value = parentPath(next.key);
});

async function run(perform: () => Promise<void>) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    await perform();
    close();
  } catch (err) {
    error.value = apiErrorMessage(err, name.value);
  } finally {
    busy.value = false;
  }
}

// After a move the folder the asset left reloads (its rows changed), and
// the page follows the asset if it was showing it.
async function settle(from: string, to?: string) {
  await store.load(from);
  if (onAssetPage.value) await router.push(assetRoute(id, to ?? from));
}

// --- Rename: a single segment, in place.
const newName = ref("");
const renameInvalid = computed(
  () => newName.value.includes("/") || newName.value.trim() === "",
);
const renameSame = computed(() => newName.value.trim() === name.value);
function rename() {
  const from = folder.value;
  const to = childPath(from, newName.value.trim());
  void run(async () => {
    await store.move(key.value, to);
    await settle(from, to);
  });
}

// --- Move: browse the existing folders, one level at a time, from the
// asset's own folder; "Move here" lands it in the level shown.
const pick = ref("");
const pickLevel = ref<AssetSubfolder[]>([]);
const pickLoading = ref(false);
watch(
  [pick, moveOpen],
  async ([path, open]) => {
    if (!open) return;
    pickLoading.value = true;
    try {
      pickLevel.value = (await store.load(path)).folders;
    } finally {
      pickLoading.value = false;
    }
  },
  { immediate: true },
);
const pickCrumbs = computed(() =>
  pick.value === "" ? [ASSET_ROOT_LABEL] : [ASSET_ROOT_LABEL, ...pick.value.split("/")],
);
const moveSame = computed(() => pick.value === folder.value);
function move() {
  const from = folder.value;
  const to = childPath(pick.value, name.value);
  void run(async () => {
    await store.move(key.value, to);
    await settle(from, to);
  });
}

// --- Delete: the one-way door, confirmed.
function remove() {
  const from = folder.value;
  void run(async () => {
    await store.remove(from, key.value);
    if (onAssetPage.value) await router.push(assetRoute(id, from));
  });
}
</script>

<template>
  <Dialog
    v-model:open="renameOpen"
    title="Rename"
    :description="`Give ${name} a new name in ${folder || ASSET_ROOT_LABEL}.`"
  >
    <form class="dialog-form" @submit.prevent="rename">
      <input v-model="newName" type="text" :disabled="busy" />
      <p v-if="newName.includes('/')" class="error" role="alert">
        A name can't contain a slash — use Move to change the folder.
      </p>
      <p v-else-if="error" class="error" role="alert">{{ error }}</p>
      <div class="dialog-actions">
        <button
          type="submit"
          class="primary"
          :disabled="busy || renameInvalid || renameSame"
        >
          {{ busy ? "Renaming…" : "Rename" }}
        </button>
        <button type="button" class="ghost" @click="close">Cancel</button>
      </div>
    </form>
  </Dialog>

  <Dialog
    v-model:open="moveOpen"
    title="Move"
    :description="`Choose the folder ${name} moves to.`"
  >
    <div class="dialog-form">
      <p class="picker-path">
        <template v-for="(crumb, i) in pickCrumbs" :key="i">
          <span v-if="i > 0" class="picker-sep" aria-hidden="true">/</span>
          <span>{{ crumb }}</span>
        </template>
      </p>
      <ul class="picker-list" role="listbox" aria-label="Folders">
        <li v-if="pick !== ''">
          <button
            type="button"
            class="picker-row"
            :disabled="busy"
            @click="pick = parentPath(pick)"
          >
            <Icon class="f-icon" fill="currentColor" name="folder-up" />
            <span>Up one level</span>
          </button>
        </li>
        <li v-if="pickLoading" class="picker-empty">Loading…</li>
        <li v-else-if="!pickLevel.length" class="picker-empty">
          No folders here.
        </li>
        <li v-for="sub in pickLevel" :key="sub.name">
          <button
            type="button"
            class="picker-row"
            :disabled="busy"
            @click="pick = childPath(pick, sub.name)"
          >
            <Icon class="f-icon" fill="currentColor" name="folder" />
            <span>{{ sub.name }}</span>
          </button>
        </li>
      </ul>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="dialog-actions">
        <button
          type="button"
          class="primary"
          :disabled="busy || moveSame"
          @click="move"
        >
          {{ busy ? "Moving…" : "Move here" }}
        </button>
        <button type="button" class="ghost" @click="close">Cancel</button>
      </div>
    </div>
  </Dialog>

  <Dialog
    v-model:open="deleteOpen"
    title="Delete"
    description="This cannot be undone."
  >
    <p>
      Permanently delete <strong>{{ name }}</strong
      >?
    </p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <div class="dialog-actions">
      <button type="button" class="danger" :disabled="busy" @click="remove">
        {{ busy ? "Deleting…" : "Delete" }}
      </button>
      <button type="button" class="ghost" @click="close">Cancel</button>
    </div>
  </Dialog>
</template>
