<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import type { ReminderRecipientInput } from './messageCenter.types';
import { RiCloseFill } from '@remixicon/vue';
import { reactive, shallowRef, watch } from 'vue';

defineOptions({ name: 'AddMessageRecipientDialog' });

const emit = defineEmits<{
  submit: [payload: ReminderRecipientInput];
}>();
const visible = defineModel<boolean>({ default: false });

const formRef = shallowRef<FormInstance>();
const form = reactive<ReminderRecipientInput>({ name: '', mobile: '', email: '' });
const rules: FormRules<ReminderRecipientInput> = {
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  mobile: [{ pattern: /^$|^1\d{10}$/, message: '请输入正确的手机号', trigger: 'blur' }],
  email: [{ type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }],
};

/** 每次重新打开均提供干净表单，避免取消后的内容残留。 */
watch(visible, (isVisible) => {
  if (!isVisible) return;
  form.name = '';
  form.mobile = '';
  form.email = '';
  formRef.value?.clearValidate();
});

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  emit('submit', {
    name: form.name.trim(),
    mobile: form.mobile.trim(),
    email: form.email.trim(),
  });
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="add-message-recipient-dialog"
    title="添加提醒对象"
    width="560px"
    :close-on-click-modal="false"
    :show-close="false"
  >
    <template #header="{ close, titleId, titleClass }">
      <div class="add-message-recipient-dialog__header">
        <h2 :id="titleId" :class="titleClass">添加提醒对象</h2>
        <el-button
          class="add-message-recipient-dialog__close"
          :icon="RiCloseFill"
          text
          circle
          aria-label="关闭添加提醒对象弹窗"
          @click="close"
        />
      </div>
    </template>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="86px" @submit.prevent="submit">
      <el-form-item label="姓名" prop="name">
        <el-input v-model="form.name" maxlength="64" autocomplete="name" />
      </el-form-item>
      <el-form-item label="手机" prop="mobile">
        <el-input v-model="form.mobile" inputmode="tel" autocomplete="tel" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" type="email" autocomplete="email" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false"> 取消 </el-button>
      <el-button type="primary" @click="submit"> 保存 </el-button>
    </template>
  </el-dialog>
</template>

<!-- Dialog 默认传送至 body，使用独立类对齐消息中心的表单弹窗规格。 -->
<style lang="scss">
.add-message-recipient-dialog.el-dialog {
  overflow: hidden;
  border-radius: 12px;
}

.add-message-recipient-dialog .el-dialog__header {
  height: 56px;
  margin: 0;
  padding: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.add-message-recipient-dialog__header {
  display: flex;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  align-items: center;
  justify-content: space-between;
  padding: 0;
}

.add-message-recipient-dialog__header .el-dialog__title {
  margin: 0;
  color: #202938;
  font-size: 18px;
  font-weight: 700;
  line-height: 26px;
}

.add-message-recipient-dialog__close.el-button {
  min-width: 32px;
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: 8px;
  color: #4f5969;
  font-size: 22px;
}

.add-message-recipient-dialog__close.el-button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.add-message-recipient-dialog .el-dialog__body {
  padding: 24px 28px 8px;
}

.add-message-recipient-dialog .el-form-item {
  margin-bottom: 18px;
}

.add-message-recipient-dialog .el-form-item__label {
  color: #293445;
  font-size: 16px;
}

.add-message-recipient-dialog .el-input__wrapper {
  min-height: 40px;
  box-shadow: 0 0 0 1px #d7dde6 inset;
}

.add-message-recipient-dialog .el-dialog__footer {
  padding: 14px 28px 18px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.add-message-recipient-dialog .el-button {
  min-width: 76px;
  height: 36px;
}

@media (width <= 767px) {
  .add-message-recipient-dialog .el-dialog__header {
    height: 52px;
  }
}
</style>
