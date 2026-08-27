<script setup lang="ts">
import type { AccountSettingsTab } from '~/types/account';
import { RiFileTextFill, RiLock2Fill } from '@remixicon/vue';

defineOptions({ name: 'AccountSettingsSidebar' });

const props = defineProps<{
  modelValue: AccountSettingsTab;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: AccountSettingsTab];
}>();

function selectTab(tab: AccountSettingsTab) {
  emit('update:modelValue', tab);
}
</script>

<template>
  <aside class="account-settings-sidebar" aria-label="个人设置导航">
    <button
      class="account-settings-sidebar__item"
      :class="{ 'account-settings-sidebar__item--active': props.modelValue === 'basic' }"
      type="button"
      @click="selectTab('basic')"
    >
      <el-icon><RiFileTextFill /></el-icon>
      <span>基本信息</span>
    </button>
    <button
      class="account-settings-sidebar__item"
      :class="{ 'account-settings-sidebar__item--active': props.modelValue === 'security' }"
      type="button"
      @click="selectTab('security')"
    >
      <el-icon><RiLock2Fill /></el-icon>
      <span>账号安全</span>
    </button>
  </aside>
</template>

<style scoped lang="scss">
.account-settings-sidebar {
  display: flex;
  width: 164px;
  flex: 0 0 164px;
  flex-direction: column;
  gap: var(--el-space-xs);
  padding: var(--el-space-md);
  border-right: 1px solid var(--el-border-color-lighter);

  &__item {
    display: flex;
    align-items: center;
    width: 100%;
    height: 38px;
    gap: var(--el-space-md);
    padding: 0 var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: var(--el-font-size-base);
    text-align: left;

    &:hover {
      background: var(--el-fill-color-light);
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }
}
</style>
