<script setup lang="ts">
import type { InputInstance } from 'element-plus';
import { RiEdit2Fill } from '@remixicon/vue';
import { nextTick, shallowRef, useTemplateRef, watch } from 'vue';

defineOptions({ name: 'FormWorkspaceTitleEditor' });

const props = withDefaults(
  defineProps<{
    name: string;
    disabled?: boolean;
    saving?: boolean;
  }>(),
  {
    disabled: false,
    saving: false,
  },
);

const emit = defineEmits<{
  submit: [name: string, onSuccess: () => void];
}>();

const titleInputRef = useTemplateRef<InputInstance>('titleInput');
const editing = shallowRef(false);
const draft = shallowRef('');

// 外部改名完成后同步输入值；编辑期间保留用户尚未提交的内容。
watch(
  () => props.name,
  (name) => {
    if (!editing.value) draft.value = name;
  },
  { immediate: true },
);

async function startEditing(): Promise<void> {
  if (props.disabled || props.saving) return;
  draft.value = props.name;
  editing.value = true;
  await nextTick();
  titleInputRef.value?.focus();
  titleInputRef.value?.select();
}

function cancelEditing(): void {
  editing.value = false;
  draft.value = props.name;
}

/** Enter 与失焦共用提交入口，保存成功前保留编辑态，避免失败时丢失输入。 */
function submitEditing(): void {
  if (!editing.value || props.saving) return;
  const name = draft.value.trim();
  if (!name || name === props.name) {
    cancelEditing();
    return;
  }
  emit('submit', name, () => {
    editing.value = false;
  });
}
</script>

<template>
  <div class="form-workspace-title-editor">
    <el-input
      v-if="editing"
      ref="titleInput"
      v-model="draft"
      class="form-workspace-title-editor__input"
      :maxlength="128"
      :disabled="saving"
      aria-label="表单名称"
      @blur="submitEditing"
      @keydown.enter.prevent="submitEditing"
      @keydown.esc.prevent="cancelEditing"
    />
    <button
      v-else
      class="form-workspace-title-editor__trigger"
      type="button"
      :disabled="disabled"
      :title="name"
      :aria-label="`修改表单名称：${name}`"
      @click="startEditing"
    >
      <strong class="form-workspace-title-editor__text">{{ name }}</strong>
      <RiEdit2Fill class="form-workspace-title-editor__icon" aria-hidden="true" />
    </button>
  </div>
</template>

<style scoped lang="scss">
.form-workspace-title-editor {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  min-width: 0;
  max-width: 220px;

  &__trigger {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    width: 100%;
    min-width: 0;
    height: 32px;
    padding: 0 var(--el-space-sm);
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: var(--el-color-transparent);
    border: 0;
    border-radius: var(--el-border-radius-base);
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &:disabled {
      color: var(--el-text-color-disabled);
      cursor: default;
      background: var(--el-color-transparent);
    }
  }

  &__text {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 28px;
    white-space: nowrap;
  }

  &__icon {
    flex: 0 0 auto;
    width: 16px;
    height: 16px;
    opacity: 0;
    transition: opacity 0.18s ease;
  }

  &__trigger:hover &__icon,
  &__trigger:focus-visible &__icon {
    opacity: 1;
  }

  &__input {
    width: 100%;
  }
}
</style>
