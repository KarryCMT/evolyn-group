<script setup lang="ts">
import type { Component } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import {
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiCloseFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
} from '@remixicon/vue';
import { reactive, shallowRef, watch } from 'vue';

defineOptions({ name: 'BlankApplicationDialog' });

export interface BlankApplicationDraft {
  name: string;
  icon: BlankApplicationIcon;
}

export type BlankApplicationIcon = 'bookmark' | 'briefcase' | 'contacts' | 'chart' | 'check';

interface ApplicationIconOption {
  value: BlankApplicationIcon;
  label: string;
  icon: Component;
}

const iconOptions: ApplicationIconOption[] = [
  { value: 'bookmark', label: '书签', icon: RiBookmark3Fill },
  { value: 'briefcase', label: '公文包', icon: RiBriefcase4Fill },
  { value: 'contacts', label: '通讯录', icon: RiContactsBook3Fill },
  { value: 'chart', label: '图表', icon: RiPieChart2Fill },
  { value: 'check', label: '完成', icon: RiCheckboxCircleFill },
];

const emit = defineEmits<{
  submit: [draft: BlankApplicationDraft];
}>();

const visible = defineModel<boolean>({ default: false });
const formRef = shallowRef<FormInstance>();
const form = reactive<BlankApplicationDraft>({
  name: '',
  icon: 'bookmark',
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
  form.icon = 'bookmark';
  formRef.value?.clearValidate();
});

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;

  emit('submit', { name: form.name.trim(), icon: form.icon });
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
    append-to-body
  >
    <template #header>
      <header class="blank-application-dialog__header">
        <h2 class="blank-application-dialog__heading">创建空白应用</h2>
        <button
          class="blank-application-dialog__close"
          type="button"
          aria-label="关闭创建空白应用"
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
        <div class="blank-application-dialog__icon-list" role="radiogroup" aria-label="应用图标">
          <button
            v-for="option in iconOptions"
            :key="option.value"
            class="blank-application-dialog__icon-option"
            :class="{
              'blank-application-dialog__icon-option--selected': form.icon === option.value,
            }"
            type="button"
            role="radio"
            :aria-checked="form.icon === option.value"
            :aria-label="`选择${option.label}图标`"
            @click="form.icon = option.value"
          >
            <component :is="option.icon" />
          </button>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <footer class="blank-application-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
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
  border-radius: 14px;
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
  padding: 26px 32px 18px;
}

.blank-application-dialog .el-dialog__footer {
  border-top: 1px solid var(--el-border-color-lighter);
}

.blank-application-dialog__header {
  display: flex;
  height: 56px;
  padding: 0 16px 0 28px;
  align-items: center;
  justify-content: space-between;
}

.blank-application-dialog__heading {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 20px;
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
  font-size: 22px;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
}

.blank-application-dialog__form .el-form-item {
  margin-bottom: 24px;
}

.blank-application-dialog__form .el-form-item__label {
  height: auto;
  padding-bottom: 8px;
  color: var(--el-text-color-primary);
  font-size: 16px;
  line-height: 1.4;
}

.blank-application-dialog__form .el-input__wrapper {
  min-height: 44px;
  padding: 0 12px;
  border-radius: 8px;
  box-shadow: 0 0 0 1px var(--el-border-color) inset;
}

.blank-application-dialog__form .el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px var(--el-color-primary) inset;
}

.blank-application-dialog__form .el-input__inner {
  font-size: 15px;
}

.blank-application-dialog__icon-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
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
  border: 1px solid transparent;
  border-radius: 12px;
  font-size: 29px;
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
  padding: 0 28px;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.blank-application-dialog__footer .el-button {
  min-width: 78px;
  height: 36px;
  margin: 0;
  font-size: 15px;
}

@media (max-width: 720px) {
  .blank-application-dialog.el-dialog {
    width: calc(100vw - 32px) !important;
    min-height: 0;
  }

  .blank-application-dialog__header {
    height: 52px;
    padding: 0 12px 0 20px;
  }

  .blank-application-dialog__heading {
    font-size: 18px;
  }

  .blank-application-dialog .el-dialog__body {
    padding: 22px 20px 14px;
  }

  .blank-application-dialog__footer {
    height: 64px;
    padding: 0 20px;
  }
}
</style>
