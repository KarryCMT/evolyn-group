<script setup lang="ts">
import type { PermissionDataScope } from './permission.types';
import { computed } from 'vue';

defineOptions({ name: 'PermissionGroupEditorDataPanel' });

const dataScope = defineModel<PermissionDataScope>('dataScope', { required: true });
const match = computed({
  get: () => dataScope.value.match,
  set: (value: PermissionDataScope['match']) => {
    dataScope.value = { ...dataScope.value, match: value };
  },
});
</script>

<template>
  <section class="permission-group-editor-data-panel" aria-label="数据权限">
    <p class="permission-group-editor-data-panel__intro">可以管理哪些数据</p>
    <div class="permission-group-editor-data-panel__rule">
      <span>筛选出符合以下</span>
      <el-select v-model="match" aria-label="条件组合方式">
        <el-option label="所有" value="all" />
        <el-option label="任一" value="any" />
      </el-select>
      <span>条件的数据</span>
    </div>
    <p class="permission-group-editor-data-panel__hint">暂未添加筛选条件时，成员可管理全部数据。</p>
  </section>
</template>

<style scoped lang="scss">
.permission-group-editor-data-panel {
  &__intro {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__rule {
    display: flex;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 32px;
  }

  &__rule :deep(.el-select) {
    width: 108px;
  }

  &__hint {
    margin: var(--el-space-lg) 0 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 20px;
  }
}
</style>
