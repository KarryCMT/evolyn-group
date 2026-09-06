<script setup lang="ts">
import type { RuntimeFieldEmits, RuntimeFieldProps } from '@evolyn.do/form/runtime-web';
import type { UserGroupWidget, UserWidget } from '@evolyn.do/form/schema';
import type { MemberListItemDto } from '~/api/member';
import { RiAddLine, RiCloseFill, RiUserFill } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import MemberPickerDialog from './MemberPickerDialog.vue';

defineOptions({ name: 'MemberSelectionField' });

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();
const pickerVisible = shallowRef(false);
const memberNames = shallowRef<Record<string, string>>({});
const widget = computed(() => props.item.widget as UserWidget | UserGroupWidget);
const multiple = computed(() => widget.value.type === 'usergroup');
const selectedIds = computed<string[]>(() => {
  if (multiple.value)
    return Array.isArray(props.modelValue) ? props.modelValue.filter(isString) : [];
  return typeof props.modelValue === 'string' && props.modelValue ? [props.modelValue] : [];
});
const disabled = computed(() => props.disabled || props.readonly);

function isString(value: unknown): value is string {
  return typeof value === 'string';
}
function openPicker(): void {
  if (!disabled.value) pickerVisible.value = true;
}
function onConfirm(members: MemberListItemDto[]): void {
  memberNames.value = {
    ...memberNames.value,
    ...Object.fromEntries(members.map((member) => [String(member.id), member.name])),
  };
  const ids = members.map((member) => String(member.id));
  emit('update:modelValue', multiple.value ? ids : (ids[0] ?? null));
  emit('blur');
}
function memberName(id: string): string {
  return memberNames.value[id] ?? `成员 ${id}`;
}
function remove(id: string, event: MouseEvent): void {
  event.stopPropagation();
  const next = selectedIds.value.filter((item) => item !== id);
  emit('update:modelValue', multiple.value ? next : null);
  emit('blur');
}
</script>

<template>
  <div
    class="form-member-selection"
    :class="{
      'form-member-selection--multiple': multiple,
      'form-member-selection--disabled': disabled,
      'form-member-selection--has-value': selectedIds.length > 0,
      'form-member-selection--error': errors.length > 0,
    }"
  >
    <button
      type="button"
      class="form-member-selection__control"
      :disabled="disabled"
      @click="openPicker"
    >
      <template v-if="selectedIds.length">
        <span v-for="id in selectedIds" :key="id" class="form-member-selection__tag">
          <i><RiUserFill /></i>{{ memberName(id) }}<RiCloseFill @click="remove(id, $event)" />
        </span>
      </template>
      <span v-else class="form-member-selection__placeholder"><RiAddLine />选择成员</span>
    </button>
    <MemberPickerDialog
      v-model="pickerVisible"
      :multiple="multiple"
      :selected-ids="selectedIds"
      :scope="widget.scope"
      :department-ids="widget.departments"
      @confirm="onConfirm"
    />
  </div>
</template>

<style scoped lang="scss">
.form-member-selection {
  display: block;
  width: 100%;
  min-width: 0;

  // 运行时优先使用 form 的 --evf 主题变量；独立宿主未提供时再回退到 Element Plus。
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
  // 多选空态固定成参考图中的整行选择区；有标签后仍可自然向下扩展。
  &--multiple .form-member-selection__control {
    min-height: 60px;
    align-content: center;
  }
  &--multiple.form-member-selection--has-value .form-member-selection__control {
    align-content: flex-start;
  }
  &--disabled .form-member-selection__control {
    color: var(--evf-color-text-disabled, var(--el-text-color-disabled));
    cursor: not-allowed;
    background: var(--evf-color-fill-light, var(--el-fill-color-light));
  }
  &--error .form-member-selection__control {
    border-color: var(--el-color-danger);
  }
}
</style>
