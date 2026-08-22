<script setup lang="ts">
import { shallowRef } from 'vue';
import FavoritePickerDialog from './FavoritePickerDialog.vue';
import FavoritesDialog from './FavoritesDialog.vue';
import { useFavoriteApplications } from './useFavoriteApplications';

defineOptions({ name: 'FavoritesWorkspaceDialog' });

const visible = defineModel<boolean>({ default: false });
const pickerVisible = shallowRef(false);
const { favoriteApplications, selectedApplicationIds, replaceFavoriteApplications } =
  useFavoriteApplications();

function saveFavorites(ids: string[]) {
  replaceFavoriteApplications(ids);
}
</script>

<template>
  <FavoritesDialog
    v-model="visible"
    :applications="favoriteApplications"
    @add="pickerVisible = true"
  />
  <FavoritePickerDialog
    v-model="pickerVisible"
    :selected-ids="selectedApplicationIds"
    @confirm="saveFavorites"
  />
</template>
