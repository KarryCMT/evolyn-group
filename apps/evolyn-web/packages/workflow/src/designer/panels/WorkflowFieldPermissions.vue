<script setup lang="ts">
import { ElEmpty, ElOption, ElScrollbar, ElSelect } from 'element-plus';
import { computed, ref } from 'vue';
import type { WorkflowField, WorkflowFieldPermission } from '../../schema';

/**
 * 审批节点字段权限矩阵：widgetName → hidden/readonly/editable/required 四档。
 * 未显式配置的字段不写入 formPermissions（运行时按默认可编辑处理），
 * 「默认」选项即从配置中移除该键。
 */
defineOptions({ name: 'WorkflowFieldPermissions' });

const props = defineProps<{
  fields: readonly WorkflowField[];
  formPermissions: Record<string, WorkflowFieldPermission> | undefined;
}>();

const emit = defineEmits<{
  update: [permissions: Record<string, WorkflowFieldPermission>];
}>();

const PERMISSION_OPTIONS: Array<{ value: '' | WorkflowFieldPermission; label: string }> = [
  { value: '', label: '默认' },
  { value: 'hidden', label: '隐藏' },
  { value: 'readonly', label: '只读' },
  { value: 'editable', label: '可编辑' },
  { value: 'required', label: '必填' },
];

const keyword = ref('');
const filteredFields = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return props.fields;
  return props.fields.filter((field) => field.label.toLocaleLowerCase().includes(normalized));
});

function permissionOf(field: WorkflowField): '' | WorkflowFieldPermission {
  return props.formPermissions?.[field.widgetName] ?? '';
}

function updatePermission(field: WorkflowField, value: '' | WorkflowFieldPermission) {
  const next: Record<string, WorkflowFieldPermission> = { ...(props.formPermissions ?? {}) };
  if (value === '') {
    delete next[field.widgetName];
  } else {
    next[field.widgetName] = value;
  }
  emit('update', next);
}
</script>

<template>
  <div class="workflow-field-permissions">
    <div class="workflow-field-permissions__header">
      <span>字段</span>
      <span>权限</span>
    </div>
    <ElScrollbar class="workflow-field-permissions__list">
      <label
        v-for="field in filteredFields"
        :key="field.widgetName"
        class="workflow-field-permissions__row"
      >
        <span class="workflow-field-permissions__label" :title="field.label">
          {{ field.label
          }}<i v-if="field.required" class="workflow-field-permissions__required">*</i>
        </span>
        <ElSelect
          :model-value="permissionOf(field)"
          size="small"
          @update:model-value="
            (value) => updatePermission(field, value as '' | WorkflowFieldPermission)
          "
        >
          <ElOption
            v-for="option in PERMISSION_OPTIONS"
            :key="option.value"
            :value="option.value"
            :label="option.label"
          />
        </ElSelect>
      </label>
      <ElEmpty
        v-if="filteredFields.length === 0"
        class="workflow-field-permissions__empty"
        :image-size="64"
        description="暂无表单字段，请先在表单设计中添加"
      />
    </ElScrollbar>
  </div>
</template>

<style scoped lang="scss">
.workflow-field-permissions {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;

  &__header {
    display: flex;
    min-height: 32px;
    padding: 0 var(--el-space-xs);
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-regular);
    font-size: 13px;
    font-weight: 600;
  }

  &__list {
    min-height: 120px;
    flex: 1;
  }

  &__row {
    display: flex;
    padding: var(--el-space-xs);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-xs);

    &:hover {
      background: var(--el-fill-color-light);
    }
  }

  &__label {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__required {
    margin-left: 2px;
    color: var(--el-color-danger);
    font-style: normal;
  }

  &__empty {
    padding: var(--el-space-lg) 0;
  }
}
</style>
