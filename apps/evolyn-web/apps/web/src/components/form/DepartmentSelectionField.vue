<script setup lang="ts">
import type { RuntimeFieldEmits, RuntimeFieldProps } from '@evolyn.do/form/runtime-web';
import type { DeptGroupWidget, DeptWidget } from '@evolyn.do/form/schema';
import {
  EvolynMemberDepartmentRolePicker,
  type EvolynMemberDepartmentRolePickerSelection,
  type EvolynMemberDepartmentRolePickerTreeNode,
} from '@evolyn.do/ui';
import { RiAddLine, RiBuilding4Fill, RiCloseFill } from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';
import { loadDepartmentOptions, type DepartmentOption } from './departmentOptions';
import { useAuth } from '~/composables/auth';

defineOptions({ name: 'DepartmentSelectionField' });

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

// 本字段只负责把运行时值与通用确认式选择器连接起来；弹窗的草稿、搜索和虚拟树
// 交互由 EvolynMemberDepartmentRolePicker 收敛，避免预览与运行页实现两套选择逻辑。
const options = shallowRef<DepartmentOption[]>([]);
const loading = shallowRef(false);
const loadFailed = shallowRef(false);
const pickerVisible = shallowRef(false);
const pickerSelections = shallowRef<EvolynMemberDepartmentRolePickerSelection[]>([]);
const { loadUserInfo, userInfo } = useAuth();
const widget = computed(() => props.item.widget as DeptWidget | DeptGroupWidget);
const multiple = computed(() => widget.value.type === 'deptgroup');
const disabled = computed(() => props.disabled || props.readonly);
const tenantID = computed(() => {
  const id = userInfo.value?.tenant.id;
  return id === undefined || id === null ? null : String(id);
});
const selectedReferences = computed<string[]>(() => {
  if (multiple.value)
    return Array.isArray(props.modelValue) ? props.modelValue.filter(isString) : [];
  return typeof props.modelValue === 'string' && props.modelValue ? [props.modelValue] : [];
});
const departmentTree = computed<EvolynMemberDepartmentRolePickerTreeNode[]>(() =>
  options.value.map(toPickerNode),
);
const departmentByID = computed(() => {
  const entries = new Map<string, EvolynMemberDepartmentRolePickerTreeNode>();
  const visit = (nodes: readonly EvolynMemberDepartmentRolePickerTreeNode[]) => {
    for (const node of nodes) {
      entries.set(String(node.id), node);
      if (node.children?.length) visit(node.children);
    }
  };
  visit(departmentTree.value);
  return entries;
});
const currentMemberDepartmentIds = computed(() => {
  // 登录聚合接口的类型承诺为数组，但旧租户或接口异常时可能回传 null；预览不应
  // 因“当前用户所在部门”这一辅助页签而中断整个表单渲染。
  const departments = userInfo.value?.member.departments;
  return Array.isArray(departments) ? departments.map((department) => String(department.id)) : [];
});

function isString(value: unknown): value is string {
  return typeof value === 'string';
}

function toPickerNode(option: DepartmentOption): EvolynMemberDepartmentRolePickerTreeNode {
  return {
    id: option.value,
    label: option.label,
    disabled: option.disabled,
    children: option.children?.map(toPickerNode),
  };
}

async function ensureOptions(): Promise<void> {
  if (options.value.length > 0 || loading.value) return;
  loading.value = true;
  loadFailed.value = false;
  try {
    options.value = [...(await loadDepartmentOptions(tenantID.value))];
  } catch {
    // 请求错误只影响选择器呈现；提交时仍由服务端按当前租户目录终审。
    loadFailed.value = true;
  } finally {
    loading.value = false;
  }
}

function resetPickerSelections(): void {
  pickerSelections.value = selectedReferences.value.map((id) => {
    const node = departmentByID.value.get(id);
    return { id, label: node?.label ?? `已删除部门（${id}）`, type: 'department' };
  });
}

async function openPicker(): Promise<void> {
  if (disabled.value) return;
  pickerVisible.value = true;
  // 当前成员部门来自登录聚合接口。设计器预览可能早于全局登录资料加载，打开
  // 选择器时补拉一次，保证“当前用户所在部门”不依赖页面进入时序。
  if (!Array.isArray(userInfo.value?.member.departments)) await loadUserInfo();
  await ensureOptions();
  // 异步目录响应可能晚于关闭或租户切换；仅对仍开启的当前弹窗写入草稿。
  if (pickerVisible.value) resetPickerSelections();
}

function confirmSelection(selections: EvolynMemberDepartmentRolePickerSelection[]): void {
  const selected = selections
    .filter((selection) => selection.type === 'department')
    .map((selection) => String(selection.id));
  emit('update:modelValue', multiple.value ? selected : (selected[0] ?? null));
  emit('blur');
}

function departmentLabel(id: string): string {
  return departmentByID.value.get(id)?.label ?? `已删除部门（${id}）`;
}

