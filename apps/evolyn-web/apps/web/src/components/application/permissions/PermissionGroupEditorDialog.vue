<script setup lang="ts">
import type {
  AssetPermissionGroup,
  PermissionAssetType,
  PermissionDataScope,
  PermissionFieldPermission,
  PermissionOperation,
} from './permission.types';
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, ref, shallowRef, watch } from 'vue';
import PermissionGroupEditorDataPanel from './PermissionGroupEditorDataPanel.vue';
import PermissionGroupEditorFieldsPanel from './PermissionGroupEditorFieldsPanel.vue';
import PermissionGroupEditorNamePanel from './PermissionGroupEditorNamePanel.vue';
import PermissionGroupEditorOperationsPanel from './PermissionGroupEditorOperationsPanel.vue';

defineOptions({ name: 'PermissionGroupEditorDialog' });

const props = defineProps<{
  assetType: PermissionAssetType | undefined;
  group: AssetPermissionGroup | undefined;
}>();
const emit = defineEmits<{
  confirm: [group: AssetPermissionGroup];
}>();
type EditorSection = 'name' | 'operations' | 'fields' | 'data';
type PermissionGroupEditorDraft = Omit<
  AssetPermissionGroup,
  'operations' | 'fields' | 'dataScope'
> & {
  operations: PermissionOperation[];
  fields: PermissionFieldPermission[];
  dataScope: PermissionDataScope;
};

const visible = defineModel<boolean>({ default: false });
const activeSection = shallowRef<EditorSection>('name');
const draft = ref<PermissionGroupEditorDraft>();

const sections: Array<{ value: EditorSection; label: string }> = [
  { value: 'name', label: '名称信息' },
  { value: 'operations', label: '操作权限' },
  { value: 'fields', label: '字段权限' },
  { value: 'data', label: '数据权限' },
];

const defaultFields: PermissionFieldPermission[] = [
  { field: 'employee_name', label: '员工姓名', required: true, visible: true, editable: true },
  { field: 'contact_phone', label: '联系电话', required: true, visible: true, editable: true },
  { field: 'department', label: '所属部门', visible: true, editable: true },
  { field: 'position', label: '岗位', visible: true, editable: true },
  { field: 'id_number', label: '身份证号码', visible: true, editable: true },
  { field: 'gender', label: '性别', visible: true, editable: true },
  { field: 'birthday', label: '出生日期', visible: true, editable: true },
];

// 默认操作集与设计 §3.2 一致：普通表单 9 项，流程表单在普通操作之上追加 3 项流程专属操作。
const standardOperations: PermissionOperation[] = [
  'view',
  'add',
  'copy',
  'edit',
  'delete',
  'batch_print',
  'batch_modify',
  'import',
  'export',
];
const workflowOperations: PermissionOperation[] = [
  ...standardOperations,
  'workflow_owner_transfer',
  'workflow_terminate',
  'workflow_activate',
];

const isWorkflow = computed(() => props.assetType === 'workflow-form');
const confirmDisabled = computed(() => !draft.value?.name.trim());

/** 将父级只读权限组复制为弹窗草稿，取消不会污染卡片上的当前配置。 */
function createDraft(group: AssetPermissionGroup): PermissionGroupEditorDraft {
  const defaultOperations = isWorkflow.value ? workflowOperations : standardOperations;
  const dataScope: PermissionDataScope = group.dataScope
    ? { ...group.dataScope }
    : { match: 'all' };
  return {
    ...group,
    subjects: [...group.subjects],
    operations: [...(group.operations ?? defaultOperations)],
    fields: (group.fields ?? defaultFields).map((field) => ({ ...field })),
    dataScope,
  };
}

function close() {
  visible.value = false;
}

/**
 * 提交前的配置期校验，与后端设计方案 §4 规则逐字对齐：
 * 必填字段不得隐藏（否则成员无值可填且值不出网）；操作含「添加」时必填字段必须可编辑
 * （添加路径没有旧值可回填）。校验失败仅提示，不关闭弹窗，便于返回对应分区修正。
 */
function validateDraft(): string | undefined {
  if (!draft.value) return undefined;
  const hiddenRequired = draft.value.fields?.find((field) => field.required && !field.visible);
  if (hiddenRequired) {
    return `必填字段「${hiddenRequired.label}」不可隐藏，请先在字段权限中恢复其可见性`;
  }
  if (draft.value.operations?.includes('add')) {
    const lockedRequired = draft.value.fields?.find((field) => field.required && !field.editable);
    if (lockedRequired) {
      return `操作权限包含「添加」时，必填字段「${lockedRequired.label}」必须可编辑`;
    }
  }
  return undefined;
}

function confirm() {
  if (!draft.value || confirmDisabled.value) return;
  const violation = validateDraft();
  if (violation) {
    ElMessage.warning(violation);
    return;
  }
  emit('confirm', {
    ...draft.value,
    name: draft.value.name.trim(),
    description: draft.value.description.trim(),
    subjects: [...draft.value.subjects],
    operations: [...(draft.value.operations ?? [])],
    fields: draft.value.fields?.map((field) => ({ ...field })),
    dataScope: draft.value.dataScope ? { ...draft.value.dataScope } : undefined,
  });
  close();
}

