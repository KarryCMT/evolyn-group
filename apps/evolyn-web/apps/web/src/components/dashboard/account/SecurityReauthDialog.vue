<script setup lang="ts">
import { ElMessage } from 'element-plus';
import { RiCloseFill, RiShieldKeyholeFill } from '@remixicon/vue';
import { reactive, shallowRef, watch } from 'vue';
import { reauthAccountSecurity } from '~/api/account';
import { encryptPassword } from '~/api/conf';

defineOptions({ name: 'SecurityReauthDialog' });

const props = defineProps<{
  modelValue: boolean;
  title: string;
  description: string;
  confirmText: string;
  passwordInitialized: boolean;
  actionLoading: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  verified: [reauthToken: string];
}>();

const submitting = shallowRef(false);
const form = reactive({ password: '' });

function close() {
  form.password = '';
  emit('update:modelValue', false);
}

async function verify() {
  if (props.actionLoading) return;
  if (!props.passwordInitialized) {
    ElMessage.warning('请先在基本资料中设置登录密码');
    return;
  }
  if (!form.password) {
    ElMessage.warning('请输入当前密码');
    return;
  }

  submitting.value = true;
  try {
    const password = await encryptPassword(form.password);
    const { reauthToken } = await reauthAccountSecurity({ password });
    // 明文密码仅用于本次请求，获得短时令牌后立即清除。
    form.password = '';
    emit('verified', reauthToken);
  } catch {
    ElMessage.error('身份验证失败，请检查密码后重试');
  } finally {
    submitting.value = false;
  }
}

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) form.password = '';
  },
);
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    width="460px"
    class="security-reauth-dialog"
    :show-close="false"
    :close-on-click-modal="false"
    @update:model-value="close"
  >
    <template #header>
      <div class="security-reauth-dialog__header">
        <span>{{ title }}</span>
        <button type="button" aria-label="关闭身份验证" @click="close"><RiCloseFill /></button>
      </div>
    </template>

    <section class="security-reauth-dialog__content">
      <el-icon class="security-reauth-dialog__hero"><RiShieldKeyholeFill /></el-icon>
      <p>{{ description }}</p>
      <el-form label-position="top" @submit.prevent="verify">
        <el-form-item label="当前密码">
          <el-input
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            show-password
            placeholder="请输入当前密码"
          />
        </el-form-item>
      </el-form>
    </section>

    <template #footer>
      <el-button :disabled="submitting || actionLoading" @click="close">取消</el-button>
      <el-button type="primary" :loading="submitting || actionLoading" @click="verify">
        {{ confirmText }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.security-reauth-dialog) {
  --el-dialog-padding-primary: 0;

  border-radius: 12px;
}

:global(.security-reauth-dialog .el-dialog__body) {
  padding: 0;
}

:global(.security-reauth-dialog .el-dialog__footer) {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 24px 22px;
}

:global(.security-reauth-dialog .el-dialog__footer .el-button) {
  min-width: 72px;
  height: 38px;
  margin: 0;
}

.security-reauth-dialog {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 56px;
    padding: 0 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    color: var(--el-text-color-primary);
    font-size: 18px;
    font-weight: 600;
    line-height: 26px;
  }

  &__header > button {
    display: grid;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: 6px;
    place-items: center;
    background: transparent;
    color: var(--el-text-color-regular);
    cursor: pointer;
    font-size: 22px;
  }

  &__header > button:hover {
    background: var(--el-fill-color-light);
  }

  &__content {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24px;
    color: var(--el-text-color-regular);
  }

  &__content > p,
  &__content > .el-form {
    width: 100%;
  }

  &__content > p {
    margin: 0 0 18px;
    line-height: 22px;
  }

  &__hero {
    margin-bottom: 12px;
    color: var(--el-color-primary);
    font-size: 42px;
  }
}
</style>
