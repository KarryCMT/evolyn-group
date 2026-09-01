<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import type { Component } from 'vue';
import {
  RiArticleFill,
  RiBarChartBoxFill,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCalendarCheckFill,
  RiCheckboxCircleFill,
  RiCloseFill,
  RiContactsBook3Fill,
  RiFileList3Fill,
  RiShoppingCart2Fill,
} from '@remixicon/vue';
import { reactive, shallowRef, watch } from 'vue';

defineOptions({ name: 'FormAppearanceDialog' });

const props = defineProps<{
  initialName: string;
  initialIcon: string | null;
  /** 返回 true 表示已保存，由父级接收 success 后统一关闭并卸载弹窗。 */
  submit: (payload: FormAppearanceDraft) => Promise<boolean>;
}>();

const emit = defineEmits<{
  /** 保存成功后通知父级清理弹窗状态，避免局部 model 残留。 */
  success: [];
}>();

interface FormAppearanceDraft {
  name: string;
  icon: string;
}

interface IconOption {
  key: string;
  label: string;
  icon: Component;
}

const visible = defineModel<boolean>({ default: false });
const formRef = shallowRef<FormInstance>();
const submitting = shallowRef(false);
const draft = reactive<FormAppearanceDraft>({ name: '', icon: 'file-list' });
const iconOptions: IconOption[] = [
  { key: 'file-list', label: '表单', icon: RiFileList3Fill },
  { key: 'shopping-cart', label: '采购', icon: RiShoppingCart2Fill },
  { key: 'briefcase', label: '业务', icon: RiBriefcase4Fill },
  { key: 'contacts', label: '联系人', icon: RiContactsBook3Fill },
  { key: 'calendar', label: '日程', icon: RiCalendarCheckFill },
  { key: 'check', label: '任务', icon: RiCheckboxCircleFill },
  { key: 'chart', label: '图表', icon: RiBarChartBoxFill },
  { key: 'article', label: '文档', icon: RiArticleFill },
  { key: 'bookmark', label: '收藏', icon: RiBookmark3Fill },
];
const rules: FormRules<FormAppearanceDraft> = {
  name: [
    { required: true, message: '请输入名称', trigger: 'blur' },
    { max: 128, message: '名称不能超过 128 个字符', trigger: 'blur' },
  ],
};

// 每次由菜单操作重新打开时，从最新菜单快照初始化，取消不会写回任何数据。
watch(
  visible,
  (isVisible) => {
    if (!isVisible) return;
    draft.name = props.initialName;
    draft.icon = iconOptions.some((option) => option.key === props.initialIcon)
      ? String(props.initialIcon)
      : 'file-list';
    formRef.value?.clearValidate();
  },
  { immediate: true },
);

function selectIcon(icon: string) {
  if (!submitting.value) draft.icon = icon;
}

