<script setup lang="ts">
import { shallowRef } from 'vue';
import FavoritePickerDialog from './FavoritePickerDialog.vue';
import FavoritesDialog from './FavoritesDialog.vue';
import { useFavoriteApps } from './useFavoriteApps';

defineOptions({ name: 'FavoritesWorkspaceDialog' });

const visible = defineModel<boolean>({ default: false });
const pickerVisible = shallowRef(false);
const { favoriteApps, selectedAppIds, replaceFavoriteApps } = useFavoriteApps();

function saveFavorites(ids: string[]) {
  replaceFavoriteApps(ids);
}
</script>

<template>
  <FavoritesDialog v-model="visible" :apps="favoriteApps" @add="pickerVisible = true" />
  <FavoritePickerDialog
    v-model="pickerVisible"
    :selected-ids="selectedAppIds"
    @confirm="saveFavorites"
  />
</template>
