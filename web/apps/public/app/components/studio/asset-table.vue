<script setup lang="ts">
import Menu from "@zoobzio/foundation/components/core/menu.vue";

import { computed, useRoute } from "#imports";

import { ASSET_ACTIONS, DEFAULT_ASSET_SORT } from "~/constants/assets";
import { assetIcon, assetRoute, sortAssets, sortFolders } from "~/utils/assets";
import {
  formatBytes,
  formatDate,
  formatRelative,
  keyName,
} from "~/utils/format";
import { childPath, folderPath } from "~/utils/path";
import { useAssetStore } from "~/stores/assets";
import { useAssetMenu } from "~/composables/asset-menu";
import { useFolderView } from "~/composables/folder-view";
import { useNow } from "~/composables/clock";

/**
 * One folder's contents as a table: subfolders first, then files, each a
 * link to its own route, with size, when it was last written, and (for
 * files) the action menu, whose changes run through the page's dialogs. A folder's date is its newest write beneath it,
 * from the bookkeeping; a file's is what object storage reports. Reads the
 * level the page loaded into the store, narrowed and ordered by the
 * toolbar's view state, and follows both as they change.
 */
const route = useRoute();
const id = String(route.params.id);
const path = folderPath(route.params.path);

const store = useAssetStore(id);
const level = store.level(path);
const now = useNow();
const { query, sort } = useFolderView("assets", id, path, DEFAULT_ASSET_SORT);
const folders = computed(() =>
  sortFolders(level.value?.folders ?? [], sort.value, query.value),
);
const files = computed(() =>
  sortAssets(level.value?.assets ?? [], sort.value, query.value),
);
const empty = computed(() => !folders.value.length && !files.value.length);
const searching = computed(() => query.value.trim() !== "");

const menu = useAssetMenu(id);
</script>

<template>
  <section class="panel panel-flush">
    <table class="data-table">
      <thead>
        <tr>
          <th class="col-icon"><span class="sr-only">Type</span></th>
          <th>Name</th>
          <th class="col-size">Size</th>
          <th class="col-date">Modified</th>
          <th class="col-actions"><span class="sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="empty">
          <td class="placeholder-cell" colspan="5">
            {{
              searching ? `Nothing matches "${query.trim()}".` : "Empty folder"
            }}
          </td>
        </tr>
        <tr v-for="folder in folders" :key="`folder:${folder.name}`">
          <td class="col-icon">
            <Icon class="f-icon row-icon" fill="currentColor" name="folder" />
          </td>
          <td>
            <NuxtLink
              class="row-link"
              :to="assetRoute(id, childPath(path, folder.name))"
            >
              {{ folder.name }}
            </NuxtLink>
          </td>
          <td class="col-size">{{ formatBytes(folder.size) }}</td>
          <td class="col-date">
            <time
              v-if="folder.last_written_at"
              :datetime="folder.last_written_at"
              :title="formatDate(folder.last_written_at)"
            >
              {{ formatRelative(folder.last_written_at, now) }}
            </time>
            <template v-else>—</template>
          </td>
          <td class="col-actions" />
        </tr>
        <tr v-for="asset in files" :key="asset.key">
          <td class="col-icon">
            <Icon
              class="f-icon row-icon"
              fill="currentColor"
              :name="assetIcon(asset.kind)"
            />
          </td>
          <td>
            <NuxtLink class="row-link" :to="assetRoute(id, asset.key)">
              {{ keyName(asset.key) }}
            </NuxtLink>
          </td>
          <td class="col-size">{{ formatBytes(asset.size) }}</td>
          <td class="col-date">
            <time
              v-if="asset.last_modified"
              :datetime="asset.last_modified"
              :title="formatDate(asset.last_modified)"
            >
              {{ formatRelative(asset.last_modified, now) }}
            </time>
            <template v-else>—</template>
          </td>
          <td class="col-actions">
            <Menu
              :groups="ASSET_ACTIONS"
              align="end"
              @select="menu.select(asset.key, $event)"
            >
              <button type="button" class="f-button" aria-label="Actions">
                <Icon class="f-icon" fill="currentColor" name="actions" />
              </button>
            </Menu>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>
