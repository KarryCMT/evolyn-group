<script setup lang="ts">
import { RiQuestionFill, RiSearchLine } from '@remixicon/vue';
import { ElCheckbox, ElEmpty, ElInput, ElTooltip } from 'element-plus';
import { computed, ref } from 'vue';
import type { WorkflowField, WorkflowFieldPermission } from '../../schema';

/** 字段权限矩阵：可见/可编辑映射既有权限枚举，简报字段独立持久化。 */
defineOptions({ name: 'WorkflowFieldPermissions' });

const props = defineProps<{
  fields: readonly WorkflowField[];
  formPermissions: Record<string, WorkflowFieldPermission> | undefined;
  summaryFields: readonly string[] | undefined;
}>();

const emit = defineEmits<{
  updatePermissions: [permissions: Record<string, WorkflowFieldPermission>];
  updateSummaryFields: [fields: string[]];
}>();

const keyword = ref('');
const filteredFields = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return props.fields;
  return props.fields.filter(
    (field) =>
      field.label.toLocaleLowerCase().includes(normalized) ||
      field.widgetName.toLocaleLowerCase().includes(normalized),
  );
});

function permissionOf(field: WorkflowField): WorkflowFieldPermission {
  return props.formPermissions?.[field.widgetName] ?? 'editable';
}

function updatePermission(field: WorkflowField, permission: WorkflowFieldPermission) {
  const next = { ...(props.formPermissions ?? {}) };
  if (permission === 'editable') delete next[field.widgetName];
  else next[field.widgetName] = permission;
  emit('updatePermissions', next);
}

function setVisible(field: WorkflowField, visible: boolean) {
  updatePermission(field, visible ? 'editable' : 'hidden');
}

function setEditable(field: WorkflowField, editable: boolean) {
  updatePermission(field, editable ? 'editable' : 'readonly');
}

function setSummary(field: WorkflowField, enabled: boolean) {
  const next = new Set(props.summaryFields ?? []);
  enabled ? next.add(field.widgetName) : next.delete(field.widgetName);
  emit('updateSummaryFields', [...next]);
}

function setAll(column: 'visible' | 'editable' | 'summary', enabled: boolean) {
  if (column === 'summary') {
    emit('updateSummaryFields', enabled ? props.fields.map((field) => field.widgetName) : []);
    return;
  }
  const next = { ...(props.formPermissions ?? {}) };
  for (const field of props.fields) {
    if (column === 'visible') {
      if (enabled) delete next[field.widgetName];
      else next[field.widgetName] = 'hidden';
    } else if (enabled) {
      delete next[field.widgetName];
    } else if (next[field.widgetName] !== 'hidden') {
      next[field.widgetName] = 'readonly';
    }
  }
  emit('updatePermissions', next);
}

const allVisible = computed(
  () => props.fields.length > 0 && props.fields.every((field) => permissionOf(field) !== 'hidden'),
);
const allEditable = computed(
  () =>
    props.fields.length > 0 &&
    props.fields.every((field) => ['editable', 'required'].includes(permissionOf(field))),
);
const allSummary = computed(
  () =>
    props.fields.length > 0 &&
    props.fields.every((field) => props.summaryFields?.includes(field.widgetName)),
);
</script>

<template>
  <div class="workflow-field-permissions">
    <ElInput v-model="keyword" class="workflow-field-permissions__search" placeholder="搜索">
      <template #prefix><RiSearchLine /></template>
    </ElInput>

    <div class="workflow-field-permissions__header workflow-field-permissions__grid">
      <span>字段</span><span>可见</span><span>可编辑</span>
      <span class="workflow-field-permissions__summary-title">
        简报
        <ElTooltip content="勾选后，该字段会显示在流程简报中" placement="top">
          <RiQuestionFill />
        </ElTooltip>
      </span>
    </div>

    <div v-if="fields.length" class="workflow-field-permissions__all workflow-field-permissions__grid">
      <span>全选</span>
      <ElCheckbox :model-value="allVisible" @change="(value) => setAll('visible', Boolean(value))" />
      <ElCheckbox :model-value="allEditable" @change="(value) => setAll('editable', Boolean(value))" />
      <ElCheckbox :model-value="allSummary" @change="(value) => setAll('summary', Boolean(value))" />
    </div>

    <div class="workflow-field-permissions__list">
      <div
        v-for="field in filteredFields"
        :key="field.widgetName"
        class="workflow-field-permissions__row workflow-field-permissions__grid"
      >
        <span class="workflow-field-permissions__label" :title="field.label">
          {{ field.label }}<i v-if="field.required">*</i>
        </span>
        <ElCheckbox
          :model-value="permissionOf(field) !== 'hidden'"
          @change="(value) => setVisible(field, Boolean(value))"
        />
        <ElCheckbox
          :model-value="['editable', 'required'].includes(permissionOf(field))"
          :disabled="permissionOf(field) === 'hidden'"
          @change="(value) => setEditable(field, Boolean(value))"
        />
        <ElCheckbox
          :model-value="summaryFields?.includes(field.widgetName) ?? false"
          @change="(value) => setSummary(field, Boolean(value))"
        />
      </div>
      <ElEmpty
        v-if="filteredFields.length === 0"
        class="workflow-field-permissions__empty"
        :image-size="58"
        description="暂无匹配字段"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.workflow-field-permissions {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;

  &__search {
    margin-bottom: 12px;
    :deep(.el-input__wrapper) { background: var(--el-fill-color-lighter); box-shadow: none; }
  }

  &__grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 48px 58px 48px;
    align-items: center;
    column-gap: 4px;
  }

  &__header {
    min-height: 34px;
    color: var(--el-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    text-align: center;
    > :first-child { text-align: left; }
  }

  &__summary-title {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 3px;
    svg { width: 15px; color: var(--el-text-color-placeholder); }
  }

  &__all,
  &__row {
    min-height: 42px;
    text-align: center;
    :deep(.el-checkbox) { justify-self: center; }
  }

  &__all > span {
    width: fit-content;
    color: var(--el-color-primary);
  }

  &__list { min-height: 0; overflow-y: auto; flex: 1; }
  &__row:hover { background: var(--el-fill-color-light); }

  &__label {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    text-align: left;
    text-overflow: ellipsis;
    white-space: nowrap;
    i { margin-left: 2px; color: var(--el-color-danger); font-style: normal; }
  }

  &__empty { padding: 20px 0; }
}
</style>