async function confirm() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid || submitting.value) return;

  submitting.value = true;
  try {
    const success = await props.submit({ name: draft.name.trim(), icon: draft.icon });
    if (success) emit('success');
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="form-appearance-dialog"
    width="460px"
    top="22vh"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    append-to-body
  >
    <template #header>
      <header class="form-appearance-dialog__header">
        <h2 class="form-appearance-dialog__heading">修改名称和图标</h2>
        <button
          class="form-appearance-dialog__close"
          type="button"
          aria-label="关闭修改名称和图标"
          :disabled="submitting"
          @click="visible = false"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </header>
    </template>

    <el-form
      ref="formRef"
      class="form-appearance-dialog__form"
      :model="draft"
      :rules="rules"
      label-position="top"
      @submit.prevent="confirm"
    >
      <el-form-item label="名称" prop="name">
        <el-input v-model="draft.name" maxlength="128" autofocus />
      </el-form-item>

      <el-form-item label="图标" prop="icon">
        <div class="form-appearance-dialog__icon-grid" role="radiogroup" aria-label="表单图标">
          <button
            v-for="option in iconOptions"
            :key="option.key"
            class="form-appearance-dialog__icon-option"
            :class="{ 'form-appearance-dialog__icon-option--selected': draft.icon === option.key }"
            type="button"
            role="radio"
            :aria-checked="draft.icon === option.key"
            :aria-label="`选择${option.label}图标`"
            :disabled="submitting"
            @click="selectIcon(option.key)"
          >
            <component :is="option.icon" aria-hidden="true" />
          </button>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <footer class="form-appearance-dialog__footer">
        <el-button :disabled="submitting" @click="visible = false"> 取消 </el-button>
        <el-button type="primary" :loading="submitting" @click="confirm"> 确定 </el-button>
      </footer>
    </template>
  </el-dialog>
</template>

<!-- 弹窗渲染到 body，以下样式用唯一块类限制作用范围。 -->
<style lang="scss">
.form-appearance-dialog.el-dialog {
  max-width: calc(100vw - 32px);
  margin-bottom: 0;
  overflow: hidden;
  border-radius: var(--el-border-radius-large);
}

.form-appearance-dialog .el-dialog__header,
.form-appearance-dialog .el-dialog__footer {
  padding: 0;
  margin: 0;
}

.form-appearance-dialog .el-dialog__header {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.form-appearance-dialog .el-dialog__body {
  padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-lg);
}

.form-appearance-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.form-appearance-dialog__header {
  display: flex;
  height: 54px;
  padding: 0 var(--el-space-lg) 0 var(--el-space-3xl);
  align-items: center;
  justify-content: space-between;
}

.form-appearance-dialog__heading {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-large);
  font-weight: 650;
  line-height: 1.4;
}

.form-appearance-dialog__close {
  display: inline-flex;
  width: 32px;
  height: 32px;
  padding: 0;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
  font-size: 22px;

  &:hover:not(:disabled) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.form-appearance-dialog__form .el-form-item {
  margin-bottom: var(--el-space-2xl);
}

.form-appearance-dialog__form .el-form-item__label {
  padding-bottom: var(--el-space-sm);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-base);
  line-height: 1.4;
}

.form-appearance-dialog__form .el-input__wrapper {
  min-height: 40px;
  padding: 0 var(--el-space-md);
  border-radius: var(--el-border-radius-medium);
  box-shadow: 0 0 0 1px var(--el-border-color) inset;
}

.form-appearance-dialog__form .el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset;
}

.form-appearance-dialog__icon-grid {
  display: grid;
  grid-template-columns: repeat(9, 36px);
  gap: var(--el-space-sm);
}

.form-appearance-dialog__icon-option {
  display: inline-flex;
  width: 36px;
  height: 36px;
  padding: 0;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: var(--el-fill-color-lighter);
  border: 1px solid transparent;
  border-radius: var(--el-border-radius-medium);
  font-size: 19px;
  transition:
    color 0.16s ease,
    background-color 0.16s ease,
    border-color 0.16s ease,
    transform 0.16s ease;

  &:hover:not(:disabled) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    transform: translateY(-1px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &:disabled {
    cursor: not-allowed;
  }

  &--selected {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
}

.form-appearance-dialog__footer {
  display: flex;
  height: 64px;
  padding: 0 var(--el-space-3xl);
  align-items: center;
  justify-content: flex-end;
  gap: var(--el-space-md);
}

.form-appearance-dialog__footer .el-button {
  min-width: 72px;
  height: 34px;
  margin: 0;
}

@media (max-width: 520px) {
  .form-appearance-dialog.el-dialog {
    width: calc(100vw - 32px) !important;
  }

  .form-appearance-dialog .el-dialog__body {
    padding: var(--el-space-xl) var(--el-space-2xl) var(--el-space-md);
  }

  .form-appearance-dialog__header,
  .form-appearance-dialog__footer {
    padding-right: var(--el-space-2xl);
    padding-left: var(--el-space-2xl);
  }

  .form-appearance-dialog__icon-grid {
    grid-template-columns: repeat(6, 36px);
  }
}
</style>
