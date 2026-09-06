<script setup lang="ts">
import type { RuntimeFieldEmits, RuntimeFieldProps } from '@evolyn.do/form/runtime-web';
import type { DeptGroupWidget, DeptWidget } from '@evolyn.do/form/schema';
import { ElTreeSelect } from 'element-plus';
import { computed, onMounted, shallowRef, watch } from 'vue';
import { loadDepartmentOptions, type DepartmentOption } from './departmentOptions';
import { useAuth } from '~/composables/auth';

defineOptions({ name: 'DepartmentSelectionField' });

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();

const options = shallowRef<DepartmentOption[]>([]);
const loading = shallowRef(false);
const loadFailed = shallowRef(false);
const { userInfo } = useAuth();
const widget = computed(() => props.item.widget as DeptWidget | DeptGroupWidget);
const multiple = computed(() => widget.value.type === 'deptgroup');
const disabled = computed(() => props.disabled || props.readonly);
const tenantID = computed(() => {
  const id = userInfo.value?.tenant.id;
  return id === undefined || id === null ? null : String(id);
});
const modelValue = computed<string | string[] | undefined>(() => {
  if (multiple.value) {
    return Array.isArray(props.modelValue) ? props.modelValue.filter(isString) : [];
  }
  return typeof props.modelValue === 'string' && props.modelValue ? props.modelValue : undefined;
});
const placeholder = computed(() => {
  if (loading.value) return '正在加载部门';
  if (loadFailed.value) return '部门加载失败';
  return multiple.value ? '选择部门' : '请选择部门';
});

function isString(value: unknown): value is string {
  return typeof value === 'string';
}

async function ensureOptions(): Promise<void> {
  if (options.value.length > 0 || loading.value) return;
  loading.value = true;
  loadFailed.value = false;
  try {
    options.value = [...(await loadDepartmentOptions(tenantID.value))];
  } catch {
    // 请求错误只影响本字段呈现；用户再次展开时允许重试，提交仍由服务端终审。
    loadFailed.value = true;
  } finally {
    loading.value = false;
  }
}

function update(value: unknown): void {
  if (multiple.value) {
    emit('update:modelValue', Array.isArray(value) ? value.filter(isString) : []);
  } else {
    emit('update:modelValue', typeof value === 'string' && value !== '' ? value : null);
  }
  emit('blur');
}

function onVisibleChange(open: boolean): void {
  if (open) void ensureOptions();
}

onMounted(() => void ensureOptions());
watch(tenantID, () => {
  options.value = [];
  void ensureOptions();
});
</script>

<template>
  <ElTreeSelect
    :id="`evf-department-${item.widget.widgetName}`"
    class="form-department-selection"
    :class="{ 'is-error': errors.length > 0 }"
    :model-value="modelValue"
    :data="options"
    node-key="value"
    value-key="value"
    :multiple="multiple"
    :show-checkbox="multiple"
    check-strictly
    filterable
    clearable
    :placeholder="placeholder"
    :loading="loading"
    :disabled="disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    @visible-change="onVisibleChange"
    @update:model-value="update"
  />
</template>

<style scoped lang="scss">
.form-department-selection {
  width: 100%;
  min-width: 0;
}
</style>