function removeDepartment(id: string, event: MouseEvent): void {
  event.stopPropagation();
  const next = selectedReferences.value.filter((reference) => reference !== id);
  emit('update:modelValue', multiple.value ? next : null);
  emit('blur');
}

watch(tenantID, () => {
  options.value = [];
  pickerVisible.value = false;
  resetPickerSelections();
});
watch(
  [selectedReferences, departmentByID],
  () => {
    if (!pickerVisible.value) resetPickerSelections();
  },
  { immediate: true },
);
</script>

<template>
  <div
    class="form-department-selection"
    :class="{
      'form-department-selection--multiple': multiple,
      'form-department-selection--disabled': disabled,
      'form-department-selection--has-value': selectedReferences.length > 0,
      'form-department-selection--error': errors.length > 0,
    }"
  >
    <button
      :id="`evf-department-${item.widget.widgetName}`"
      class="form-department-selection__control"
      :disabled="disabled"
      type="button"
      :aria-required="!item.widget.allowBlank || undefined"
      :aria-invalid="errors.length > 0 || undefined"
      @click="openPicker"
    >
      <template v-if="selectedReferences.length">
        <span v-for="id in selectedReferences" :key="id" class="form-department-selection__tag">
          <i><RiBuilding4Fill /></i>{{ departmentLabel(id)
          }}<RiCloseFill @click="removeDepartment(id, $event)" />
        </span>
      </template>
      <span v-else class="form-department-selection__placeholder"><RiAddLine />选择部门</span>
    </button>
  </div>

  <EvolynMemberDepartmentRolePicker
    v-model:open="pickerVisible"
    v-model="pickerSelections"
    title="部门列表"
    :departments="departmentTree"
    :current-member-department-ids="currentMemberDepartmentIds"
    :show-current-member-department-tab="true"
    :selectable-types="['department']"
    :department-multiple="multiple"
    :allow-empty="true"
    :empty-text="
      loading ? '正在加载部门…' : loadFailed ? '部门加载失败，请关闭后重试' : '暂无可选择的部门'
    "
    :current-member-department-empty-text="
      loading ? '正在加载当前用户部门…' : '当前用户暂未归属部门'
    "
    @confirm="confirmSelection"
  />
</template>

<style scoped lang="scss">
.form-department-selection {
  display: block;
  width: 100%;
  min-width: 0;

  // 部门与成员字段的选择区尺寸保持一致；两者仅替换目录来源和图标，避免同类
  // 组织字段在预览及运行时发生无意义的行高跳变。
  &__control {
    display: flex;
    width: 100%;
    min-width: 0;
    min-height: 32px;
    padding: var(--el-space-xs) var(--el-space-sm);
    align-items: center;
    flex-wrap: wrap;
    gap: var(--el-space-xs);
    border: 1px dashed var(--evf-color-border, var(--el-border-color));
    border-radius: var(--el-border-radius-small);
    color: var(--evf-color-text-regular, var(--el-text-color-regular));
    background: var(--evf-color-bg, var(--el-bg-color));
    cursor: pointer;
    font: inherit;
    text-align: left;
  }
  &__control:hover:not(:disabled) {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__control:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
  &__placeholder,
  &__tag,
  &__tag i {
    display: inline-flex;
    align-items: center;
  }
  &__placeholder {
    width: 100%;
    justify-content: center;
    gap: var(--el-space-xs);
    font-size: var(--el-font-size-small);
  }
  &__placeholder svg {
    width: 16px;
    height: 16px;
  }
  &__tag {
    height: 24px;
    padding: 0 var(--el-space-xs);
    gap: var(--el-space-xs);
    border-radius: var(--el-border-radius-small);
    background: var(--evf-color-fill-light, var(--el-fill-color-light));
    font-size: var(--el-font-size-small);
  }
  &__tag i {
    width: 18px;
    height: 18px;
    justify-content: center;
    color: var(--el-color-primary);
    font-style: normal;
  }
  &__tag i svg,
  &__tag > svg {
    width: 15px;
    height: 15px;
  }
  &__tag > svg {
    cursor: pointer;
  }
  &--multiple .form-department-selection__control {
    // 多部门选择器与设计画布预览统一限制为 68px；部门标签换行超过可用高度时，
    // 仅在控件内部纵向滚动，避免长列表撑高表单布局。
    box-sizing: border-box;
    height: 68px;
    min-height: 68px;
    max-height: 68px;
    overflow-x: hidden;
    overflow-y: auto;
    align-content: center;
  }
  &--multiple.form-department-selection--has-value .form-department-selection__control {
    align-content: flex-start;
  }
  &--disabled .form-department-selection__control {
    color: var(--evf-color-text-disabled, var(--el-text-color-disabled));
    cursor: not-allowed;
    background: var(--evf-color-fill-light, var(--el-fill-color-light));
  }
  &--error .form-department-selection__control {
    border-color: var(--el-color-danger);
  }
}
</style>