watch(visible, (open) => {
  if (!open || !props.group) return;
  activeSection.value = 'name';
  draft.value = createDraft(props.group);
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="permission-group-editor-dialog"
    width="960px"
    :style="{ width: 'min(960px, calc(100vw - 32px))' }"
    :show-close="false"
    :close-on-click-modal="false"
    append-to-body
    destroy-on-close
  >
    <template #header>
      <header class="permission-group-editor-dialog__header">
        <h2 class="permission-group-editor-dialog__title">编辑权限组</h2>
        <button
          class="permission-group-editor-dialog__close"
          type="button"
          aria-label="关闭编辑权限组"
          @click="close"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>
    </template>

    <div v-if="draft" class="permission-group-editor-dialog__body">
      <nav class="permission-group-editor-dialog__navigation" aria-label="权限组设置项">
        <button
          v-for="section in sections"
          :key="section.value"
          class="permission-group-editor-dialog__navigation-item"
          :class="{
            'permission-group-editor-dialog__navigation-item--active':
              activeSection === section.value,
          }"
          type="button"
          @click="activeSection = section.value"
        >
          {{ section.label }}
        </button>
      </nav>

      <main class="permission-group-editor-dialog__content">
        <PermissionGroupEditorNamePanel
          v-if="activeSection === 'name'"
          v-model:group-name="draft.name"
          v-model:description="draft.description"
        />
        <PermissionGroupEditorOperationsPanel
          v-else-if="activeSection === 'operations'"
          v-model="draft.operations"
          :workflow="isWorkflow"
        />
        <PermissionGroupEditorFieldsPanel
          v-else-if="activeSection === 'fields'"
          v-model="draft.fields"
        />
        <PermissionGroupEditorDataPanel v-else v-model:data-scope="draft.dataScope" />
      </main>
    </div>

    <template #footer>
      <footer class="permission-group-editor-dialog__footer">
        <el-button @click="close"> 取消 </el-button>
        <el-button type="primary" :disabled="confirmDisabled" @click="confirm"> 确定 </el-button>
      </footer>
    </template>
  </el-dialog>
</template>

<!-- 弹窗传送至 body，使用唯一块类隔离中尺寸弹窗的布局与 Element Plus 默认样式。 -->
<style lang="scss">
.permission-group-editor-dialog.el-dialog {
  display: flex;
  height: min(760px, calc(100vh - 32px));
  max-width: calc(100vw - 32px);
  margin: var(--el-space-lg) auto;
  overflow: hidden;
  flex-direction: column;
  border-radius: var(--el-border-radius-large);
}

.permission-group-editor-dialog .el-dialog__header,
.permission-group-editor-dialog .el-dialog__footer {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
}

.permission-group-editor-dialog .el-dialog__header {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.permission-group-editor-dialog .el-dialog__body {
  display: flex;
  min-height: 0;
  padding: 0;
  flex: 1;
  overflow: hidden;
}

.permission-group-editor-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.permission-group-editor-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
  align-items: center;
  justify-content: space-between;
}

.permission-group-editor-dialog__title {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-extra-large);
  font-weight: 650;
  line-height: 28px;
}

.permission-group-editor-dialog__close {
  display: inline-flex;
  width: 32px;
  height: 32px;
  padding: 0;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--el-border-radius-base);
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;

  svg {
    width: 22px;
    height: 22px;
  }

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

.permission-group-editor-dialog__body {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
}

.permission-group-editor-dialog__navigation {
  width: 184px;
  padding: var(--el-space-lg) 0;
  flex: 0 0 auto;
  border-right: 1px solid var(--el-border-color-lighter);
}

.permission-group-editor-dialog__navigation-item {
  display: block;
  width: 100%;
  min-height: 54px;
  padding: 0 var(--el-space-3xl);
  border: 0;
  border-left: 3px solid transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  font: inherit;
  font-size: var(--el-font-size-base);
  text-align: left;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -2px;
  }

  &--active {
    border-left-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    font-weight: 650;
  }
}

.permission-group-editor-dialog__content {
  display: flex;
  min-width: 0;
  min-height: 0;
  padding: var(--el-space-3xl);
  flex: 1;
  flex-direction: column;
  overflow: hidden;
}

.permission-group-editor-dialog__footer {
  display: flex;
  min-height: 72px;
  padding: 0 var(--el-space-3xl);
  align-items: center;
  justify-content: flex-end;
  gap: var(--el-space-md);
}

@media (max-width: 680px) {
  .permission-group-editor-dialog.el-dialog {
    height: min(680px, calc(100vh - 24px));
    margin: var(--el-space-md) auto;
  }

  .permission-group-editor-dialog__navigation {
    width: 136px;
  }

  .permission-group-editor-dialog__navigation-item {
    padding: 0 var(--el-space-lg);
    font-size: var(--el-font-size-small);
  }

  .permission-group-editor-dialog__content {
    padding: var(--el-space-xl);
  }
}
</style>
