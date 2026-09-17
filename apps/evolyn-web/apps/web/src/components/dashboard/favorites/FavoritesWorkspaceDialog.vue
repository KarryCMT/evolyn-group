<script setup lang="ts">
import { shallowRef } from 'vue';
import FavoritePickerDialog from './FavoritePickerDialog.vue';
import FavoritesDialog from './FavoritesDialog.vue';

defineOptions({ name: 'FavoritesWorkspaceDialog' });

const visible = defineModel<boolean>({ default: false });
const pickerVisible = shallowRef(false);
</script>

<template>
  <FavoritesDialog v-model="visible" @add="pickerVisible = true" />
  <!-- 选择器内部以差量方式写回（收藏/取消逐项调用），列表状态由共享
       composable 维护，关闭后两处入口即时一致。 -->
  <FavoritePickerDialog v-model="pickerVisible" />
</template>
