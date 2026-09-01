<script setup lang="ts">
import { ElForm, ElFormItem, ElOption, ElSelect, ElTreeSelect } from 'element-plus';
import { computed } from 'vue';
import type {
  WorkflowActorOptions,
  WorkflowAssigneeSpec,
  WorkflowAssigneeType,
  WorkflowField,
} from '../../schema';

/**
 * 审批人/抄送对象规格编辑器（AssigneeSpec）：type 切换决定参数区，
 * roleCode 语义 = 角色名称；表单用户字段来自注入的表单字段契约。
 */
defineOptions({ name: 'WorkflowAssigneeEditor' });

const props = defineProps<{
  spec: WorkflowAssigneeSpec | undefined;
  actorOptions: WorkflowActorOptions | undefined;
  fields: readonly WorkflowField[];
  /** 表单未提供成员字段时隐藏 form_field 选项（协议仍支持，由调用方决定） */
  label: string;
}>();

const emit = defineEmits<{
  update: [spec: WorkflowAssigneeSpec | undefined];
}>();

const TYPE_OPTIONS: Array<{ value: WorkflowAssigneeType; label: string }> = [
  { value: 'user', label: '指定成员' },
  { value: 'role', label: '角色' },
  { value: 'department', label: '部门成员' },
  { value: 'department_manager', label: '部门负责人' },
  { value: 'starter_manager', label: '发起人直属主管' },
  { value: 'form_field', label: '表单用户字段' },
];

const currentType = computed<WorkflowAssigneeType | undefined>(() => props.spec?.type);

/** 表单用户字段候选：优先成员选择类字段，未标记时退回全部字段可选 */
const userFields = computed(() => {
  const marked = props.fields.filter((field) => field.userField);
  return marked.length > 0 ? marked : props.fields;
});

/** 部门树：el-tree-select 需要 value/label/children 标准结构 */
const departmentTree = computed(() =>
  (props.actorOptions?.departments ?? []).map((node) => ({
    value: node.id,
    label: node.label,
    children: node.children?.length ? mapChildren(node.children) : undefined,
  })),
);

function mapChildren(
  nodes: WorkflowActorOptions['departments'],
): Array<{ value: number; label: string; children?: unknown[] }> {
  return nodes.map((node) => ({
    value: node.id,
    label: node.label,
    children: node.children?.length ? mapChildren(node.children) : undefined,
  }));
}

function switchType(type: WorkflowAssigneeType) {
  // 切换类型即重置参数：不同 type 的参数字段互不通用，避免残留脏配置
  emit('update', { type });
}

function patchSpec(patch: Partial<WorkflowAssigneeSpec>) {
  if (!props.spec) return;
  emit('update', { ...props.spec, ...patch });
}
</script>

<template>
  <ElForm class="workflow-assignee" label-position="top" size="default" @submit.prevent>
    <ElFormItem :label="`${label}类型`" class="workflow-assignee__item">
      <ElSelect
        :model-value="currentType"
        placeholder="选择类型"
        data-test="assignee-type"
        @update:model-value="switchType"
      >
        <ElOption
          v-for="option in TYPE_OPTIONS"
          :key="option.value"
          :value="option.value"
          :label="option.label"
        />
      </ElSelect>
    </ElFormItem>

    <ElFormItem
      v-if="spec?.type === 'user'"
      label="选择成员（可多选）"
      class="workflow-assignee__item"
    >
      <ElSelect
        :model-value="spec.userIds ?? []"
        multiple
        filterable
        placeholder="搜索并选择成员"
        @update:model-value="(value: number[]) => patchSpec({ userIds: value })"
      >
        <ElOption
          v-for="member in actorOptions?.members ?? []"
          :key="member.id"
          :value="member.id"
          :label="member.label"
        />
      </ElSelect>
    </ElFormItem>

    <ElFormItem v-else-if="spec?.type === 'role'" label="选择角色" class="workflow-assignee__item">
      <ElSelect
        :model-value="spec.roleCode"
        filterable
        placeholder="选择角色（按角色成员解析）"
        @update:model-value="(value: string) => patchSpec({ roleCode: value })"
      >
        <ElOption
          v-for="role in actorOptions?.roles ?? []"
          :key="role.code"
          :value="role.code"
          :label="role.label"
        />
      </ElSelect>
    </ElFormItem>

    <ElFormItem
      v-else-if="spec?.type === 'department' || spec?.type === 'department_manager'"
      label="选择部门"
      class="workflow-assignee__item"
    >
      <ElTreeSelect
        :model-value="spec.deptId"
        :data="departmentTree"
        check-strictly
        default-expand-all
        placeholder="选择部门"
        @update:model-value="(value: number) => patchSpec({ deptId: value })"
      />
    </ElFormItem>

    <ElFormItem
      v-else-if="spec?.type === 'form_field'"
      label="选择表单用户字段"
      class="workflow-assignee__item"
    >
      <ElSelect
        :model-value="spec.formField"
        placeholder="选择表单中的成员字段"
        @update:model-value="(value: string) => patchSpec({ formField: value })"
      >
        <ElOption
          v-for="field in userFields"
          :key="field.widgetName"
          :value="field.widgetName"
          :label="field.label"
        />
      </ElSelect>
    </ElFormItem>

    <p v-else-if="spec?.type === 'starter_manager'" class="workflow-assignee__hint">
      按发起人的汇报关系解析直属主管，无需额外配置。
    </p>
    <p v-else class="workflow-assignee__hint">请先选择{{ label }}类型。</p>
  </ElForm>
</template>

<style scoped lang="scss">
.workflow-assignee {
  &__item {
    :deep(.el-select),
    :deep(.el-tree-select) {
      width: 100%;
    }
  }

  &__hint {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.7;
  }
}
</style>
