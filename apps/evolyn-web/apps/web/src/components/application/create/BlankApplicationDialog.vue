<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import type { EvolynIconPickerValue } from '@evolyn.do/ui';
import { RiCloseFill } from '@remixicon/vue';
import { reactive, shallowRef, watch } from 'vue';
import ApplicationIconPicker from '../ApplicationIconPicker.vue';

defineOptions({ name: 'BlankApplicationDialog' });

export interface BlankApplicationDraft {
  name: string;
  icon: BlankApplicationIcon;
}

export type BlankApplicationIcon = EvolynIconPickerValue;

const props = defineProps<{
  /** 异步提交处理：resolve true 表示创建成功（父级随后关闭弹窗）；false 表示失败，保持弹窗开启以保留填写内容 */
  submit: (draft: BlankApplicationDraft) => Promise<boolean>;
}>();

const emit = defineEmits<{
  /** 提交成功：由父级负责关闭弹窗（含上级「新建应用」弹窗） */
  success: [draft: BlankApplicationDraft];
}>();

const visible = defineModel<boolean>({ default: false });
const formRef = shallowRef<FormInstance>();
const submitting = shallowRef(false);
const form = reactive<BlankApplicationDraft>({
  name: '',
  icon: { type: 'remix', name: 'bookmark', background: '#f7be54,#eda426' },
});
const iconValue = shallowRef<EvolynIconPickerValue>({
  type: 'remix',
  name: 'bookmark',
  background: '#f7be54,#eda426',
});
const rules: FormRules<BlankApplicationDraft> = {
  name: [
    { required: true, message: '请输入应用名称', trigger: 'blur' },
    { max: 64, message: '应用名称不能超过 64 个字符', trigger: 'blur' },
  ],
};

// 每次重新打开表单时恢复初始草稿，取消不会影响上一级的模板选择。
watch(visible, (isVisible) => {
  if (!isVisible) return;
  form.name = '';
  iconValue.value = { type: 'remix', name: 'bookmark', background: '#f7be54,#eda426' };
  form.icon = iconValue.value;
  formRef.value?.clearValidate();
});

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid || submitting.value) return;

  submitting.value = true;
  try {
    // 等待创建请求完成才决定去留：失败时保持弹窗开启并保留填写内容，
    // 错误提示由父级处理函数负责（按 errCode 分支）
    form.icon = iconValue.value;
    const draft = { name: form.name.trim(), icon: iconValue.value };
    const ok = await props.submit(draft);
    if (ok) emit('success', draft);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="blank-application-dialog"
    width="560px"
    top="18vh"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="!submitting"
    append-to-body
  >
    <template #header>
      <header class="blank-application-dialog__header">
        <h2 class="blank-application-dialog__heading">创建空白应用</h2>
        <button
          class="blank-application-dialog__close"
          type="button"
          aria-label="关闭创建空白应用"
          :disabled="submitting"
          @click="visible = false"
        >
          <RiCloseFill />
        </button>
      </header>
    </template>

    <el-form
      ref="formRef"
      class="blank-application-dialog__form"
      :model="form"
      :rules="rules"
      label-position="top"
      @submit.prevent="submit"
    >
      <el-form-item label="名称" prop="name">
        <el-input
          v-model="form.name"
          maxlength="64"
          placeholder="给应用命名，例如“客户管理系统”"
          autofocus
        />
      </el-form-item>

      <el-form-item label="图标" prop="icon">
        <ApplicationIconPicker v-model="iconValue" />
      </el-form-item>
    </el-form>

    <template #footer>
      <footer class="blank-application-dialog__footer">
        <el-button :disabled="submitting" @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">确定</el-button>
      </footer>
    </template>
  </el-dialog>
</template>

<!-- 弹窗已传送至 body，样式必须以唯一块类限制作用范围。 -->
<style lang="scss">
.blank-application-dialog.el-dialog {
  display: flex;
  max-width: calc(100vw - 32px);
  min-height: 382px;
  margin-bottom: 0;
  overflow: hidden;
  flex-direction: column;
  border-radius: var(--el-border-radius-large);
}

.blank-application-dialog .el-dialog__header,
.blank-application-dialog .el-dialog__footer {
  flex: 0 0 auto;
  padding: 0;
  margin: 0;
}

.blank-application-dialog .el-dialog__header {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.blank-application-dialog .el-dialog__body {
  flex: 1;
  min-height: 0;
  padding: var(--el-space-3xl) var(--el-space-4xl) var(--el-space-xl);
}

.blank-application-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.blank-application-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
  align-items: center;
  justify-content: space-between;
}

.blank-application-dialog__heading {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-extra-large);
  font-weight: 650;
  line-height: 26px;
}

.blank-application-dialog__close {
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
  font-size: var(--el-font-size-extra-large);

  &:hover:not(:disabled) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible:not(:disabled) {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &:disabled {
    cursor: not-allowed;
    color: var(--el-text-color-placeholder);
  }
}

.blank-application-dialog__form .el-form-item {
  margin-bottom: var(--el-space-3xl);
}

.blank-application-dialog__form .el-form-item__label {
  height: auto;
  padding-bottom: var(--el-space-md);
  color: var(--el-text-color-primary);
  font-size: var(--el-font-size-medium);
  line-height: 1.4;
}

.blank-application-dialog__form .el-input__wrapper {
  min-height: 44px;
  padding: 0 var(--el-space-lg);
  border-radius: var(--el-border-radius-medium);
  box-shadow: 0 0 0 1px var(--el-border-color) inset;
}

.blank-application-dialog__form .el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset;
}

.blank-application-dialog__form .el-input__inner {
  font-size: var(--el-font-size-base);
}

.blank-application-dialog__icon-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--el-space-md);
}

.blank-application-dialog__icon-option {
  display: inline-flex;
  width: 56px;
  height: 56px;
  padding: 0;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-color-transparent);
  border-radius: var(--el-border-radius-large);
  font-size: 28.9996px;
  transition:
    color 0.18s ease,
    background-color 0.18s ease,
    border-color 0.18s ease,
    transform 0.18s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    transform: translateY(-1px);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }

  &--selected {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
}

.blank-application-dialog__footer {
  display: flex;
  height: 72px;
  padding: 0 var(--el-space-3xl);
  align-items: center;
  justify-content: flex-end;
  gap: var(--el-space-md);
}

.blank-application-dialog__footer .el-button {
  min-width: 78px;
  height: 36px;
  margin: 0;
  font-size: var(--el-font-size-base);
}

@media (max-width: 720px) {
  .blank-application-dialog.el-dialog {
    width: calc(100vw - 32px) !important;
    min-height: 0;
  }

  .blank-application-dialog__header {
    height: 52px;
    padding: 0 var(--el-space-lg) 0 var(--el-space-2xl);
  }

  .blank-application-dialog__heading {
    font-size: var(--el-font-size-large);
  }

  .blank-application-dialog .el-dialog__body {
    padding: var(--el-space-2xl) var(--el-space-2xl) var(--el-space-lg);
  }

  .blank-application-dialog__footer {
    height: 64px;
    padding: 0 var(--el-space-2xl);
  }
}
</style>
