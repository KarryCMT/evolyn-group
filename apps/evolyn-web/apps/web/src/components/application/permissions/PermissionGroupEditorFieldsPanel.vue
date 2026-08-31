<script setup lang="ts">
import type { PermissionFieldPermission } from './permission.types';
import { RiSearchLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';

defineOptions({ name: 'PermissionGroupEditorFieldsPanel' });

const fields = defineModel<PermissionFieldPermission[]>({ required: true });
const keyword = shallowRef('');
const visibleFields = computed(() => {
  const normalizedKeyword = keyword.value.trim();
  return normalizedKeyword
    ? fields.value.filter((field) => field.label.includes(normalizedKeyword))
    : fields.value;
});

function updatePermission(
  fieldName: string,
  property: 'visible' | 'editable',
  value: boolean | string | number,
) {
  fields.value = fields.value.map((field) => {
    if (field.field !== fieldName) return field;
    const enabled = Boolean(value);
    return property === 'visible'
      ? { ...field, visible: enabled, editable: enabled ? field.editable : false }
      : { ...field, editable: enabled, visible: enabled || field.visible };
  });
}
</script>

<template>
  <section class="permission-group-editor-fields-panel" aria-label="字段权限">
    <header class="permission-group-editor-fields-panel__header">
      <p class="permission-group-editor-fields-panel__intro">可以查看和编辑数据的哪些字段</p>
      <el-input
        v-model="keyword"
        class="permission-group-editor-fields-panel__search"
        placeholder="搜索"
      >
        <template #prefix>
          <RiSearchLine aria-hidden="true" />
        </template>
      </el-input>
    </header>
    <div class="permission-group-editor-fields-panel__table" role="table">
      <div class="permission-group-editor-fields-panel__table-header" role="row">
        <span>字段</span>
        <span>可见</span>
        <span>可编辑</span>
      </div>
      <el-scrollbar class="permission-group-editor-fields-panel__rows">
        <div
          v-for="field in visibleFields"
          :key="field.field"
          class="permission-group-editor-fields-panel__row"
          role="row"
        >
          <span class="permission-group-editor-fields-panel__field-label">
            <i v-if="field.required" aria-hidden="true">*</i>{{ field.label }}
          </span>
          <el-checkbox
            :model-value="field.visible"
            :disabled="field.required && field.visible"
            :aria-label="`${field.label}可见`"
            @update:model-value="updatePermission(field.field, 'visible', $event)"
          />
          <el-checkbox
            :model-value="field.editable"
            :disabled="!field.visible"
            :aria-label="`${field.label}可编辑`"
            @update:model-value="updatePermission(field.field, 'editable', $event)"
          />
        </div>
        <el-empty v-if="!visibleFields.length" description="未找到匹配字段" :image-size="64" />
      </el-scrollbar>
    </div>
  </section>
</template>

<style scoped lang="scss">
.permission-group-editor-fields-panel {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;

  &__header,
  &__table-header,
  &__row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 120px 120px;
    align-items: center;
  }

  &__header {
    margin-bottom: var(--el-space-lg);
    gap: var(--el-space-xl);
  }

  &__intro {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__search {
    width: 220px;
    justify-self: end;
  }

  &__table {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    border: 1px solid var(--el-border-color-lighter);
  }

  &__table-header {
    min-height: 52px;
    padding: 0 var(--el-space-lg);
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-small);
    font-weight: 600;
  }

  &__rows {
    min-height: 0;
    flex: 1;
  }

  &__row {
    min-height: 62px;
    padding: 0 var(--el-space-lg);
    border-top: 1px solid var(--el-border-color-lighter);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-small);
  }

  &__field-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;

    i {
      margin-right: 2px;
      color: var(--el-color-danger);
      font-style: normal;
    }
  }
}
</style>
